package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/crabcoder/crabcoder/pkg/model"
)

type OllamaProvider struct {
	baseURL string
	model   string
	client  *http.Client
}

func NewOllamaProvider(baseURL, model string) *OllamaProvider {
	return &OllamaProvider{
		baseURL: baseURL,
		model:   model,
		client:  &http.Client{},
	}
}

func (p *OllamaProvider) GetName() string { return p.model }

func (p *OllamaProvider) GetTools() []model.ToolDefinition {
	return []model.ToolDefinition{
		{Name: "read_file", Description: "Read a file from the local filesystem"},
		{Name: "write_file", Description: "Write a file to the local filesystem"},
		{Name: "edit_file", Description: "Perform exact string replacements in an existing file"},
		{Name: "bash", Description: "Execute a shell command"},
	}
}

type ollamaRequest struct {
	Model    string           `json:"model"`
	Messages []ollamaMessage  `json:"messages"`
	Stream   bool             `json:"stream"`
	Tools    []ollamaTool     `json:"tools,omitempty"`
}

type ollamaMessage struct {
	Role       string          `json:"role"`
	Content    string          `json:"content"`
	ToolCalls  []ollamaToolCall `json:"tool_calls,omitempty"`
	ToolCallID string          `json:"tool_call_id,omitempty"`
}

type ollamaToolCall struct {
	ID       string         `json:"id"`
	Type     string         `json:"type"`
	Function ollamaFunction `json:"function"`
}

type ollamaFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type ollamaTool struct {
	Type     string             `json:"type"`
	Function ollamaToolFunction `json:"function"`
}

type ollamaToolFunction struct {
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Parameters  ollamaParams `json:"parameters"`
}

type ollamaParams struct {
	Type       string                  `json:"type"`
	Properties map[string]ollamaProp   `json:"properties"`
	Required   []string                `json:"required,omitempty"`
}

type ollamaProp struct {
	Type        string   `json:"type"`
	Description string   `json:"description,omitempty"`
	Enum        []string `json:"enum,omitempty"`
	Items       *ollamaProp `json:"items,omitempty"`
}

type ollamaResponse struct {
	Message ollamaRespMessage `json:"message"`
	Done    bool              `json:"done"`
}

type ollamaRespMessage struct {
	Role      string          `json:"role"`
	Content   string          `json:"content"`
	ToolCalls []ollamaToolCall `json:"tool_calls"`
}

func (p *OllamaProvider) Chat(ctx context.Context, messages []model.Message, opts *ChatOptions) (*ChatResponse, error) {
	reqBody := p.buildRequest(messages, opts, false)

	body, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/api/chat", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ollama request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ollama HTTP %d: %s", resp.StatusCode, string(body))
	}

	var or ollamaResponse
	if err := json.NewDecoder(resp.Body).Decode(&or); err != nil {
		return nil, fmt.Errorf("decode ollama response: %w", err)
	}

	return p.toResponse(or), nil
}

func (p *OllamaProvider) StreamChat(ctx context.Context, messages []model.Message, opts *ChatOptions) (<-chan ChatChunk, error) {
	reqBody := p.buildRequest(messages, opts, true)

	body, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/api/chat", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}

	ch := make(chan ChatChunk, 64)
	go func() {
		defer resp.Body.Close()
		defer close(ch)
		p.readStream(resp.Body, ch)
	}()

	return ch, nil
}

func (p *OllamaProvider) buildRequest(messages []model.Message, opts *ChatOptions, stream bool) ollamaRequest {
	var ollamaMsgs []ollamaMessage
	for _, msg := range messages {
		om := ollamaMessage{
			Role:       string(msg.Role),
			Content:    msg.Content,
			ToolCallID: msg.ToolCallID,
		}
		ollamaMsgs = append(ollamaMsgs, om)
	}

	var ollamaTools []ollamaTool
	for _, t := range opts.Tools {
		ot := ollamaTool{
			Type: "function",
			Function: ollamaToolFunction{
				Name:        t.Name,
				Description: t.Description,
				Parameters: ollamaParams{
					Type:       t.Parameters.Type,
					Required:   t.Parameters.Required,
					Properties: make(map[string]ollamaProp),
				},
			},
		}
		for k, v := range t.Parameters.Properties {
			op := ollamaProp{
				Type:        v.Type,
				Description: v.Description,
				Enum:        v.Enum,
			}
			if v.Items != nil {
				op.Items = &ollamaProp{Type: v.Items.Type, Enum: v.Items.Enum}
			}
			ot.Function.Parameters.Properties[k] = op
		}
		ollamaTools = append(ollamaTools, ot)
	}

	return ollamaRequest{
		Model:    p.model,
		Messages: ollamaMsgs,
		Stream:   stream,
		Tools:    ollamaTools,
	}
}

func (p *OllamaProvider) toResponse(or ollamaResponse) *ChatResponse {
	resp := &ChatResponse{
		Content: or.Message.Content,
	}
	for _, tc := range or.Message.ToolCalls {
		var args map[string]any
		json.Unmarshal([]byte(tc.Function.Arguments), &args)
		resp.ToolCalls = append(resp.ToolCalls, ToolCall{
			ID:   tc.ID,
			Name: tc.Function.Name,
			Args: args,
		})
	}
	return resp
}

func (p *OllamaProvider) readStream(r io.Reader, ch chan<- ChatChunk) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var chunk struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
			Done bool `json:"done"`
		}
		if err := json.Unmarshal([]byte(line), &chunk); err != nil {
			continue
		}
		if chunk.Done {
			ch <- ChatChunk{Done: true}
			return
		}
		if chunk.Message.Content != "" {
			ch <- ChatChunk{Content: chunk.Message.Content}
		}
	}
}

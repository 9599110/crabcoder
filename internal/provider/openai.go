package provider

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

type OpenAIProvider struct {
	apiKey  string
	baseURL string
	model   string
	client  *http.Client
}

func NewOpenAIProvider(apiKey, baseURL, model string) *OpenAIProvider {
	return &OpenAIProvider{
		apiKey:  apiKey,
		baseURL: baseURL,
		model:   model,
		client:  &http.Client{},
	}
}

// --- Request/Response types ---

type openAIMessage struct {
	Role       string         `json:"role"`
	Content    string         `json:"content,omitempty"`
	ToolCallID string         `json:"tool_call_id,omitempty"`
	ToolCalls  []openAIToolCall `json:"tool_calls,omitempty"`
}

type openAIToolCall struct {
	ID       string          `json:"id"`
	Type     string          `json:"type"`
	Function openAIFunction  `json:"function"`
}

type openAIFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type openAIRequest struct {
	Model    string           `json:"model"`
	Messages []openAIMessage  `json:"messages"`
	Tools    []openAITool     `json:"tools,omitempty"`
	Stream   bool             `json:"stream"`
}

type openAITool struct {
	Type     string             `json:"type"`
	Function openAIToolFunction `json:"function"`
}

type openAIToolFunction struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  openAIParams   `json:"parameters"`
}

type openAIParams struct {
	Type       string                    `json:"type"`
	Properties map[string]openAIProp     `json:"properties"`
	Required   []string                  `json:"required,omitempty"`
}

type openAIProp struct {
	Type        string   `json:"type"`
	Description string   `json:"description,omitempty"`
	Enum        []string `json:"enum,omitempty"`
	Items       *openAIProp `json:"items,omitempty"`
}

type openAIResponse struct {
	Choices []openAIChoice `json:"choices"`
	Error   *openAIError   `json:"error,omitempty"`
}

type openAIChoice struct {
	Message openAIRespMessage `json:"message"`
}

type openAIRespMessage struct {
	Content   string          `json:"content"`
	ToolCalls []openAIToolCall `json:"tool_calls"`
}

type openAIError struct {
	Message string `json:"message"`
}

// --- Chat ---

func (p *OpenAIProvider) Chat(ctx context.Context, messages []model.Message, tools []model.ToolDefinition) (*ChatResponse, error) {
	reqBody := p.buildRequest(messages, tools, false)

	body, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("openai request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("openai HTTP %d: %s", resp.StatusCode, string(body))
	}

	var or openAIResponse
	if err := json.NewDecoder(resp.Body).Decode(&or); err != nil {
		return nil, fmt.Errorf("decode openai response: %w", err)
	}
	if or.Error != nil {
		return nil, fmt.Errorf("openai error: %s", or.Error.Message)
	}

	return p.toResponse(or), nil
}

// --- StreamChat ---

func (p *OpenAIProvider) StreamChat(ctx context.Context, messages []model.Message, tools []model.ToolDefinition) (<-chan StreamChunk, error) {
	reqBody := p.buildRequest(messages, tools, true)

	body, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}

	ch := make(chan StreamChunk, 64)
	go func() {
		defer resp.Body.Close()
		defer close(ch)
		p.readSSE(resp.Body, ch)
	}()

	return ch, nil
}

// --- Helpers ---

func (p *OpenAIProvider) buildRequest(messages []model.Message, tools []model.ToolDefinition, stream bool) openAIRequest {
	var openAIMsgs []openAIMessage
	for _, msg := range messages {
		om := openAIMessage{
			Role:       string(msg.Role),
			Content:    msg.Content,
			ToolCallID: msg.ToolCallID,
		}
		openAIMsgs = append(openAIMsgs, om)
	}

	var openAITools []openAITool
	for _, t := range tools {
		ot := openAITool{
			Type: "function",
			Function: openAIToolFunction{
				Name:        t.Name,
				Description: t.Description,
				Parameters: openAIParams{
					Type:       t.Parameters.Type,
					Required:   t.Parameters.Required,
					Properties: make(map[string]openAIProp),
				},
			},
		}
		for k, v := range t.Parameters.Properties {
			op := openAIProp{
				Type:        v.Type,
				Description: v.Description,
				Enum:        v.Enum,
			}
			if v.Items != nil {
				op.Items = &openAIProp{Type: v.Items.Type, Enum: v.Items.Enum}
			}
			ot.Function.Parameters.Properties[k] = op
		}
		openAITools = append(openAITools, ot)
	}

	return openAIRequest{
		Model:    p.model,
		Messages: openAIMsgs,
		Tools:    openAITools,
		Stream:   stream,
	}
}

func (p *OpenAIProvider) toResponse(or openAIResponse) *ChatResponse {
	resp := &ChatResponse{}
	if len(or.Choices) == 0 {
		return resp
	}
	msg := or.Choices[0].Message
	resp.Content = msg.Content
	for _, tc := range msg.ToolCalls {
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

func (p *OpenAIProvider) readSSE(r io.Reader, ch chan<- StreamChunk) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" || line == "data: [DONE]" {
			continue
		}
		if len(line) < 6 || line[:6] != "data: " {
			continue
		}
		data := line[6:]

		var chunk struct {
			Choices []struct {
				Delta struct {
					Content   string `json:"content"`
					ToolCalls []struct {
						Index    int    `json:"index"`
						ID       string `json:"id"`
						Function struct {
							Name      string `json:"name"`
							Arguments string `json:"arguments"`
						} `json:"function"`
					} `json:"tool_calls"`
				} `json:"delta"`
			} `json:"choices"`
		}
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}
		if len(chunk.Choices) == 0 {
			continue
		}
		delta := chunk.Choices[0].Delta
		if delta.Content != "" {
			ch <- StreamChunk{Content: delta.Content}
		}
		for _, tc := range delta.ToolCalls {
			ch <- StreamChunk{
				ToolCallID:   tc.ID,
				ToolCallName: tc.Function.Name,
				ToolCallArgs: tc.Function.Arguments,
			}
		}
	}

	ch <- StreamChunk{Done: true}
}

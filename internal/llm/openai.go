package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

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
		client:  &http.Client{Timeout: 120 * time.Second},
	}
}

func (p *OpenAIProvider) GetName() string { return p.model }

func (p *OpenAIProvider) GetTools() []model.ToolDefinition {
	return []model.ToolDefinition{
		{Name: "read_file", Description: "Read a file from the local filesystem"},
		{Name: "write_file", Description: "Write a file to the local filesystem"},
		{Name: "edit_file", Description: "Perform exact string replacements in an existing file"},
		{Name: "bash", Description: "Execute a shell command"},
	}
}

// --- Request/Response types ---

type openAIMessage struct {
	Role             string          `json:"role"`
	Content          string          `json:"content"`
	ReasoningContent string          `json:"reasoning_content,omitempty"`
	ToolCallID       string          `json:"tool_call_id,omitempty"`
	ToolCalls        []openAIToolCall `json:"tool_calls,omitempty"`
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
	Model      string           `json:"model"`
	Messages   []openAIMessage  `json:"messages"`
	Tools      []openAITool     `json:"tools,omitempty"`
	Stream     bool             `json:"stream"`
	ToolChoice string           `json:"tool_choice,omitempty"`
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

type openAIUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type openAIResponse struct {
	Choices []openAIChoice `json:"choices"`
	Usage   openAIUsage    `json:"usage,omitempty"`
	Error   *openAIError   `json:"error,omitempty"`
}

type openAIChoice struct {
	Message openAIRespMessage `json:"message"`
}

type openAIRespMessage struct {
	Content          string          `json:"content"`
	ReasoningContent string          `json:"reasoning_content"`
	ToolCalls        []openAIToolCall `json:"tool_calls"`
}

type openAIError struct {
	Message string `json:"message"`
}

// --- Chat ---

func (p *OpenAIProvider) Chat(ctx context.Context, messages []model.Message, opts *ChatOptions) (*ChatResponse, error) {
	reqBody := p.buildRequest(messages, opts, false)

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

func (p *OpenAIProvider) StreamChat(ctx context.Context, messages []model.Message, opts *ChatOptions) (<-chan ChatChunk, error) {
	reqBody := p.buildRequest(messages, opts, true)

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

	ch := make(chan ChatChunk, 64)
	go func() {
		defer resp.Body.Close()
		defer close(ch)
		p.readSSE(resp.Body, ch)
	}()

	return ch, nil
}

// --- Helpers ---

func (p *OpenAIProvider) buildRequest(messages []model.Message, opts *ChatOptions, stream bool) openAIRequest {
	var openAIMsgs []openAIMessage
	for _, msg := range messages {
		om := openAIMessage{
			Role:             string(msg.Role),
			Content:          msg.Content,
			ReasoningContent: msg.Reasoning,
			ToolCallID:       msg.ToolCallID,
		}
		// Serialize tool calls for assistant messages
		if msg.Role == model.RoleAssistant && len(msg.ToolCalls) > 0 {
			for _, tc := range msg.ToolCalls {
				argsJSON, _ := json.Marshal(tc.Args)
				om.ToolCalls = append(om.ToolCalls, openAIToolCall{
					ID:   tc.ID,
					Type: "function",
					Function: openAIFunction{
						Name:      tc.Name,
						Arguments: string(argsJSON),
					},
				})
			}
		}
		openAIMsgs = append(openAIMsgs, om)
	}

	var openAITools []openAITool
	for _, t := range opts.Tools {
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
	resp := &ChatResponse{
		PromptTokens: or.Usage.PromptTokens,
		TotalTokens:  or.Usage.TotalTokens,
	}
	if len(or.Choices) == 0 {
		return resp
	}
	msg := or.Choices[0].Message
	resp.Content = msg.Content
	resp.Reasoning = msg.ReasoningContent
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

func (p *OpenAIProvider) readSSE(r io.Reader, ch chan<- ChatChunk) {
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
			ch <- ChatChunk{Content: delta.Content}
		}
		for _, tc := range delta.ToolCalls {
			ch <- ChatChunk{
				ToolCallID:   tc.ID,
				ToolCallName: tc.Function.Name,
				ToolCallArgs: tc.Function.Arguments,
			}
		}
	}

	ch <- ChatChunk{Done: true}
}

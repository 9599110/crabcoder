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

const anthropicAPIVersion = "2023-06-01"

type AnthropicProvider struct {
	apiKey  string
	baseURL string
	model   string
	client  *http.Client
}

func NewAnthropicProvider(apiKey, baseURL, model string) *AnthropicProvider {
	return &AnthropicProvider{
		apiKey:  apiKey,
		baseURL: baseURL,
		model:   model,
		client:  &http.Client{Timeout: 120 * time.Second},
	}
}

func (p *AnthropicProvider) GetName() string { return p.model }

func (p *AnthropicProvider) GetTools() []model.ToolDefinition {
	return []model.ToolDefinition{
		{Name: "read_file", Description: "Read a file from the local filesystem"},
		{Name: "write_file", Description: "Write a file to the local filesystem"},
		{Name: "edit_file", Description: "Perform exact string replacements in an existing file"},
		{Name: "bash", Description: "Execute a shell command"},
	}
}

// --- Request/Response types ---

type anthropicMessage struct {
	Role    string            `json:"role"`
	Content []anthropicBlock  `json:"content"`
}

type anthropicBlock struct {
	Type       string `json:"type"`
	Text       string `json:"text,omitempty"`
	ToolUseID  string `json:"tool_use_id,omitempty"`
	Content    string `json:"content,omitempty"` // for tool_result
	Name       string `json:"name,omitempty"`
	ID         string `json:"id,omitempty"`
	Input      map[string]any `json:"input,omitempty"`
}

type anthropicRequest struct {
	Model     string              `json:"model"`
	MaxTokens int                 `json:"max_tokens"`
	Messages  []anthropicMessage  `json:"messages"`
	System    string              `json:"system,omitempty"`
	Tools     []anthropicTool     `json:"tools,omitempty"`
	Stream    bool                `json:"stream"`
}

type anthropicTool struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema inputSchema `json:"input_schema"`
}

type inputSchema struct {
	Type       string                  `json:"type"`
	Properties map[string]propDef      `json:"properties"`
	Required   []string                `json:"required,omitempty"`
}

type propDef struct {
	Type        string   `json:"type"`
	Description string   `json:"description,omitempty"`
	Enum        []string `json:"enum,omitempty"`
	Items       *propDef `json:"items,omitempty"`
}

type anthropicResponse struct {
	Content []anthropicResponseBlock `json:"content"`
	Error   *anthropicError          `json:"error,omitempty"`
}

type anthropicResponseBlock struct {
	Type  string         `json:"type"`
	Text  string         `json:"text,omitempty"`
	Name  string         `json:"name,omitempty"`
	ID    string         `json:"id,omitempty"`
	Input map[string]any `json:"input,omitempty"`
}

type anthropicError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

// --- Chat ---

func (p *AnthropicProvider) Chat(ctx context.Context, messages []model.Message, opts *ChatOptions) (*ChatResponse, error) {
	reqBody := p.buildRequest(messages, opts, false)

	body, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/v1/messages", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("x-api-key", p.apiKey)
	req.Header.Set("anthropic-version", anthropicAPIVersion)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("anthropic request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("anthropic HTTP %d: %s", resp.StatusCode, string(body))
	}

	var ar anthropicResponse
	if err := json.NewDecoder(resp.Body).Decode(&ar); err != nil {
		return nil, fmt.Errorf("decode anthropic response: %w", err)
	}
	if ar.Error != nil {
		return nil, fmt.Errorf("anthropic error: %s - %s", ar.Error.Type, ar.Error.Message)
	}

	return p.toResponse(ar), nil
}

// --- StreamChat ---

func (p *AnthropicProvider) StreamChat(ctx context.Context, messages []model.Message, opts *ChatOptions) (<-chan ChatChunk, error) {
	reqBody := p.buildRequest(messages, opts, true)

	body, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/v1/messages", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("x-api-key", p.apiKey)
	req.Header.Set("anthropic-version", anthropicAPIVersion)
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

func (p *AnthropicProvider) buildRequest(messages []model.Message, opts *ChatOptions, stream bool) anthropicRequest {
	var anthropicMsgs []anthropicMessage
	for _, msg := range messages {
		am := anthropicMessage{Role: string(msg.Role)}
		switch msg.Role {
		case model.RoleUser:
			am.Content = []anthropicBlock{{Type: "text", Text: msg.Content}}
		case model.RoleAssistant:
			if len(msg.ToolCalls) > 0 {
				for _, tc := range msg.ToolCalls {
					am.Content = append(am.Content, anthropicBlock{
						Type:  "tool_use",
						ID:    tc.ID,
						Name:  tc.Name,
						Input: tc.Args,
					})
				}
			}
			if msg.Content != "" {
				am.Content = append(am.Content, anthropicBlock{Type: "text", Text: msg.Content})
			}
		case model.RoleTool:
			am.Content = []anthropicBlock{{
				Type:      "tool_result",
				ToolUseID: msg.ToolCallID,
				Content:   msg.Content,
			}}
		case model.RoleSystem:
			// system is handled separately in Anthropic API
			continue
		}
		anthropicMsgs = append(anthropicMsgs, am)
	}

	var anthropicTools []anthropicTool
	for _, t := range opts.Tools {
		at := anthropicTool{
			Name:        t.Name,
			Description: t.Description,
			InputSchema: inputSchema{
				Type:       t.Parameters.Type,
				Properties: make(map[string]propDef),
				Required:   t.Parameters.Required,
			},
		}
		for k, v := range t.Parameters.Properties {
			pd := propDef{
				Type:        v.Type,
				Description: v.Description,
				Enum:        v.Enum,
			}
			if v.Items != nil {
				pd.Items = &propDef{Type: v.Items.Type, Enum: v.Items.Enum}
			}
			at.InputSchema.Properties[k] = pd
		}
		anthropicTools = append(anthropicTools, at)
	}

	req := anthropicRequest{
		Model:     p.model,
		MaxTokens: 32000,
		Messages:  anthropicMsgs,
		Tools:     anthropicTools,
		Stream:    stream,
	}

	// Extract system message
	for _, msg := range messages {
		if msg.Role == model.RoleSystem {
			req.System = msg.Content
			break
		}
	}

	return req
}

func (p *AnthropicProvider) toResponse(ar anthropicResponse) *ChatResponse {
	resp := &ChatResponse{}
	for _, block := range ar.Content {
		switch block.Type {
		case "text":
			resp.Content += block.Text
		case "tool_use":
			resp.ToolCalls = append(resp.ToolCalls, ToolCall{
				ID:   block.ID,
				Name: block.Name,
				Args: block.Input,
			})
		}
	}
	return resp
}

func (p *AnthropicProvider) readSSE(r io.Reader, ch chan<- ChatChunk) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" || line == "event: message_start" || line == "event: ping" {
			continue
		}

		// Anthropic SSE format: "data: {...}"
		if len(line) < 6 || line[:5] != "data:" {
			continue
		}
		data := line[5:]
		if len(data) > 0 && data[0] == ' ' {
			data = data[1:]
		}

		var event map[string]json.RawMessage
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			continue
		}

		eventType := ""
		if v, ok := event["type"]; ok {
			json.Unmarshal(v, &eventType)
		}

		switch eventType {
		case "content_block_delta":
			var delta struct {
				Delta struct {
					Type       string `json:"type"`
					Text       string `json:"text"`
					PartialJSON string `json:"partial_json"`
				} `json:"delta"`
			}
			json.Unmarshal([]byte(data), &delta)
			if delta.Delta.Type == "text_delta" {
				ch <- ChatChunk{Content: delta.Delta.Text}
			} else if delta.Delta.Type == "input_json_delta" {
				ch <- ChatChunk{ToolCallArgs: delta.Delta.PartialJSON}
			}

		case "message_stop":
			ch <- ChatChunk{Done: true}
		}
	}
}

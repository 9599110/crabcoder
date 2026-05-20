package ai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type OpenAIProvider struct {
	config ProviderConfig
	client *http.Client
}

func NewOpenAIProvider(cfg ProviderConfig) *OpenAIProvider {
	return &OpenAIProvider{
		config: cfg,
		client: &http.Client{Timeout: 120 * time.Second},
	}
}

func (p *OpenAIProvider) Name() string { return "openai" }

func (p *OpenAIProvider) ListModels() []string {
	return []string{"gpt-4o", "gpt-4-turbo", "gpt-3.5-turbo"}
}

func (p *OpenAIProvider) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	body := p.buildBody(req, false)
	resp, err := p.doRequest(ctx, body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result openaiResp
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	if len(result.Choices) == 0 {
		return nil, fmt.Errorf("空响应")
	}

	choice := result.Choices[0]
	content := choice.Message.Content
	var toolCalls []ToolCall
	for _, tc := range choice.Message.ToolCalls {
		toolCalls = append(toolCalls, ToolCall{
			ID:   tc.ID,
			Name: tc.Function.Name,
			Args: parseJSONMap(tc.Function.Arguments),
		})
	}

	return &ChatResponse{
		Content:   content,
		Stop:      choice.FinishReason,
		ToolCalls: toolCalls,
		Usage: &Usage{
			InputTokens:  result.Usage.PromptTokens,
			OutputTokens: result.Usage.CompletionTokens,
		},
	}, nil
}

func (p *OpenAIProvider) Stream(ctx context.Context, req *ChatRequest) (<-chan *StreamEvent, error) {
	body := p.buildBody(req, true)
	resp, err := p.doRequest(ctx, body)
	if err != nil {
		return nil, err
	}

	ch := make(chan *StreamEvent, 100)

	go func() {
		defer close(ch)
		defer resp.Body.Close()

		scanner := bufio.NewScanner(io.LimitReader(resp.Body, 1024*1024))
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if !strings.HasPrefix(line, "data: ") {
				continue
			}
			data := strings.TrimPrefix(line, "data: ")
			if data == "[DONE]" {
				ch <- &StreamEvent{Done: true}
				return
			}

			var event openaiStreamChunk
			if err := json.Unmarshal([]byte(data), &event); err != nil {
				continue
			}
			if len(event.Choices) > 0 {
				delta := event.Choices[0].Delta
				if delta.Content != "" {
					ch <- &StreamEvent{Content: delta.Content}
				}
				if event.Choices[0].FinishReason != "" {
					ch <- &StreamEvent{Done: true}
					return
				}
			}
		}
	}()

	return ch, nil
}

func (p *OpenAIProvider) buildBody(req *ChatRequest, stream bool) []byte {
	messages := make([]map[string]any, len(req.Messages))
	for i, msg := range req.Messages {
		messages[i] = map[string]any{"role": msg.Role, "content": msg.Content}
	}

	body := map[string]any{
		"model":    p.config.Model,
		"messages": messages,
		"stream":   stream,
	}
	if req.System != "" {
		// prepend system message for OpenAI
		messages = append([]map[string]any{{"role": "system", "content": req.System}}, messages...)
		body["messages"] = messages
	}
	if req.Options != nil && req.Options.MaxTokens > 0 {
		body["max_tokens"] = req.Options.MaxTokens
	}
	if len(req.Tools) > 0 {
		tools := make([]map[string]any, len(req.Tools))
		for i, t := range req.Tools {
			tools[i] = map[string]any{
				"type": "function",
				"function": map[string]any{
					"name":        t.Name,
					"description": t.Description,
					"parameters":  t.InputSchema,
				},
			}
		}
		body["tools"] = tools
	}

	b, _ := json.Marshal(body)
	return b
}

func (p *OpenAIProvider) doRequest(ctx context.Context, body []byte) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", p.config.BaseURL+"/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+p.config.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		resp.Body.Close()
		return nil, fmt.Errorf("API 错误 %d: %s", resp.StatusCode, string(body))
	}
	return resp, nil
}

func parseJSONMap(s string) map[string]any {
	var m map[string]any
	json.Unmarshal([]byte(s), &m)
	return m
}

type openaiResp struct {
	Choices []struct {
		Message struct {
			Content   string `json:"content"`
			ToolCalls []struct {
				ID       string `json:"id"`
				Function struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				} `json:"function"`
			} `json:"tool_calls"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage"`
}

type openaiStreamChunk struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
}

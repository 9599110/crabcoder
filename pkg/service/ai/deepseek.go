package ai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// DeepSeekProvider 通过 Anthropic 兼容 API 调用 DeepSeek 模型
// 支持思考模式 (thinking/reasoning)
type DeepSeekProvider struct {
	config ProviderConfig
	client *http.Client
}

func NewDeepSeekProvider(cfg ProviderConfig) *DeepSeekProvider {
	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://api.deepseek.com/anthropic"
	}
	return &DeepSeekProvider{
		config: cfg,
		client: &http.Client{Timeout: 600 * time.Second},
	}
}

func (p *DeepSeekProvider) Name() string { return "deepseek" }

func (p *DeepSeekProvider) ListModels() []string {
	return []string{
		"deepseek-chat",      // 通用对话模型
		"deepseek-coder",     // 代码专用模型
		"deepseek-v3",        // V3 版本
		"deepseek-v4-pro",    // V4 Pro 版本
		"deepseek-v4-flash",  // V4 Flash 版本
	}
}

// Chat 非流式对话
func (p *DeepSeekProvider) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	body := p.buildBody(req, false)
	resp, err := p.doRequest(ctx, body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result deepseekResp
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	content, reasoning, toolCalls := p.extractContent(result.Content)

	return &ChatResponse{
		Content:          content,
		ReasoningContent: reasoning,
		Stop:             result.StopReason,
		ToolCalls:        toolCalls,
		Usage: &Usage{
			InputTokens:  result.Usage.InputTokens,
			OutputTokens: result.Usage.OutputTokens,
		},
	}, nil
}

// Stream 流式对话，支持思考模式
func (p *DeepSeekProvider) Stream(ctx context.Context, req *ChatRequest) (<-chan *StreamEvent, error) {
	body := p.buildBody(req, true)
	resp, err := p.doRequest(ctx, body)
	if err != nil {
		return nil, err
	}

	ch := make(chan *StreamEvent, 200)

	go func() {
		defer close(ch)
		defer resp.Body.Close()

		scanner := bufio.NewScanner(io.LimitReader(resp.Body, 2*1024*1024))
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if !strings.HasPrefix(line, "data: ") {
				continue
			}
			data := strings.TrimPrefix(line, "data: ")

			var event deepseekStreamEvent
			if err := json.Unmarshal([]byte(data), &event); err != nil {
				continue
			}

			switch event.Type {
			case "content_block_start":
				// 新的内容块开始
				if event.ContentBlock != nil {
					switch event.ContentBlock.Type {
					case "thinking":
						if event.ContentBlock.Thinking != "" {
							ch <- &StreamEvent{
								Type:             "thinking",
								ReasoningContent: event.ContentBlock.Thinking,
							}
						}
					case "text":
						if event.ContentBlock.Text != "" {
							ch <- &StreamEvent{
								Type:    "content",
								Content: event.ContentBlock.Text,
							}
						}
					}
				}

			case "content_block_delta":
				if event.Delta.Type == "thinking_delta" {
					ch <- &StreamEvent{
						Type:             "thinking",
						ReasoningContent: event.Delta.Thinking,
					}
				} else if event.Delta.Type == "text_delta" {
					ch <- &StreamEvent{
						Type:    "content",
						Content: event.Delta.Text,
					}
				}

			case "message_stop":
				ch <- &StreamEvent{Type: "done", Done: true}
				return

			case "error":
				ch <- &StreamEvent{
					Type:  "error",
					Error: errors.New(event.Error.Message),
					Done:  true,
				}
				return
			}
		}
	}()

	return ch, nil
}

// buildBody 构建请求体（Anthropic 格式）
func (p *DeepSeekProvider) buildBody(req *ChatRequest, stream bool) []byte {
	messages := make([]map[string]any, len(req.Messages))
	for i, msg := range req.Messages {
		content := p.buildMessageContent(msg)
		m := map[string]any{"role": msg.Role, "content": content}

		if msg.ToolCallID != "" {
			m["tool_call_id"] = msg.ToolCallID
		}
		messages[i] = m
	}

	body := map[string]any{
		"model":      p.config.Model,
		"max_tokens": p.config.MaxTokens,
		"messages":   messages,
		"stream":     stream,
	}

	if req.System != "" {
		body["system"] = req.System
	}

	// 思考模式配置
	thinkingEnabled := true
	effort := "high"
	if req.Options != nil {
		thinkingEnabled = req.Options.ThinkingEnabled
		if req.Options.ThinkingEffort != "" {
			effort = normalizeEffort(req.Options.ThinkingEffort)
		}
	}

	if thinkingEnabled {
		body["thinking"] = map[string]any{
			"type": "enabled",
		}
		body["output_config"] = map[string]any{
			"effort": effort,
		}
	}

	if len(req.Tools) > 0 {
		tools := make([]map[string]any, len(req.Tools))
		for i, t := range req.Tools {
			tools[i] = map[string]any{
				"name":         t.Name,
				"description":  t.Description,
				"input_schema": t.InputSchema,
			}
		}
		body["tools"] = tools
	}

	b, _ := json.Marshal(body)
	return b
}

// buildMessageContent 构建消息内容（支持 reasoning_content）
func (p *DeepSeekProvider) buildMessageContent(msg Message) any {
	var blocks []map[string]any

	// 如果有思维链内容且在工具调用场景下，必须回传
	if msg.ReasoningContent != "" {
		blocks = append(blocks, map[string]any{
			"type":     "thinking",
			"thinking": msg.ReasoningContent,
		})
	}

	if msg.Content != "" {
		blocks = append(blocks, map[string]any{
			"type": "text",
			"text": msg.Content,
		})
	}

	if len(msg.ToolCalls) > 0 {
		for _, tc := range msg.ToolCalls {
			blocks = append(blocks, map[string]any{
				"type":  "tool_use",
				"id":    tc.ID,
				"name":  tc.Name,
				"input": tc.Args,
			})
		}
	}

	// 如果只有一个 text 块且无其他内容，简化为字符串
	if len(blocks) == 1 && blocks[0]["type"] == "text" && msg.ReasoningContent == "" && len(msg.ToolCalls) == 0 {
		return msg.Content
	}

	return blocks
}

func (p *DeepSeekProvider) doRequest(ctx context.Context, body []byte) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", p.config.BaseURL+"/v1/messages", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("x-api-key", p.config.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("content-type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	if resp.StatusCode != 200 {
		errBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		resp.Body.Close()
		return nil, fmt.Errorf("API 错误 %d: %s", resp.StatusCode, string(errBody))
	}
	return resp, nil
}

func (p *DeepSeekProvider) extractContent(blocks []deepseekContentBlock) (string, string, []ToolCall) {
	var text, thinking strings.Builder
	var toolCalls []ToolCall

	for _, block := range blocks {
		switch block.Type {
		case "text":
			text.WriteString(block.Text)
		case "thinking":
			thinking.WriteString(block.Thinking)
		case "tool_use":
			toolCalls = append(toolCalls, ToolCall{
				ID:   block.ID,
				Name: block.Name,
				Args: block.Input,
			})
		}
	}
	return text.String(), thinking.String(), toolCalls
}

// normalizeEffort 规范化思考强度参数
// low, medium → high; xhigh → max
func normalizeEffort(effort string) string {
	switch effort {
	case "low", "medium":
		return "high"
	case "xhigh":
		return "max"
	case "high", "max":
		return effort
	default:
		return "high"
	}
}

// --- 响应类型定义 ---

type deepseekContentBlock struct {
	Type     string         `json:"type"`
	Text     string         `json:"text,omitempty"`
	Thinking string         `json:"thinking,omitempty"`
	ID       string         `json:"id,omitempty"`
	Name     string         `json:"name,omitempty"`
	Input    map[string]any `json:"input,omitempty"`
}

type deepseekResp struct {
	ID         string                 `json:"id"`
	Type       string                 `json:"type"`
	Role       string                 `json:"role"`
	Content    []deepseekContentBlock `json:"content"`
	StopReason string                 `json:"stop_reason"`
	Usage      struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

type deepseekStreamEvent struct {
	Type         string                `json:"type"`
	Index        int                   `json:"index"`
	ContentBlock *deepseekContentBlock `json:"content_block,omitempty"`
	Delta        deepseekStreamDelta   `json:"delta"`
	Error        struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}

type deepseekStreamDelta struct {
	Type     string `json:"type"`
	Text     string `json:"text,omitempty"`
	Thinking string `json:"thinking,omitempty"`
}

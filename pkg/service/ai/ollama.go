package ai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type OllamaProvider struct {
	config ProviderConfig
	client *http.Client
}

func NewOllamaProvider(cfg ProviderConfig) *OllamaProvider {
	if cfg.BaseURL == "" {
		cfg.BaseURL = "http://localhost:11434"
	}
	return &OllamaProvider{
		config: cfg,
		client: &http.Client{Timeout: 300 * time.Second},
	}
}

func (p *OllamaProvider) Name() string { return "ollama" }

func (p *OllamaProvider) ListModels() []string {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", p.config.BaseURL+"/api/tags", nil)
	if err != nil {
		return nil
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	var result struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil
	}

	models := make([]string, len(result.Models))
	for i, m := range result.Models {
		models[i] = m.Name
	}
	return models
}

func (p *OllamaProvider) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	body := p.buildBody(req, false)

	resp, err := p.doRequest(ctx, "/api/chat", body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result ollamaResp
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	return &ChatResponse{
		Content: result.Message.Content,
		Usage: &Usage{
			InputTokens:  result.PromptEvalCount,
			OutputTokens: result.EvalCount,
		},
	}, nil
}

func (p *OllamaProvider) Stream(ctx context.Context, req *ChatRequest) (<-chan *StreamEvent, error) {
	body := p.buildBody(req, true)

	resp, err := p.doRequest(ctx, "/api/chat", body)
	if err != nil {
		return nil, err
	}

	ch := make(chan *StreamEvent, 100)

	go func() {
		defer close(ch)
		defer resp.Body.Close()

		scanner := bufio.NewScanner(io.LimitReader(resp.Body, 1024*1024))
		for scanner.Scan() {
			var event ollamaStreamResp
			if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
				continue
			}
			if event.Message.Content != "" {
				ch <- &StreamEvent{Content: event.Message.Content}
			}
			if event.Done {
				ch <- &StreamEvent{Done: true}
				return
			}
		}
	}()

	return ch, nil
}

func (p *OllamaProvider) buildBody(req *ChatRequest, stream bool) []byte {
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
		body["system"] = req.System
	}
	if req.Options != nil {
		if req.Options.MaxTokens > 0 {
			body["max_tokens"] = req.Options.MaxTokens
		}
		if req.Options.Temperature > 0 {
			body["temperature"] = req.Options.Temperature
		}
	}

	b, _ := json.Marshal(body)
	return b
}

func (p *OllamaProvider) doRequest(ctx context.Context, path string, body []byte) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", p.config.BaseURL+path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w (Ollama 是否在运行？)", err)
	}
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		resp.Body.Close()
		return nil, fmt.Errorf("API 错误 %d: %s", resp.StatusCode, string(body))
	}
	return resp, nil
}

type ollamaResp struct {
	Message struct {
		Content string `json:"content"`
	} `json:"message"`
	PromptEvalCount int `json:"prompt_eval_count"`
	EvalCount       int `json:"eval_count"`
}

type ollamaStreamResp struct {
	Message struct {
		Content string `json:"content"`
	} `json:"message"`
	Done bool `json:"done"`
}

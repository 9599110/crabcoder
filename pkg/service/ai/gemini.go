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

type GeminiProvider struct {
	config ProviderConfig
	client *http.Client
}

func NewGeminiProvider(cfg ProviderConfig) *GeminiProvider {
	return &GeminiProvider{
		config: cfg,
		client: &http.Client{Timeout: 120 * time.Second},
	}
}

func (p *GeminiProvider) Name() string { return "gemini" }

func (p *GeminiProvider) ListModels() []string {
	return []string{"gemini-pro", "gemini-2.5-flash", "gemini-2.5-pro"}
}

func (p *GeminiProvider) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	body := p.buildBody(req)
	url := fmt.Sprintf("%s/v1/models/%s:generateContent?key=%s",
		p.config.BaseURL, p.config.Model, p.config.APIKey)

	resp, err := p.doRequest(ctx, url, body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result geminiResp
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	var content strings.Builder
	for _, cand := range result.Candidates {
		for _, part := range cand.Content.Parts {
			if part.Text != "" {
				content.WriteString(part.Text)
			}
		}
	}

	return &ChatResponse{
		Content: content.String(),
		Usage: &Usage{
			InputTokens:  result.UsageMetadata.PromptTokenCount,
			OutputTokens: result.UsageMetadata.CandidatesTokenCount,
		},
	}, nil
}

func (p *GeminiProvider) Stream(ctx context.Context, req *ChatRequest) (<-chan *StreamEvent, error) {
	body := p.buildBody(req)
	url := fmt.Sprintf("%s/v1/models/%s:streamGenerateContent?alt=sse&key=%s",
		p.config.BaseURL, p.config.Model, p.config.APIKey)

	resp, err := p.doRequest(ctx, url, body)
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

			var event geminiStreamResp
			if err := json.Unmarshal([]byte(data), &event); err != nil {
				continue
			}
			for _, cand := range event.Candidates {
				for _, part := range cand.Content.Parts {
					if part.Text != "" {
						ch <- &StreamEvent{Content: part.Text}
					}
				}
			}
		}
		ch <- &StreamEvent{Done: true}
	}()

	return ch, nil
}

func (p *GeminiProvider) buildBody(req *ChatRequest) []byte {
	contents := make([]map[string]any, len(req.Messages))
	for i, msg := range req.Messages {
		role := "user"
		if msg.Role == "assistant" {
			role = "model"
		}
		contents[i] = map[string]any{
			"role":  role,
			"parts": []map[string]any{{"text": msg.Content}},
		}
	}

	body := map[string]any{"contents": contents}
	if req.System != "" {
		body["system_instruction"] = map[string]any{
			"parts": []map[string]any{{"text": req.System}},
		}
	}

	b, _ := json.Marshal(body)
	return b
}

func (p *GeminiProvider) doRequest(ctx context.Context, url string, body []byte) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
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

type geminiResp struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
	UsageMetadata struct {
		PromptTokenCount     int `json:"promptTokenCount"`
		CandidatesTokenCount int `json:"candidatesTokenCount"`
	} `json:"usageMetadata"`
}

type geminiStreamResp struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
}

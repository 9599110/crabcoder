package tools

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"golang.org/x/net/html"

	"github.com/crabcoder/crabcoder/pkg/model"
)

type WebFetchExecutor struct{}

func (e *WebFetchExecutor) Execute(ctx context.Context, args map[string]any) (*model.TaskResult, error) {
	url, _ := args["url"].(string)
	prompt, _ := args["prompt"].(string)
	if url == "" {
		return &model.TaskResult{Success: false, Error: "url is required"}, nil
	}

	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return &model.TaskResult{Success: false, Error: err.Error()}, nil
	}
	req.Header.Set("User-Agent", "CrabCoder/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return &model.TaskResult{Success: false, Error: err.Error()}, nil
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024))
	if err != nil {
		return &model.TaskResult{Success: false, Error: err.Error()}, nil
	}

	text := extractText(string(body))
	_ = prompt

	return &model.TaskResult{
		Success: true,
		Output:  text,
		Metrics: map[string]any{
			"status_code": resp.StatusCode,
			"bytes":       len(body),
		},
	}, nil
}

func extractText(htmlStr string) string {
	doc, err := html.Parse(strings.NewReader(htmlStr))
	if err != nil {
		return htmlStr
	}
	var sb strings.Builder
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.TextNode {
			sb.WriteString(n.Data)
			sb.WriteByte(' ')
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if c.Data != "script" && c.Data != "style" {
				f(c)
			}
		}
	}
	f(doc)
	return strings.TrimSpace(sb.String())
}

func (e *WebFetchExecutor) Validate(args map[string]any) error {
	if url, _ := args["url"].(string); url == "" {
		return fmt.Errorf("url is required")
	}
	return nil
}

func (e *WebFetchExecutor) GetDefinition() model.ToolDefinition {
	return model.ToolDefinition{
		Name:        "web_fetch",
		Description: "Fetch content from a URL and convert HTML to markdown.",
		Parameters: model.ParameterSchema{
			Type: "object",
			Properties: map[string]model.ParameterProperty{
				"url":    {Type: "string", Description: "The URL to fetch content from."},
				"prompt": {Type: "string", Description: "The prompt to run on the fetched content."},
			},
			Required: []string{"url"},
		},
	}
}

func (e *WebFetchExecutor) GetRiskLevel() RiskLevel { return RiskLow }

type WebSearchExecutor struct{}

func (e *WebSearchExecutor) Execute(ctx context.Context, args map[string]any) (*model.TaskResult, error) {
	query, _ := args["query"].(string)
	if query == "" {
		return &model.TaskResult{Success: false, Error: "query is required"}, nil
	}

	client := &http.Client{Timeout: 30 * time.Second}
	searchURL := fmt.Sprintf("https://html.duckduckgo.com/html/?q=%s", strings.ReplaceAll(query, " ", "+"))
	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return &model.TaskResult{Success: false, Error: err.Error()}, nil
	}
	req.Header.Set("User-Agent", "CrabCoder/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return &model.TaskResult{Success: false, Error: err.Error()}, nil
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 512*1024))
	if err != nil {
		return &model.TaskResult{Success: false, Error: err.Error()}, nil
	}

	results := parseDuckDuckGo(string(body))
	return &model.TaskResult{Success: true, Output: results}, nil
}

func parseDuckDuckGo(htmlStr string) string {
	doc, err := html.Parse(strings.NewReader(htmlStr))
	if err != nil {
		return ""
	}
	var results []string
	var crawl func(*html.Node)
	crawl = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			var cls, href, text string
			for _, attr := range n.Attr {
				switch attr.Key {
				case "class":
					cls = attr.Val
				case "href":
					href = attr.Val
				}
			}
			if strings.Contains(cls, "result__a") {
				for c := n.FirstChild; c != nil; c = c.NextSibling {
					if c.Type == html.TextNode {
						text += c.Data
					}
				}
				if href != "" && text != "" {
					results = append(results, fmt.Sprintf("- [%s](%s)", strings.TrimSpace(text), href))
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			crawl(c)
		}
	}
	crawl(doc)
	if len(results) == 0 {
		return "No results found."
	}
	return strings.Join(results, "\n")
}

func (e *WebSearchExecutor) Validate(args map[string]any) error {
	if q, _ := args["query"].(string); q == "" {
		return fmt.Errorf("query is required")
	}
	return nil
}

func (e *WebSearchExecutor) GetDefinition() model.ToolDefinition {
	return model.ToolDefinition{
		Name:        "web_search",
		Description: "Search the web.",
		Parameters: model.ParameterSchema{
			Type: "object",
			Properties: map[string]model.ParameterProperty{
				"query": {Type: "string", Description: "The search query."},
			},
			Required: []string{"query"},
		},
	}
}

func (e *WebSearchExecutor) GetRiskLevel() RiskLevel { return RiskLow }

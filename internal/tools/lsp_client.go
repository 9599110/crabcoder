package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

// lspServer manages a single LSP server process.
type lspServer struct {
	lang     string
	cmd      *exec.Cmd
	stdin    io.WriteCloser
	stdout   io.ReadCloser
	pending  map[any]chan json.RawMessage
	seq      int
	mu       sync.Mutex
	cancel   context.CancelFunc
	diags    map[string]string // uri -> cached diagnostics
	diagsMu  sync.RWMutex
	rootURI  string
	rootPath string
}

var lspServersMu sync.Mutex
var lspServers = map[string]*lspServer{}

func getOrStartLSPServer(ctx context.Context, lang string, filePath string) (*lspServer, error) {
	lspServersMu.Lock()
	defer lspServersMu.Unlock()

	if srv, ok := lspServers[lang]; ok && srv != nil {
		return srv, nil
	}

	var cmd *exec.Cmd
	switch lang {
	case "go":
		cmd = exec.Command("gopls", "-v", "serve", "-rpc.trace")
	default:
		return nil, fmt.Errorf("LSP: unsupported language %q", lang)
	}

	_, cancel := context.WithCancel(context.Background())
	cmd.Env = os.Environ()

	stdin, err := cmd.StdinPipe()
	if err != nil {
		cancel()
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		return nil, err
	}
	cmd.Stderr = nil // suppress gopls debug output

	absPath, _ := filepath.Abs(filePath)
	rootPath := findProjectRoot(absPath)
	rootURI := "file://" + rootPath

	if err := cmd.Start(); err != nil {
		cancel()
		return nil, fmt.Errorf("LSP %s start: %w", lang, err)
	}

	srv := &lspServer{
		lang:     lang,
		cmd:      cmd,
		stdin:    stdin,
		stdout:   stdout,
		pending:  make(map[any]chan json.RawMessage),
		diags:    make(map[string]string),
		cancel:   cancel,
		rootURI:  rootURI,
		rootPath: rootPath,
	}

	lspServers[lang] = srv

	// Read loop
	go srv.readLoop()

	// Initialize
	if err := srv.initialize(ctx, rootURI, rootPath); err != nil {
		cancel()
		delete(lspServers, lang)
		return nil, err
	}

	return srv, nil
}

func findProjectRoot(filePath string) string {
	dir := filepath.Dir(filePath)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return filepath.Dir(filePath)
		}
		dir = parent
	}
}

func (s *lspServer) readLoop() {
	decoder := json.NewDecoder(s.stdout)
	for {
		var raw json.RawMessage
		if err := decoder.Decode(&raw); err != nil {
			return
		}

		// Check if it's a notification (no id) or response (has id)
		var base struct {
			ID     any    `json:"id"`
			Method string `json:"method"`
		}
		if err := json.Unmarshal(raw, &base); err != nil {
			continue
		}

		if base.Method == "textDocument/publishDiagnostics" {
			s.handleDiagnostics(raw)
			continue
		}
		if base.ID == nil {
			continue // skip other notifications
		}

		s.mu.Lock()
		ch, ok := s.pending[base.ID]
		s.mu.Unlock()
		if ok {
			ch <- raw
		}
	}
}

func (s *lspServer) call(ctx context.Context, method string, params any) (json.RawMessage, error) {
	s.mu.Lock()
	s.seq++
	id := s.seq
	ch := make(chan json.RawMessage, 1)
	s.pending[id] = ch
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		delete(s.pending, id)
		s.mu.Unlock()
	}()

	req := struct {
		JSONRPC string `json:"jsonrpc"`
		ID      int    `json:"id"`
		Method  string `json:"method"`
		Params  any    `json:"params,omitempty"`
	}{"2.0", id, method, params}

	data, _ := json.Marshal(req)
	s.mu.Lock()
	_, err := io.Copy(s.stdin, bytes.NewReader(append(data, '\n')))
	s.mu.Unlock()
	if err != nil {
		return nil, err
	}

	select {
	case resp := <-ch:
		var rpcResp struct {
			Error *struct {
				Code    int    `json:"code"`
				Message string `json:"message"`
			} `json:"error"`
			Result json.RawMessage `json:"result"`
		}
		if err := json.Unmarshal(resp, &rpcResp); err != nil {
			return nil, err
		}
		if rpcResp.Error != nil {
			return nil, fmt.Errorf("LSP error %d: %s", rpcResp.Error.Code, rpcResp.Error.Message)
		}
		return rpcResp.Result, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (s *lspServer) notify(method string, params any) {
	req := struct {
		JSONRPC string `json:"jsonrpc"`
		Method  string `json:"method"`
		Params  any    `json:"params,omitempty"`
	}{"2.0", method, params}

	data, _ := json.Marshal(req)
	s.mu.Lock()
	io.Copy(s.stdin, bytes.NewReader(append(data, '\n')))
	s.mu.Unlock()
}

func (s *lspServer) initialize(ctx context.Context, rootURI, rootPath string) error {
	result, err := s.call(ctx, "initialize", map[string]any{
		"processId": os.Getpid(),
		"rootUri":   rootURI,
		"rootPath":  rootPath,
		"capabilities": map[string]any{
			"textDocument": map[string]any{
				"definition":  map[string]any{"dynamicRegistration": true},
				"references":  map[string]any{"dynamicRegistration": true},
				"hover":       map[string]any{"dynamicRegistration": true, "contentFormat": []string{"markdown", "plaintext"}},
				"diagnostics": map[string]any{"dynamicRegistration": true},
			},
			"workspace": map[string]any{
				"symbol": map[string]any{"dynamicRegistration": true},
			},
		},
		"initializationOptions": map[string]any{},
	})
	if err != nil {
		return fmt.Errorf("LSP initialize: %w", err)
	}
	_ = result

	s.notify("initialized", map[string]any{})
	s.notify("textDocument/didOpen", map[string]any{
		"textDocument": map[string]any{
			"uri":        rootURI,
			"languageId": s.lang,
			"version":    1,
			"text":       "",
		},
	})
	return nil
}

func (s *lspServer) handleDiagnostics(raw json.RawMessage) {
	var notif struct {
		Params struct {
			URI         string `json:"uri"`
			Diagnostics []struct {
				Message  string `json:"message"`
				Severity int    `json:"severity"`
				Range    struct {
					Start struct{ Line, Character int } `json:"start"`
					End   struct{ Line, Character int } `json:"end"`
				} `json:"range"`
			} `json:"diagnostics"`
		} `json:"params"`
	}
	if err := json.Unmarshal(raw, &notif); err != nil {
		return
	}

	var buf bytes.Buffer
	for _, d := range notif.Params.Diagnostics {
		sev := "INFO"
		switch d.Severity {
		case 1:
			sev = "ERROR"
		case 2:
			sev = "WARN"
		case 3:
			sev = "INFO"
		case 4:
			sev = "HINT"
		}
		buf.WriteString(fmt.Sprintf("%s:%d:%d: %s [%s]\n",
			notif.Params.URI, d.Range.Start.Line+1, d.Range.Start.Character+1, d.Message, sev))
	}

	s.diagsMu.Lock()
	s.diags[notif.Params.URI] = buf.String()
	s.diagsMu.Unlock()
}

func (s *lspServer) definition(ctx context.Context, uri string, line, character int) (string, error) {
	result, err := s.call(ctx, "textDocument/definition", map[string]any{
		"textDocument": map[string]any{"uri": uri},
		"position":     map[string]any{"line": line - 1, "character": character - 1},
	})
	if err != nil {
		return "", err
	}

	// Parse location or locations
	var locations []struct {
		URI   string `json:"uri"`
		Range struct {
			Start struct{ Line, Character int } `json:"start"`
			End   struct{ Line, Character int } `json:"end"`
		} `json:"range"`
	}
	if err := json.Unmarshal(result, &locations); err != nil {
		// Try single location
		var loc struct {
			URI   string `json:"uri"`
			Range struct {
				Start struct{ Line, Character int } `json:"start"`
				End   struct{ Line, Character int } `json:"end"`
			} `json:"range"`
		}
		if err := json.Unmarshal(result, &loc); err != nil {
			return "No definition found.", nil
		}
		locations = []struct {
			URI   string `json:"uri"`
			Range struct {
				Start struct{ Line, Character int } `json:"start"`
				End   struct{ Line, Character int } `json:"end"`
			} `json:"range"`
		}{loc}
	}

	var out bytes.Buffer
	for _, l := range locations {
		path := uriToPath(l.URI)
		out.WriteString(fmt.Sprintf("%s:%d:%d\n", path, l.Range.Start.Line+1, l.Range.Start.Character+1))
	}
	return out.String(), nil
}

func (s *lspServer) references(ctx context.Context, uri string, line, character int) (string, error) {
	result, err := s.call(ctx, "textDocument/references", map[string]any{
		"textDocument": map[string]any{"uri": uri},
		"position":     map[string]any{"line": line - 1, "character": character - 1},
		"context":      map[string]any{"includeDeclaration": false},
	})
	if err != nil {
		return "", err
	}

	var locations []struct {
		URI   string `json:"uri"`
		Range struct {
			Start struct{ Line, Character int } `json:"start"`
		} `json:"range"`
	}
	if err := json.Unmarshal(result, &locations); err != nil {
		return "No references found.", nil
	}

	var out bytes.Buffer
	for _, l := range locations {
		path := uriToPath(l.URI)
		out.WriteString(fmt.Sprintf("%s:%d:%d\n", path, l.Range.Start.Line+1, l.Range.Start.Character+1))
	}
	return out.String(), nil
}

func (s *lspServer) hover(ctx context.Context, uri string, line, character int) (string, error) {
	result, err := s.call(ctx, "textDocument/hover", map[string]any{
		"textDocument": map[string]any{"uri": uri},
		"position":     map[string]any{"line": line - 1, "character": character - 1},
	})
	if err != nil {
		return "", err
	}

	var hover struct {
		Contents struct {
			Kind  string `json:"kind"`
			Value string `json:"value"`
		} `json:"contents"`
	}
	if err := json.Unmarshal(result, &hover); err != nil {
		// Try markdown content array
		var alt struct {
			Contents []struct {
				Kind  string `json:"kind"`
				Value string `json:"value"`
			} `json:"contents"`
		}
		if err := json.Unmarshal(result, &alt); err != nil {
			return string(result), nil
		}
		for _, c := range alt.Contents {
			if c.Kind == "markdown" || c.Kind == "plaintext" {
				return c.Value, nil
			}
		}
		return fmt.Sprintf("%v", alt.Contents), nil
	}
	return hover.Contents.Value, nil
}

func (s *lspServer) documentSymbols(ctx context.Context, uri string) (string, error) {
	result, err := s.call(ctx, "textDocument/documentSymbol", map[string]any{
		"textDocument": map[string]any{"uri": uri},
	})
	if err != nil {
		return "", err
	}

	var symbols []struct {
		Name           string `json:"name"`
		Kind           int    `json:"kind"`
		Range          struct {
			Start struct{ Line, Character int } `json:"start"`
		} `json:"selectionRange"`
	}
	if err := json.Unmarshal(result, &symbols); err != nil {
		return string(result), nil
	}

	var out bytes.Buffer
	for _, s := range symbols {
		out.WriteString(fmt.Sprintf("%s:%d  %s\n", uri, s.Range.Start.Line+1, s.Name))
	}
	return out.String(), nil
}

func (s *lspServer) workspaceSymbols(ctx context.Context, query string) (string, error) {
	result, err := s.call(ctx, "workspace/symbol", map[string]any{
		"query": query,
	})
	if err != nil {
		return "", err
	}

	var symbols []struct {
		Name       string `json:"name"`
		Kind       int    `json:"kind"`
		Location   struct {
			URI   string `json:"uri"`
			Range struct {
				Start struct{ Line, Character int } `json:"start"`
			} `json:"range"`
		} `json:"location"`
	}
	if err := json.Unmarshal(result, &symbols); err != nil {
		return string(result), nil
	}

	var out bytes.Buffer
	for _, s := range symbols {
		path := uriToPath(s.Location.URI)
		out.WriteString(fmt.Sprintf("%s:%d:%d  %s\n", path, s.Location.Range.Start.Line+1, s.Location.Range.Start.Character+1, s.Name))
	}
	return out.String(), nil
}

var extToLang = map[string]string{
	".rs": "rust", ".ts": "typescript", ".tsx": "typescript",
	".js": "javascript", ".jsx": "javascript", ".py": "python",
	".go": "go", ".java": "java", ".c": "c", ".h": "c",
	".cpp": "cpp", ".hpp": "cpp", ".rb": "ruby", ".lua": "lua", ".zig": "zig",
}

func detectLanguage(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	return extToLang[ext]
}

func uriToPath(uri string) string {
	if len(uri) > 7 && uri[:7] == "file://" {
		return uri[7:]
	}
	return uri
}

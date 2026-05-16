package rpc

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sync"
)

type Handler func(ctx context.Context, req *Request) (*Response, error)

type Server struct {
	mu       sync.RWMutex
	handlers map[string]Handler
}

func NewServer() *Server {
	return &Server{
		handlers: make(map[string]Handler),
	}
}

func (s *Server) Register(method string, handler Handler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.handlers[method] = handler
}

func (s *Server) Handle(ctx context.Context, reader io.Reader, writer io.Writer) error {
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Bytes()

		var req Request
		if err := json.Unmarshal(line, &req); err != nil {
			errResp := NewErrorResponse(nil, -32700, "parse error")
			data, _ := json.Marshal(errResp)
			fmt.Fprintln(writer, string(data))
			continue
		}

		s.mu.RLock()
		handler, ok := s.handlers[req.Method]
		s.mu.RUnlock()

		if !ok {
			errResp := NewErrorResponse(req.ID, -32601, fmt.Sprintf("method %q not found", req.Method))
			data, _ := json.Marshal(errResp)
			fmt.Fprintln(writer, string(data))
			continue
		}

		resp, err := handler(ctx, &req)
		if err != nil {
			resp = NewErrorResponse(req.ID, -32000, err.Error())
		}

		data, _ := json.Marshal(resp)
		fmt.Fprintln(writer, string(data))
	}
	return scanner.Err()
}

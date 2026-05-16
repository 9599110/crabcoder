package rpc

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
)

type Client struct {
	mu       sync.Mutex
	reader   io.Reader
	writer   io.Writer
	pending  map[any]chan *Response
	seq      atomic.Int64
}

func NewClient(reader io.Reader, writer io.Writer) *Client {
	return &Client{
		reader:  reader,
		writer:  writer,
		pending: make(map[any]chan *Response),
	}
}

func (c *Client) Call(ctx context.Context, method string, params any) (*Response, error) {
	id := c.seq.Add(1)
	req, err := NewRequest(id, method, params)
	if err != nil {
		return nil, err
	}

	ch := make(chan *Response, 1)
	c.mu.Lock()
	c.pending[id] = ch
	c.mu.Unlock()

	data, _ := json.Marshal(req)
	fmt.Fprintln(c.writer, string(data))

	select {
	case resp := <-ch:
		return resp, nil
	case <-ctx.Done():
		c.mu.Lock()
		delete(c.pending, id)
		c.mu.Unlock()
		return nil, ctx.Err()
	}
}

func (c *Client) Start(ctx context.Context) {
	scanner := bufio.NewScanner(c.reader)
	for scanner.Scan() {
		line := scanner.Bytes()
		var resp Response
		if err := json.Unmarshal(line, &resp); err != nil {
			continue
		}

		c.mu.Lock()
		ch, ok := c.pending[resp.ID]
		if ok {
			delete(c.pending, resp.ID)
		}
		c.mu.Unlock()

		if ok {
			ch <- &resp
		}
	}
}

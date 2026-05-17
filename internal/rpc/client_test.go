package rpc

import (
	"context"
	"io"
	"testing"
)

func TestNewClient(t *testing.T) {
	client := NewClient(nil, nil)
	if client == nil {
		t.Fatal("expected non-nil client")
	}
	if client.pending == nil {
		t.Error("expected pending map to be initialized")
	}
}

func TestClient_Call_ContextCanceled(t *testing.T) {
	client := NewClient(nil, io.Discard)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := client.Call(ctx, "test", nil)
	if err == nil {
		t.Error("expected context canceled error")
	}
}

func TestClient_Call_ContextTimeout(t *testing.T) {
	client := NewClient(nil, io.Discard)

	ctx, cancel := context.WithTimeout(context.Background(), 1)
	defer cancel()

	_, err := client.Call(ctx, "test", nil)
	if err == nil {
		t.Error("expected timeout error")
	}
}

func TestNewClient_SeqStartsAtOne(t *testing.T) {
	client := NewClient(nil, io.Discard)

	id := client.seq.Add(1)
	if id != 1 {
		t.Errorf("expected first seq=1, got %d", id)
	}
	id = client.seq.Add(1)
	if id != 2 {
		t.Errorf("expected second seq=2, got %d", id)
	}
}

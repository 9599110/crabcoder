package rpc

import (
	"encoding/json"
	"testing"
)

func TestNewRequest(t *testing.T) {
	req, err := NewRequest(1, "test_method", map[string]string{"key": "value"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.JSONRPC != JSONRPCVersion {
		t.Errorf("expected jsonrpc=%q, got %q", JSONRPCVersion, req.JSONRPC)
	}
	if req.ID.(int) != 1 {
		t.Errorf("expected id=1, got %v", req.ID)
	}
	if req.Method != "test_method" {
		t.Errorf("expected method='test_method', got %q", req.Method)
	}

	var params map[string]string
	if err := json.Unmarshal(req.Params, &params); err != nil {
		t.Fatalf("failed to unmarshal params: %v", err)
	}
	if params["key"] != "value" {
		t.Errorf("expected params.key='value', got %q", params["key"])
	}
}

func TestNewRequest_StringID(t *testing.T) {
	req, err := NewRequest("req-1", "method", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.ID != "req-1" {
		t.Errorf("expected id='req-1', got %v", req.ID)
	}
}

func TestNewRequest_NoParams(t *testing.T) {
	req, err := NewRequest(42, "no_params", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(req.Params) != "null" {
		t.Errorf("expected null params, got %s", string(req.Params))
	}
}

func TestNewResponse(t *testing.T) {
	resp, err := NewResponse(1, map[string]int{"count": 5})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.JSONRPC != JSONRPCVersion {
		t.Errorf("expected jsonrpc=%q, got %q", JSONRPCVersion, resp.JSONRPC)
	}
	if resp.Error != nil {
		t.Error("expected no error")
	}

	var result map[string]int
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if result["count"] != 5 {
		t.Errorf("expected count=5, got %d", result["count"])
	}
}

func TestNewResponse_StringResult(t *testing.T) {
	resp, err := NewResponse("id-x", "hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var result string
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if result != "hello" {
		t.Errorf("expected 'hello', got %q", result)
	}
}

func TestNewResponse_NilResult(t *testing.T) {
	resp, err := NewResponse(1, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(resp.Result) != "null" {
		t.Errorf("expected null result, got %s", string(resp.Result))
	}
}

func TestNewErrorResponse(t *testing.T) {
	resp := NewErrorResponse(99, -32600, "Invalid Request")
	if resp.JSONRPC != JSONRPCVersion {
		t.Errorf("expected jsonrpc=%q, got %q", JSONRPCVersion, resp.JSONRPC)
	}
	if resp.Error == nil {
		t.Fatal("expected error to be non-nil")
	}
	if resp.Error.Code != -32600 {
		t.Errorf("expected code=-32600, got %d", resp.Error.Code)
	}
	if resp.Error.Message != "Invalid Request" {
		t.Errorf("expected message='Invalid Request', got %q", resp.Error.Message)
	}
}

func TestNewErrorResponse_NilData(t *testing.T) {
	resp := NewErrorResponse(0, -1, "error")
	if resp.Error.Data != nil {
		t.Errorf("expected nil data, got %v", resp.Error.Data)
	}
}

func TestRequest_MarshalRoundtrip(t *testing.T) {
	req, _ := NewRequest(1, "test", map[string]any{"a": 1})
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var parsed Request
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if parsed.Method != "test" {
		t.Errorf("expected method='test', got %q", parsed.Method)
	}
}

func TestResponse_MarshalRoundtrip(t *testing.T) {
	resp, _ := NewResponse(1, "ok")
	data, _ := json.Marshal(resp)

	var parsed Response
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	var result string
	json.Unmarshal(parsed.Result, &result)
	if result != "ok" {
		t.Errorf("expected 'ok', got %q", result)
	}
}

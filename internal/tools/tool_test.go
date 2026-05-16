package tools

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestReadFile(t *testing.T) {
	exec := &ReadFileExecutor{}

	// Create temp file
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(tmpFile, []byte("line1\nline2\nline3\nline4\nline5"), 0644); err != nil {
		t.Fatal(err)
	}

	// Read entire file
	result, err := exec.Execute(context.Background(), map[string]any{"path": tmpFile})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Success {
		t.Fatalf("expected success: %s", result.Error)
	}
	if result.Output != "line1\nline2\nline3\nline4\nline5" {
		t.Fatalf("unexpected output: %q", result.Output)
	}

	// Read with offset and limit
	result, err = exec.Execute(context.Background(), map[string]any{
		"path":   tmpFile,
		"offset": float64(1),
		"limit":  float64(2),
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Output != "line2\nline3" {
		t.Fatalf("unexpected output: %q", result.Output)
	}

	// Read nonexistent file
	result, err = exec.Execute(context.Background(), map[string]any{"path": "/nonexistent/file.txt"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Success {
		t.Fatal("expected failure for nonexistent file")
	}
}

func TestWriteFile(t *testing.T) {
	exec := &WriteFileExecutor{}

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "output.txt")

	result, err := exec.Execute(context.Background(), map[string]any{
		"path":    tmpFile,
		"content": "hello world",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Success {
		t.Fatalf("expected success: %s", result.Error)
	}

	data, _ := os.ReadFile(tmpFile)
	if string(data) != "hello world" {
		t.Fatalf("expected 'hello world', got %q", string(data))
	}
}

func TestEditFile(t *testing.T) {
	exec := &EditFileExecutor{}

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "edit.txt")
	os.WriteFile(tmpFile, []byte("Hello Alice!"), 0644)

	result, err := exec.Execute(context.Background(), map[string]any{
		"path":       tmpFile,
		"old_string": "Alice",
		"new_string": "Bob",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Success {
		t.Fatalf("expected success: %s", result.Error)
	}

	data, _ := os.ReadFile(tmpFile)
	if string(data) != "Hello Bob!" {
		t.Fatalf("expected 'Hello Bob!', got %q", string(data))
	}
}

func TestEditFileDuplicate(t *testing.T) {
	exec := &EditFileExecutor{}

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "dup.txt")
	os.WriteFile(tmpFile, []byte("hello hello"), 0644)

	result, err := exec.Execute(context.Background(), map[string]any{
		"path":       tmpFile,
		"old_string": "hello",
		"new_string": "world",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Success {
		t.Fatal("expected failure for duplicate match without replace_all")
	}
}

func TestEditFileReplaceAll(t *testing.T) {
	exec := &EditFileExecutor{}

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "dup.txt")
	os.WriteFile(tmpFile, []byte("hello hello"), 0644)

	result, err := exec.Execute(context.Background(), map[string]any{
		"path":        tmpFile,
		"old_string":  "hello",
		"new_string":  "world",
		"replace_all": true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Success {
		t.Fatalf("expected success: %s", result.Error)
	}

	data, _ := os.ReadFile(tmpFile)
	if string(data) != "world world" {
		t.Fatalf("expected 'world world', got %q", string(data))
	}
}

func TestShellExecute(t *testing.T) {
	exec := &ShellExecutor{DefaultTimeout: 5e9} // 5s

	result, err := exec.Execute(context.Background(), map[string]any{
		"command": "echo hello",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Success {
		t.Fatalf("expected success: %s", result.Error)
	}
	if result.Output != "hello" {
		t.Fatalf("expected 'hello', got %q", result.Output)
	}
}

func TestToolRegistry(t *testing.T) {
	reg := NewToolRegistry()

	exec := &ReadFileExecutor{}
	if err := reg.Register(exec.GetDefinition().Name, exec); err != nil {
		t.Fatal(err)
	}

	// Duplicate registration
	if err := reg.Register(exec.GetDefinition().Name, exec); err == nil {
		t.Fatal("expected duplicate registration error")
	}

	// Get
	got := reg.Get("read_file")
	if got == nil {
		t.Fatal("expected to find read_file")
	}

	// Get nonexistent
	got = reg.Get("nonexistent")
	if got != nil {
		t.Fatal("expected not found")
	}

	// List
	execs := reg.List()
	if len(execs) != 1 {
		t.Fatalf("expected 1 executor, got %d", len(execs))
	}

	// Definitions
	defs := reg.Definitions()
	if len(defs) != 1 {
		t.Fatalf("expected 1 definition, got %d", len(defs))
	}
	if defs[0].Name != "read_file" {
		t.Fatalf("expected 'read_file', got %q", defs[0].Name)
	}
}

package code

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExtractFunction(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "main.go")
	content := `package main

func main() {
	x := 1
	y := 2
	z := x + y
	println(z)
}
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := ExtractFunction(path, "compute", 4, 6)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "func compute()") {
		t.Errorf("expected 'func compute()', got: %s", result)
	}
	if !strings.Contains(result, "x := 1") {
		t.Errorf("expected body content, got: %s", result)
	}
}

func TestExtractFunction_InvalidRange(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "main.go")
	os.WriteFile(path, []byte("package main\nfunc main() {}\n"), 0644)

	_, err := ExtractFunction(path, "bad", 10, 5)
	if err == nil {
		t.Error("expected error for invalid range")
	}
}

func TestExtractFunction_BeyondFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "main.go")
	os.WriteFile(path, []byte("package main\nfunc main() {}\n"), 0644)

	_, err := ExtractFunction(path, "bad", 1, 999)
	if err == nil {
		t.Error("expected error for range beyond file")
	}
}

func TestExtractFunction_NonExistent(t *testing.T) {
	_, err := ExtractFunction("/nonexistent/main.go", "f", 1, 2)
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestDetectIndent(t *testing.T) {
	if got := detectIndent("\tline1\n\tline2"); got != "\t" {
		t.Errorf("expected tab, got %q", got)
	}
	if got := detectIndent(""); got != "\t" {
		t.Errorf("expected default tab, got %q", got)
	}
}

func TestIndentBody(t *testing.T) {
	input := "line1\nline2\n"
	result := indentBody(input, "\t")
	lines := strings.Split(strings.TrimSuffix(result, "\n"), "\n")
	for _, line := range lines {
		if !strings.HasPrefix(line, "\t") {
			t.Errorf("expected tab prefix on line %q", line)
		}
	}
}

func TestFormatGoFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "main.go")
	// Intentionally bad formatting
	content := "package main\nfunc main(){x:=1\nprintln(x)}"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	if err := FormatGoFile(path); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	formatted := string(data)

	if !strings.Contains(formatted, "x := 1") {
		t.Errorf("expected formatted code, got: %s", formatted)
	}
}

func TestFormatGoFile_Invalid(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.go")
	os.WriteFile(path, []byte("not valid go {{{"), 0644)

	err := FormatGoFile(path)
	if err == nil {
		t.Error("expected error for invalid Go file")
	}
}

func TestFormatGoFile_NonExistent(t *testing.T) {
	err := FormatGoFile("/nonexistent/main.go")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestFindDefinition(t *testing.T) {
	dir := t.TempDir()
	content := `package main

type Config struct {
	Port int
}

func NewConfig() *Config { return &Config{Port: 8080} }
`
	os.WriteFile(filepath.Join(dir, "main.go"), []byte(content), 0644)

	symbol, err := FindDefinition(dir, "Config")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if symbol == nil {
		t.Fatal("expected symbol, got nil")
	}
	if symbol.Kind != "type" {
		t.Errorf("expected type kind, got %q", symbol.Kind)
	}

	_, err = FindDefinition(dir, "NonExistent")
	if err == nil {
		t.Error("expected error for non-existent symbol")
	}
}

func TestFindReferences(t *testing.T) {
	dir := t.TempDir()
	content := `package main

var count int

func inc() {
	count++
}

func dec() {
	count--
}

func check() int {
	return count
}
`
	os.WriteFile(filepath.Join(dir, "main.go"), []byte(content), 0644)

	refs, err := FindReferences(dir, "count")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(refs) < 2 {
		t.Errorf("expected at least 2 references, got %d", len(refs))
	}
	for _, r := range refs {
		if r.Name != "count" {
			t.Errorf("expected ref name 'count', got %q", r.Name)
		}
		if r.Kind != "ref" {
			t.Errorf("expected ref kind, got %q", r.Kind)
		}
	}
}

func TestFindReferences_NotFound(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\nfunc main() {}\n"), 0644)

	refs, err := FindReferences(dir, "nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(refs) != 0 {
		t.Errorf("expected 0 refs, got %d", len(refs))
	}
}

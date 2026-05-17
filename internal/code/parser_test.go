package code

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectLang(t *testing.T) {
	tests := []struct {
		path string
		lang Lang
	}{
		{"main.go", LangGo},
		{"script.py", LangPython},
		{"lib.rs", LangRust},
		{"app.js", LangJS},
		{"app.jsx", LangJS},
		{"app.mjs", LangJS},
		{"app.ts", LangTS},
		{"app.tsx", LangTS},
		{"Main.java", LangJava},
		{"README.md", LangUnknown},
	}

	for _, tc := range tests {
		if got := detectLang(tc.path); got != tc.lang {
			t.Errorf("detectLang(%q) = %q, want %q", tc.path, got, tc.lang)
		}
	}
}

func TestParseFile_Go(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "main.go")
	content := `package main

import "fmt"

type Config struct {
	Port int
}

var defaultPort = 8080

const maxRetries = 3

func (c *Config) Start() error {
	fmt.Println("start")
	return nil
}

func main() {
	cfg := Config{Port: 8080}
	cfg.Start()
}
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := ParseFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Symbols) == 0 {
		t.Fatal("expected symbols, got none")
	}

	// Check for expected symbols
	symbolNames := make(map[string]Symbol)
	for _, s := range result.Symbols {
		symbolNames[s.Name] = s
	}

	if _, ok := symbolNames["Config"]; !ok {
		t.Error("expected Config type")
	}
	if _, ok := symbolNames["defaultPort"]; !ok {
		t.Error("expected defaultPort var")
	}
	if _, ok := symbolNames["maxRetries"]; !ok {
		t.Error("expected maxRetries const")
	}
	if _, ok := symbolNames["Start"]; !ok {
		t.Error("expected Start method")
	}
	if _, ok := symbolNames["main"]; !ok {
		t.Error("expected main func")
	}

	// Verify Start is a method with receiver
	if s := symbolNames["Start"]; s.Kind != "method" {
		t.Errorf("Start should be method, got kind=%q", s.Kind)
	}
}

func TestParseFile_Python(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "script.py")
	content := `
def hello(name):
    return f"Hello {name}"

class MyClass:
    def method(self):
        pass
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := ParseFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	names := make(map[string]string)
	for _, s := range result.Symbols {
		names[s.Name] = s.Kind
	}
	if names["hello"] != "func" {
		t.Errorf("expected hello func, got %q", names["hello"])
	}
	if names["MyClass"] != "class" {
		t.Errorf("expected MyClass class, got %q", names["MyClass"])
	}
}

func TestParseFile_Rust(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "lib.rs")
	content := `
pub fn add(a: i32, b: i32) -> i32 {
    a + b
}

fn private_helper() {}

pub(crate) fn internal_fn() {}
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := ParseFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	names := make(map[string]string)
	for _, s := range result.Symbols {
		names[s.Name] = s.Kind
	}
	if names["add"] != "func" {
		t.Errorf("expected add func, got %q", names["add"])
	}
	if names["private_helper"] != "func" {
		t.Errorf("expected private_helper func, got %q", names["private_helper"])
	}
	if names["internal_fn"] != "func" {
		t.Errorf("expected internal_fn func, got %q", names["internal_fn"])
	}
}

func TestParseFile_JavaScript(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.js")
	content := `
function greet() { return "hi"; }

const arrow = () => 1;

class Animal {
    speak() {}
}
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := ParseFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	names := make(map[string]string)
	for _, s := range result.Symbols {
		names[s.Name] = s.Kind
	}
	if names["greet"] != "func" {
		t.Errorf("expected greet func, got %q", names["greet"])
	}
	if names["Animal"] != "class" {
		t.Errorf("expected Animal class, got %q", names["Animal"])
	}
}

func TestParseFile_TypeScript(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.ts")
	content := `
function foo(): string { return "bar"; }

class MyComponent {
    render() {}
}
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := ParseFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	names := make(map[string]string)
	for _, s := range result.Symbols {
		names[s.Name] = s.Kind
	}
	if names["foo"] != "func" {
		t.Errorf("expected foo func, got %q", names["foo"])
	}
	if names["MyComponent"] != "class" {
		t.Errorf("expected MyComponent class, got %q", names["MyComponent"])
	}
}

func TestParseFile_Java(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "Main.java")
	content := `
public class Main {
    public static void main(String[] args) {
        System.out.println("hello");
    }

    private int getValue() {
        return 42;
    }
}
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := ParseFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Symbols) == 0 {
		t.Error("expected at least one method")
	}
	for _, s := range result.Symbols {
		if s.Kind != "method" {
			t.Errorf("expected method kind, got %q for %s", s.Kind, s.Name)
		}
	}
}

func TestParseFile_Unknown(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "README.md")
	if err := os.WriteFile(path, []byte("# Hello"), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := ParseFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Symbols) != 0 {
		t.Errorf("expected 0 symbols for unknown lang, got %d", len(result.Symbols))
	}
}

func TestParseFile_NonExistent(t *testing.T) {
	_, err := ParseFile("/nonexistent/file.go")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestParseDir(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\nfunc main() {}\n"), 0644)
	os.WriteFile(filepath.Join(dir, "util.py"), []byte("def helper():\n    pass\n"), 0644)
	os.WriteFile(filepath.Join(dir, "README.md"), []byte("# docs"), 0644)

	results, err := ParseDir(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) < 2 {
		t.Errorf("expected at least 2 parsed files, got %d", len(results))
	}
}

func TestTypeString(t *testing.T) {
	// Test helper via end-to-end ParseFile
	dir := t.TempDir()
	path := filepath.Join(dir, "types.go")
	content := `package main

type Handler func(req Request) Response

type Server struct {
	handler *Handler
}

func (s *Server) Listen(addr string) {}
func NewServer() *Server { return nil }
`
	os.WriteFile(path, []byte(content), 0644)

	result, err := ParseFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, s := range result.Symbols {
		switch s.Name {
		case "Listen":
			if s.Sig == "" {
				t.Error("expected sig for Listen")
			}
		}
	}
}

func TestSignatureString(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sig.go")
	content := `package main

func Greet(name string, count int) string { return "" }
func NoParams() {}
`
	os.WriteFile(path, []byte(content), 0644)

	result, err := ParseFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, s := range result.Symbols {
		if s.Name == "Greet" && s.Sig == "" {
			t.Error("expected non-empty sig for Greet")
		}
	}
}

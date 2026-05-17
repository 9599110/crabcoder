package code

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Symbol represents a code symbol (function, type, variable, etc.).
type Symbol struct {
	Name     string `json:"name"`
	Kind     string `json:"kind"` // func, type, var, const, class, method
	File     string `json:"file"`
	Line     int    `json:"line"`
	Column   int    `json:"column"`
	Package  string `json:"package,omitempty"`
	Receiver string `json:"receiver,omitempty"` // for methods
	Sig      string `json:"sig,omitempty"`       // signature snippet
}

// ParseResult holds extracted symbols and any errors.
type ParseResult struct {
	Symbols []Symbol `json:"symbols"`
	Errors  []string `json:"errors,omitempty"`
}

// Lang classifies a file by extension.
type Lang string

const (
	LangGo     Lang = "go"
	LangPython Lang = "python"
	LangRust   Lang = "rust"
	LangJS     Lang = "javascript"
	LangTS     Lang = "typescript"
	LangJava   Lang = "java"
	LangUnknown Lang = ""
)

func detectLang(path string) Lang {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".go":
		return LangGo
	case ".py":
		return LangPython
	case ".rs":
		return LangRust
	case ".js", ".jsx", ".mjs":
		return LangJS
	case ".ts", ".tsx":
		return LangTS
	case ".java":
		return LangJava
	default:
		return LangUnknown
	}
}

// ParseFile extracts symbols from a source file.
func ParseFile(path string) (*ParseResult, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}
	src := string(data)
	lang := detectLang(path)

	switch lang {
	case LangGo:
		return parseGo(path, src)
	default:
		return parseRegex(lang, path, src), nil
	}
}

func parseGo(path, src string) (*ParseResult, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, src, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("parse Go: %w", err)
	}

	var symbols []Symbol
	pkgName := f.Name.Name

	for _, decl := range f.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			pos := fset.Position(d.Pos())
			s := Symbol{
				Name:    d.Name.Name,
				Kind:    "func",
				File:    path,
				Line:    pos.Line,
				Column:  pos.Column,
				Package: pkgName,
			}
			if d.Recv != nil && len(d.Recv.List) > 0 {
				s.Kind = "method"
				s.Receiver = typeString(d.Recv.List[0].Type)
			}
			if d.Type != nil {
				s.Sig = signatureString(d.Type)
			}
			symbols = append(symbols, s)

		case *ast.GenDecl:
			pos := fset.Position(d.Pos())
			switch d.Tok {
			case token.TYPE:
				for _, spec := range d.Specs {
					if ts, ok := spec.(*ast.TypeSpec); ok {
						symbols = append(symbols, Symbol{
							Name:    ts.Name.Name,
							Kind:    "type",
							File:    path,
							Line:    pos.Line,
							Column:  pos.Column,
							Package: pkgName,
						})
					}
				}
			case token.VAR, token.CONST:
				for _, spec := range d.Specs {
					if vs, ok := spec.(*ast.ValueSpec); ok {
						for _, name := range vs.Names {
							kind := "var"
							if d.Tok == token.CONST {
								kind = "const"
							}
							symbols = append(symbols, Symbol{
								Name:    name.Name,
								Kind:    kind,
								File:    path,
								Line:    pos.Line,
								Column:  pos.Column,
								Package: pkgName,
							})
						}
					}
				}
			}
		}
	}
	return &ParseResult{Symbols: symbols}, nil
}

func typeString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + typeString(t.X)
	default:
		return fmt.Sprintf("%T", expr)
	}
}

func signatureString(ft *ast.FuncType) string {
	if ft.Params == nil {
		return "()"
	}
	var params []string
	for _, p := range ft.Params.List {
		for _, name := range p.Names {
			params = append(params, name.Name+" "+typeString(p.Type))
		}
	}
	r := "(" + strings.Join(params, ", ") + ")"
	if ft.Results != nil && len(ft.Results.List) > 0 {
		r += " "
		if len(ft.Results.List) == 1 && len(ft.Results.List[0].Names) == 0 {
			r += typeString(ft.Results.List[0].Type)
		}
	}
	return r
}

// Regex-based parser for non-Go languages.
var (
	rePyFunc  = regexp.MustCompile(`(?m)^\s*def\s+(\w+)\s*\(([^)]*)\)\s*:`)
	rePyClass = regexp.MustCompile(`(?m)^\s*class\s+(\w+)\s*[:\(]`)
	reRustFn  = regexp.MustCompile(`(?m)^\s*(?:pub(?:\s*\(\s*crate\s*\))?\s+)?fn\s+(\w+)\s*[<(]`)
	reJSTSFn  = regexp.MustCompile(`(?m)(?:function\s+(\w+)|(\w+)\s*=\s*(?:async\s+)?function|\b(\w+)\s*=\s*(?:async\s+)?\([^)]*\)\s*=>)`)
	reJSClass = regexp.MustCompile(`(?m)class\s+(\w+)`)
	reJavaFn  = regexp.MustCompile(`(?m)^\s*(?:public|private|protected|static|\s)+[\w<>]+\s+(\w+)\s*\(`)
)

func parseRegex(lang Lang, path, src string) *ParseResult {
	var symbols []Symbol
	lines := strings.Split(src, "\n")

	add := func(name, kind string, line int) {
		symbols = append(symbols, Symbol{
			Name: name,
			Kind: kind,
			File: path,
			Line: line,
		})
	}

	switch lang {
	case LangPython:
		for i, line := range lines {
			if m := rePyFunc.FindStringSubmatch(line); m != nil {
				add(m[1], "func", i+1)
			}
			if m := rePyClass.FindStringSubmatch(line); m != nil {
				add(m[1], "class", i+1)
			}
		}

	case LangRust:
		for i, line := range lines {
			if m := reRustFn.FindStringSubmatch(line); m != nil {
				add(m[1], "func", i+1)
			}
		}

	case LangJS, LangTS:
		for i, line := range lines {
			if m := reJSClass.FindStringSubmatch(line); m != nil {
				add(m[1], "class", i+1)
				continue
			}
			if m := reJSTSFn.FindStringSubmatch(line); m != nil {
				name := m[1]
				if name == "" {
					name = m[2]
				}
				if name == "" {
					name = m[3]
				}
				if name != "" {
					add(name, "func", i+1)
				}
			}
		}

	case LangJava:
		for i, line := range lines {
			if m := reJavaFn.FindStringSubmatch(line); m != nil {
				add(m[1], "method", i+1)
			}
		}
	}

	return &ParseResult{Symbols: symbols}
}

// ParseDir extracts symbols from all source files in a directory.
func ParseDir(dir string) (map[string]*ParseResult, error) {
	results := make(map[string]*ParseResult)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		path := filepath.Join(dir, e.Name())
		lang := detectLang(path)
		if lang == LangUnknown {
			// Skip unknown/irrelevant files
			ext := filepath.Ext(path)
			if ext != ".go" && ext != ".py" && ext != ".rs" && ext != ".js" && ext != ".ts" && ext != ".jsx" && ext != ".tsx" && ext != ".java" && ext != ".rb" && ext != ".lua" && ext != ".zig" {
				continue
			}
		}
		r, err := ParseFile(path)
		if err != nil {
			results[path] = &ParseResult{Errors: []string{err.Error()}}
			continue
		}
		results[path] = r
	}
	return results, nil
}

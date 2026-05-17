package code

import (
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"strings"
)

// RenameResult describes the outcome of a symbol rename.
type RenameResult struct {
	File  string `json:"file"`
	Line  int    `json:"line"`
	Old   string `json:"old"`
	New   string `json:"new"`
	Count int    `json:"count"` // occurrences replaced in this file
}

// RenameSymbol renames a Go identifier across files in dir.
func RenameSymbol(dir, oldName, newName string) ([]RenameResult, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read dir: %w", err)
	}

	var results []RenameResult
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") {
			continue
		}
		path := dir + "/" + e.Name()
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, path, data, parser.ParseComments)
		if err != nil {
			continue
		}

		// Find all occurrences of oldName
		var occurrences []token.Pos
		ast.Inspect(f, func(n ast.Node) bool {
			if ident, ok := n.(*ast.Ident); ok && ident.Name == oldName {
				occurrences = append(occurrences, ident.Pos())
			}
			return true
		})
		if len(occurrences) == 0 {
			continue
		}

		// Build replacement: replace from end to start to preserve positions
		lines := strings.Split(string(data), "\n")
		for _, pos := range occurrences {
			p := fset.Position(pos)
			if p.Line > 0 && p.Line <= len(lines) {
				line := lines[p.Line-1]
				col := p.Column - 1
				if col >= 0 && col+len(oldName) <= len(line) && line[col:col+len(oldName)] == oldName {
					lines[p.Line-1] = line[:col] + newName + line[col+len(oldName):]
				}
			}
		}

		newSrc := strings.Join(lines, "\n")
		if err := os.WriteFile(path, []byte(newSrc), 0644); err != nil {
			return results, fmt.Errorf("write %s: %w", path, err)
		}

		results = append(results, RenameResult{
			File:  path,
			Line:  fset.Position(occurrences[0]).Line,
			Old:   oldName,
			New:   newName,
			Count: len(occurrences),
		})
	}
	return results, nil
}

// FormatGoFile runs gofmt on a Go source file.
func FormatGoFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, data, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("parse: %w", err)
	}
	var buf strings.Builder
	if err := format.Node(&buf, fset, f); err != nil {
		return fmt.Errorf("format: %w", err)
	}
	return os.WriteFile(path, []byte(buf.String()), 0644)
}

// ExtractFunction extracts a function body from lines [startLine, endLine] into a new function.
func ExtractFunction(path, funcName string, startLine, endLine int) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	lines := strings.Split(string(data), "\n")
	if startLine < 1 || endLine > len(lines) || startLine > endLine {
		return "", fmt.Errorf("invalid line range: %d-%d (file has %d lines)", startLine, endLine, len(lines))
	}

	body := strings.Join(lines[startLine-1:endLine], "\n")
	ind := detectIndent(body)
	newFunc := fmt.Sprintf("func %s() {\n%s\n}", funcName, indentBody(body, ind))

	return newFunc, nil
}

func detectIndent(s string) string {
	lines := strings.Split(s, "\n")
	for _, line := range lines {
		if len(line) > 0 && (line[0] == '\t' || line[0] == ' ') {
			return "\t"
		}
	}
	return "\t"
}

func indentBody(s, indent string) string {
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		if line != "" {
			lines[i] = indent + line
		}
	}
	return strings.Join(lines, "\n")
}

// FindDefinition searches for a Go symbol definition across a directory.
func FindDefinition(dir, symbolName string) (*Symbol, error) {
	results, err := ParseDir(dir)
	if err != nil {
		return nil, err
	}
	for _, r := range results {
		for _, s := range r.Symbols {
			if s.Name == symbolName {
				return &s, nil
			}
		}
	}
	return nil, fmt.Errorf("symbol %q not found", symbolName)
}

// FindReferences locates all usages of a Go identifier across files.
func FindReferences(dir, symbolName string) ([]Symbol, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read dir: %w", err)
	}

	var refs []Symbol
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") {
			continue
		}
		path := dir + "/" + e.Name()
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, path, data, parser.ParseComments)
		if err != nil {
			continue
		}

		ast.Inspect(f, func(n ast.Node) bool {
			if ident, ok := n.(*ast.Ident); ok && ident.Name == symbolName {
				pos := fset.Position(ident.Pos())
				refs = append(refs, Symbol{
					Name:   symbolName,
					Kind:   "ref",
					File:   path,
					Line:   pos.Line,
					Column: pos.Column,
				})
			}
			return true
		})
	}
	return refs, nil
}

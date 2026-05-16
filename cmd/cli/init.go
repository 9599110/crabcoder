package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// initProject generates project context files similar to crab init.
// Creates .crabcoder/CONTEXT.md (compressed project summary), crab.json,
// and updates .gitignore. Skips files that already exist.
func initProject() string {
	cwd, _ := os.Getwd()
	contextPath := filepath.Join(cwd, ".crabcoder", "CONTEXT.md")

	if _, err := os.Stat(contextPath); err == nil {
		return "skipped: .crabcoder/CONTEXT.md already exists"
	}

	// Ensure .crabcoder/ directory
	os.MkdirAll(filepath.Join(cwd, ".crabcoder"), 0700)

	// Detect project
	lang, framework, buildCmd, testCmd, dirs := detectProject(cwd)

	// Generate CONTEXT.md
	md := renderContextMD(lang, framework, buildCmd, testCmd, dirs, cwd)
	os.WriteFile(contextPath, []byte(md), 0600)

	// Create crab.json if missing
	crabJSONPath := filepath.Join(cwd, "crab.json")
	if _, err := os.Stat(crabJSONPath); os.IsNotExist(err) {
		os.WriteFile(crabJSONPath, []byte(`{
  "permissions": {
    "defaultMode": "dontAsk"
  }
}
`), 0600)
	}

	// Ensure .gitignore entries
	ensureGitignore(cwd)

	return fmt.Sprintf("created: .crabcoder/CONTEXT.md, crab.json, .gitignore")
}

func detectProject(root string) (lang, framework, buildCmd, testCmd string, dirs []string) {
	// Scan for language markers
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err == nil {
		lang = "Go"
		buildCmd = "go build ./..."
		testCmd = "go test ./..."
		// Read module name from go.mod
		if data, err := os.ReadFile(filepath.Join(root, "go.mod")); err == nil {
			for _, line := range strings.Split(string(data), "\n") {
				if strings.HasPrefix(line, "module ") {
					framework = strings.TrimSpace(strings.TrimPrefix(line, "module "))
					break
				}
			}
		}
	} else if _, err := os.Stat(filepath.Join(root, "Cargo.toml")); err == nil {
		lang = "Rust"
		buildCmd = "cargo build"
		testCmd = "cargo test --workspace"
	} else if _, err := os.Stat(filepath.Join(root, "package.json")); err == nil {
		lang = "TypeScript/JavaScript"
		buildCmd = "npm run build"
		testCmd = "npm test"
		if data, err := os.ReadFile(filepath.Join(root, "package.json")); err == nil {
			detectJSFramework(string(data), &framework)
		}
	} else if _, err := os.Stat(filepath.Join(root, "pyproject.toml")); err == nil {
		lang = "Python"
		buildCmd = "pip install -e ."
		testCmd = "pytest"
	}

	// Scan directory structure
	entries, err := os.ReadDir(root)
	if err != nil {
		return
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		switch name {
		case "cmd", "internal", "pkg", "src", "tests", "rust", "lib", "api":
			dirs = append(dirs, name)
		}
	}
	return
}

func detectJSFramework(packageJSON string, framework *string) {
	markers := []string{"next", "react", "vue", "svelte", "nest", "vite", "express"}
	for _, m := range markers {
		if strings.Contains(packageJSON, `"`+m+`"`) {
			if *framework != "" {
				*framework += ", "
			}
			*framework += m
		}
	}
}

func renderContextMD(lang, framework, buildCmd, testCmd string, dirs []string, root string) string {
	projectName := filepath.Base(root)
	var b strings.Builder

	b.WriteString("# " + projectName + "\n\n")

	if lang != "" {
		b.WriteString("## Stack\n")
		b.WriteString("- Language: " + lang)
		if framework != "" {
			b.WriteString(" (" + framework + ")")
		}
		b.WriteString(".\n")
	}

	if buildCmd != "" || testCmd != "" {
		b.WriteString("## Build & test\n")
		if buildCmd != "" {
			b.WriteString("- `" + buildCmd + "`\n")
		}
		if testCmd != "" {
			b.WriteString("- `" + testCmd + "`\n")
		}
	}

	if len(dirs) > 0 {
		b.WriteString("\n## Key directories\n")
		for _, d := range dirs {
			b.WriteString("- `" + d + "/`\n")
		}
	}

	b.WriteString("\n## Conventions\n")
	b.WriteString("- Edit existing files over creating new ones.\n")
	b.WriteString("- No comments unless WHY is non-obvious.\n")

	return b.String()
}

func ensureGitignore(root string) {
	entries := []string{
		"# CrabCoder local artifacts",
		".crabcoder/settings.local.json",
		".crabcoder/sessions/",
	}

	gitignorePath := filepath.Join(root, ".gitignore")
	existing, err := os.ReadFile(gitignorePath)
	if err != nil {
		// Create new
		content := strings.Join(entries, "\n") + "\n"
		os.WriteFile(gitignorePath, []byte(content), 0644)
		return
	}

	// Append missing entries
	content := string(existing)
	var toAdd []string
	for _, e := range entries {
		if !strings.Contains(content, e) {
			toAdd = append(toAdd, e)
		}
	}
	if len(toAdd) > 0 {
		if !strings.HasSuffix(content, "\n") {
			content += "\n"
		}
		content += "\n" + strings.Join(toAdd, "\n") + "\n"
		os.WriteFile(gitignorePath, []byte(content), 0644)
	}
}

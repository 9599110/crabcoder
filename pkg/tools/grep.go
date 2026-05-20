package tools

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func grepSearch(root, pattern string) ([]string, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}

	var results []string

	err = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			if strings.HasPrefix(info.Name(), ".") && info.Name() != "." {
				return filepath.SkipDir
			}
			return nil
		}
		if !isTextFile(info.Name()) {
			return nil
		}
		file, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		lineNum := 0
		for scanner.Scan() {
			lineNum++
			line := scanner.Text()
			if re.MatchString(line) {
				relPath, _ := filepath.Rel(root, path)
				results = append(results, relPath+":"+itoa(lineNum)+":"+line)
			}
		}
		return nil
	})

	return results, err
}

func isTextFile(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	textExts := map[string]bool{
		".go": true, ".py": true, ".js": true, ".ts": true,
		".tsx": true, ".jsx": true, ".rs": true, ".java": true,
		".c": true, ".cpp": true, ".h": true, ".hpp": true,
		".rb": true, ".php": true, ".swift": true, ".kt": true,
		".yaml": true, ".yml": true, ".json": true, ".xml": true,
		".md": true, ".txt": true, ".toml": true, ".cfg": true,
		".sh": true, ".bash": true, ".zsh": true,
		".sql": true, ".proto": true, ".css": true, ".html": true,
		".gitignore": true, ".dockerfile": true, ".makefile": true,
	}
	return textExts[ext] || ext == ""
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

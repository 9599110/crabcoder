package tools

import (
	"os"
	"path/filepath"
	"strings"
)

func globSearch(root, pattern string) ([]string, error) {
	var matches []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() && strings.HasPrefix(info.Name(), ".") && info.Name() != "." {
			return filepath.SkipDir
		}
		relPath, err := filepath.Rel(root, path)
		if err != nil {
			return nil
		}
		matched, err := filepath.Match(pattern, relPath)
		if err != nil {
			return nil
		}
		if matched {
			matches = append(matches, relPath)
		}
		if info.IsDir() && containsDoubleStar(pattern) {
			for _, seg := range strings.Split(relPath, string(filepath.Separator)) {
				subPattern := strings.Replace(pattern, "**", seg+"/**", 1)
				if m, _ := filepath.Match(subPattern, relPath); m {
					return nil
				}
			}
		}
		return nil
	})
	return matches, err
}

func containsDoubleStar(pattern string) bool {
	return strings.Contains(pattern, "**")
}

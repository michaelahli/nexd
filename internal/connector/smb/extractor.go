package smb

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var supportedExtensions = map[string]struct{}{
	".txt":  {},
	".md":   {},
	".json": {},
	".yaml": {},
	".yml":  {},
	".csv":  {},
	".log":  {},
}

// ExtractText reads plain-text content from supported file types.
func ExtractText(path string) (string, error) {
	ext := strings.ToLower(filepath.Ext(path))
	if _, ok := supportedExtensions[ext]; !ok {
		return "", fmt.Errorf("unsupported file type %q", ext)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

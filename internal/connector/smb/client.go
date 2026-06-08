package smb

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/michaelahli/nexd/internal/connector"
)

const Type = "smb"

// Config is the SMB connector-specific configuration.
type Config struct {
	RootPath          string
	IncludeExtensions []string
}

// ParseConfig converts generic connector settings into SMB config.
func ParseConfig(cfg connector.Config) (Config, error) {
	rootPath, _ := cfg.Settings["root_path"].(string)
	if strings.TrimSpace(rootPath) == "" {
		return Config{}, fmt.Errorf("smb connector root_path is required")
	}

	parsed := Config{RootPath: filepath.Clean(rootPath)}
	if raw, ok := cfg.Settings["include_extensions"].([]any); ok {
		for _, value := range raw {
			if ext, ok := value.(string); ok && ext != "" {
				parsed.IncludeExtensions = append(parsed.IncludeExtensions, normalizeExtension(ext))
			}
		}
	}
	return parsed, nil
}

func normalizeExtension(ext string) string {
	ext = strings.TrimSpace(strings.ToLower(ext))
	if ext == "" {
		return ""
	}
	if !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}
	return ext
}

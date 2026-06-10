package lark

import (
	"fmt"
	"strings"

	"github.com/michaelahli/nexd/internal/connector"
)

const Type = "lark"

// Config is the Lark connector-specific configuration.
type Config struct {
	AppID       string
	AppSecret   string
	BaseURL     string
	FolderToken string
}

// ParseConfig converts generic connector settings into Lark config.
func ParseConfig(cfg connector.Config) (Config, error) {
	appID, _ := cfg.Settings["app_id"].(string)
	appSecret, _ := cfg.Settings["app_secret"].(string)

	if strings.TrimSpace(appID) == "" {
		return Config{}, fmt.Errorf("lark connector app_id is required")
	}
	if strings.TrimSpace(appSecret) == "" {
		return Config{}, fmt.Errorf("lark connector app_secret is required")
	}

	baseURL, _ := cfg.Settings["base_url"].(string)
	if strings.TrimSpace(baseURL) == "" {
		baseURL = "https://open.larksuite.com"
	}
	folderToken, _ := cfg.Settings["folder_token"].(string)

	return Config{
		AppID:       strings.TrimSpace(appID),
		AppSecret:   strings.TrimSpace(appSecret),
		BaseURL:     strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		FolderToken: strings.TrimSpace(folderToken),
	}, nil
}

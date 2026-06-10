package gdrive

import (
	"fmt"
	"strings"

	"github.com/michaelahli/nexd/internal/connector"
)

const Type = "gdrive"

// Config is the Google Drive connector-specific configuration.
type Config struct {
	ServiceAccountJSON string
	DriveFolderID      string
	AccessToken        string
}

// ParseConfig converts generic connector settings into Google Drive config.
func ParseConfig(cfg connector.Config) (Config, error) {
	serviceAccountJSON, _ := cfg.Settings["service_account_json"].(string)
	if strings.TrimSpace(serviceAccountJSON) == "" {
		return Config{}, fmt.Errorf("gdrive connector service_account_json is required")
	}

	driveFolderID, _ := cfg.Settings["drive_folder_id"].(string)
	accessToken, _ := cfg.Settings["access_token"].(string)
	// drive_folder_id is optional; if empty, list from root

	return Config{
		ServiceAccountJSON: strings.TrimSpace(serviceAccountJSON),
		DriveFolderID:      strings.TrimSpace(driveFolderID),
		AccessToken:        strings.TrimSpace(accessToken),
	}, nil
}

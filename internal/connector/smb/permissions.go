package smb

import (
	"github.com/google/uuid"
	"github.com/michaelahli/nexd/internal/connector"
)

// PermissionTargets returns stub permission mappings from connector settings.
func PermissionTargets(cfg connector.Config) []connector.PermissionTarget {
	raw, ok := cfg.Settings["read_user_ids"].([]any)
	if !ok {
		return nil
	}
	targets := make([]connector.PermissionTarget, 0, len(raw))
	for _, value := range raw {
		idText, ok := value.(string)
		if !ok {
			continue
		}
		userID, err := uuid.Parse(idText)
		if err != nil {
			continue
		}
		targets = append(targets, connector.PermissionTarget{UserID: &userID, PermissionType: "read"})
	}
	return targets
}

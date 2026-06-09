package gdrive

import "github.com/michaelahli/nexd/internal/connector"

// Register adds the Google Drive connector factory to a registry.
func Register(registry *connector.Registry) error {
	return registry.Register(Type, New)
}

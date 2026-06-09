package lark

import "github.com/michaelahli/nexd/internal/connector"

// Register adds the Lark connector factory to a registry.
func Register(registry *connector.Registry) error {
	return registry.Register(Type, New)
}

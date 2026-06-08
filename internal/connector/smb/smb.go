package smb

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/michaelahli/nexd/internal/connector"
)

// Connector implements the connector contract for filesystem-backed SMB-style sources.
type Connector struct {
	config    Config
	rawConfig connector.Config
	started   bool
}

// New creates an SMB connector from generic configuration.
func New(cfg connector.Config) (connector.Connector, error) {
	parsed, err := ParseConfig(cfg)
	if err != nil {
		return nil, err
	}
	return &Connector{config: parsed, rawConfig: cfg}, nil
}

// Name returns the connector name.
func (c *Connector) Name() string {
	return c.rawConfig.Name
}

// Type returns the connector type.
func (c *Connector) Type() string {
	return Type
}

// Validate checks SMB connector configuration.
func (c *Connector) Validate(ctx context.Context, cfg connector.Config) error {
	_, err := ParseConfig(cfg)
	return err
}

// Start prepares the connector.
func (c *Connector) Start(ctx context.Context, cfg connector.Config) error {
	parsed, err := ParseConfig(cfg)
	if err != nil {
		return err
	}
	c.config = parsed
	c.rawConfig = cfg
	c.started = true
	return nil
}

// Stop releases connector resources.
func (c *Connector) Stop(ctx context.Context) error {
	c.started = false
	return nil
}

// Health verifies that the root path is accessible.
func (c *Connector) Health(ctx context.Context) error {
	if !c.started {
		return fmt.Errorf("connector is not started")
	}
	info, err := os.Stat(c.config.RootPath)
	if err != nil {
		return fmt.Errorf("cannot access root path: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("root path %q is not a directory", c.config.RootPath)
	}
	return nil
}

// FullSync walks all files under root path.
func (c *Connector) FullSync(ctx context.Context) (<-chan connector.Document, <-chan error) {
	return c.sync(ctx, time.Time{})
}

// IncrementalSync walks files modified after the given timestamp.
func (c *Connector) IncrementalSync(ctx context.Context, since time.Time) (<-chan connector.Document, <-chan error) {
	return c.sync(ctx, since)
}

func (c *Connector) sync(ctx context.Context, since time.Time) (<-chan connector.Document, <-chan error) {
	docs := make(chan connector.Document)
	errs := make(chan error, 1)

	go func() {
		defer close(docs)
		defer close(errs)

		permissions := PermissionTargets(c.rawConfig)
		err := filepath.WalkDir(c.config.RootPath, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			if !c.shouldIndex(path) {
				return nil
			}

			info, err := d.Info()
			if err != nil {
				return err
			}
			if !since.IsZero() && !info.ModTime().After(since) {
				return nil
			}

			content, err := ExtractText(path)
			if err != nil {
				return nil // skip unsupported files
			}

			relPath, _ := filepath.Rel(c.config.RootPath, path)
			modifiedAt := info.ModTime()
			doc := connector.Document{
				SourceType:      Type,
				SourceID:        relPath,
				Title:           filepath.Base(path),
				Content:         content,
				FilePath:        path,
				FileSize:        info.Size(),
				MIMEType:        mimeTypeFromPath(path),
				SourceUpdatedAt: &modifiedAt,
				Permissions:     permissions,
			}

			select {
			case <-ctx.Done():
				return ctx.Err()
			case docs <- doc:
			}
			return nil
		})
		if err != nil {
			errs <- err
		}
	}()
	return docs, errs
}

func (c *Connector) shouldIndex(path string) bool {
	if len(c.config.IncludeExtensions) == 0 {
		return true
	}
	ext := normalizeExtension(filepath.Ext(path))
	for _, allowed := range c.config.IncludeExtensions {
		if ext == allowed {
			return true
		}
	}
	return false
}

func mimeTypeFromPath(path string) string {
	ext := filepath.Ext(path)
	switch ext {
	case ".txt", ".log":
		return "text/plain"
	case ".md":
		return "text/markdown"
	case ".json":
		return "application/json"
	case ".yaml", ".yml":
		return "application/yaml"
	case ".csv":
		return "text/csv"
	default:
		return "application/octet-stream"
	}
}

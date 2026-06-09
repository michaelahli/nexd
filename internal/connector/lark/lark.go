package lark

import (
	"context"
	"fmt"
	"time"

	"github.com/michaelahli/nexd/internal/connector"
)

// Connector implements the connector contract for Lark.
type Connector struct {
	config    Config
	rawConfig connector.Config
	client    *Client
	started   bool
}

// New creates a Lark connector from generic configuration.
func New(cfg connector.Config) (connector.Connector, error) {
	parsed, err := ParseConfig(cfg)
	if err != nil {
		return nil, err
	}
	return &Connector{config: parsed, rawConfig: cfg, client: NewClient(parsed)}, nil
}

// Name returns the connector name.
func (c *Connector) Name() string {
	return c.rawConfig.Name
}

// Type returns the connector type.
func (c *Connector) Type() string {
	return Type
}

// Validate checks Lark connector configuration.
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
	c.client = NewClient(parsed)
	c.started = true
	return nil
}

// Stop releases connector resources.
func (c *Connector) Stop(ctx context.Context) error {
	c.started = false
	return nil
}

// Health verifies that the Lark API is accessible.
func (c *Connector) Health(ctx context.Context) error {
	if !c.started {
		return fmt.Errorf("connector is not started")
	}
	if c.client == nil {
		return fmt.Errorf("client is not configured")
	}
	return c.client.Auth(ctx)
}

// FullSync fetches all documents from Lark.
func (c *Connector) FullSync(ctx context.Context) (<-chan connector.Document, <-chan error) {
	return c.sync(ctx, time.Time{})
}

// IncrementalSync fetches documents modified after the given timestamp.
func (c *Connector) IncrementalSync(ctx context.Context, since time.Time) (<-chan connector.Document, <-chan error) {
	return c.sync(ctx, since)
}

func (c *Connector) sync(ctx context.Context, since time.Time) (<-chan connector.Document, <-chan error) {
	docs := make(chan connector.Document)
	errs := make(chan error, 1)

	go func() {
		defer close(docs)
		defer close(errs)

		if c.client == nil {
			errs <- fmt.Errorf("client is not configured")
			return
		}

		larkDocs, err := c.client.ListDocs(ctx)
		if err != nil {
			errs <- err
			return
		}

		permissions := permissionTargets(c.rawConfig)
		for _, larkDoc := range larkDocs {
			if !since.IsZero() && !larkDoc.UpdatedAt.After(since) {
				continue
			}

			doc := connector.Document{
				SourceType:      Type,
				SourceID:        larkDoc.Token,
				Title:           larkDoc.Title,
				Content:         fmt.Sprintf("Lark document: %s (type: %s)", larkDoc.Title, larkDoc.Type),
				MIMEType:        "text/plain",
				SourceUpdatedAt: &larkDoc.UpdatedAt,
				Permissions:     permissions,
			}

			select {
			case <-ctx.Done():
				errs <- ctx.Err()
				return
			case docs <- doc:
			}
		}
	}()

	return docs, errs
}

func permissionTargets(cfg connector.Config) []connector.PermissionTarget {
	// Stub: real implementation would parse Lark doc permissions or use app-level settings.
	return nil
}

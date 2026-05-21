package neo4j

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"

	"github.com/neverthenetwork/inventa/internal/config"
	"github.com/neverthenetwork/inventa/internal/datastore"
)

// Plugin implements discovery.Plugin for Neo4j graph database sources.
//
// It connects to a Neo4j instance via the bolt protocol, runs a Cypher
// query to fetch topology nodes and relationships, transforms them into
// cytoscape.js Elements, and pushes the result into the shared TopologyStore.
//
// When PollIntervalSeconds > 0, it polls on that interval. When 0, it
// performs a single one-shot sync and returns.
type Plugin struct {
	cfg    *config.Neo4jSourceConfig
	logger *slog.Logger
	driver neo4j.DriverWithContext
}

// New creates a new Neo4j plugin. It validates the config but does NOT
// connect to Neo4j — the connection is established in Start().
func New(cfg *config.Neo4jSourceConfig, logger *slog.Logger) (*Plugin, error) {
	if cfg.URI == "" {
		return nil, fmt.Errorf("neo4j plugin: uri is required")
	}
	return &Plugin{
		cfg:    cfg,
		logger: logger.With("source", "neo4j"),
	}, nil
}

// Name returns the source identifier.
func (p *Plugin) Name() string { return "neo4j" }

// Start connects to Neo4j and begins syncing topology data.
//
// If PollIntervalSeconds > 0, it runs on a loop until the context is
// cancelled. If 0 or negative, it performs a single sync and returns.
func (p *Plugin) Start(ctx context.Context, store *datastore.TopologyStore) error {
	// Connect to Neo4j.
	driver, err := newDriver(p.cfg)
	if err != nil {
		return fmt.Errorf("neo4j plugin: creating driver: %w", err)
	}
	p.driver = driver
	defer func() {
		if closeErr := driver.Close(ctx); closeErr != nil {
			p.logger.Warn("error closing neo4j driver", "error", closeErr)
		}
	}()

	// Verify connectivity.
	if err := driver.VerifyConnectivity(ctx); err != nil {
		return fmt.Errorf("neo4j plugin: connectivity check: %w", err)
	}
	p.logger.Info("connected to neo4j", "uri", p.cfg.URI)

	// One-shot or polling.
	if p.cfg.PollIntervalSeconds <= 0 {
		return p.syncOnce(ctx, store)
	}
	return p.syncLoop(ctx, store)
}

// Stop closes the Neo4j driver. It is a no-op if Start has not been called.
func (p *Plugin) Stop() error {
	if p.driver == nil {
		return nil
	}
	// Use a short background context for cleanup.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return p.driver.Close(ctx)
}

// syncOnce performs a single sync and returns.
func (p *Plugin) syncOnce(ctx context.Context, store *datastore.TopologyStore) error {
	return p.sync(ctx, store)
}

// syncLoop polls Neo4j on the configured interval until the context is
// cancelled.
func (p *Plugin) syncLoop(ctx context.Context, store *datastore.TopologyStore) error {
	ticker := time.NewTicker(time.Duration(p.cfg.PollIntervalSeconds) * time.Second)
	defer ticker.Stop()

	// Immediate first sync.
	if err := p.sync(ctx, store); err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := p.sync(ctx, store); err != nil {
				p.logger.Error("neo4j sync failed", "error", err)
				// Continue polling — a single failure shouldn't stop the plugin.
			}
		}
	}
}

// sync runs the Cypher query, transforms the result, and pushes it into
// the store.
func (p *Plugin) sync(ctx context.Context, store *datastore.TopologyStore) error {
	query := p.cfg.Query
	if query == "" {
		query = DefaultCypherQuery()
	}

	p.logger.Debug("running cypher query", "query", query)

	result, err := neo4j.ExecuteQuery[*neo4j.EagerResult](
		ctx,
		p.driver,
		query,
		nil,
		neo4j.EagerResultTransformer,
	)
	if err != nil {
		return fmt.Errorf("executing cypher query: %w", err)
	}

	elements, err := Transform(result, p.Name())
	if err != nil {
		return fmt.Errorf("transforming neo4j result: %w", err)
	}

	store.Set(*elements)

	p.logger.Info("neo4j sync complete",
		"nodes", len(elements.Nodes),
		"edges", len(elements.Edges),
	)

	return nil
}

// newDriver creates a Neo4j driver from config.
func newDriver(cfg *config.Neo4jSourceConfig) (neo4j.DriverWithContext, error) {
	var auth neo4j.AuthToken
	if cfg.Username != "" || cfg.Password != "" {
		auth = neo4j.BasicAuth(cfg.Username, cfg.Password, "")
	} else {
		auth = neo4j.NoAuth()
	}

	return neo4j.NewDriverWithContext(cfg.URI, auth)
}

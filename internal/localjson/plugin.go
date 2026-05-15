// Package localjson provides a discovery plugin that loads a static
// topology from a local JSON file in cytoscape.js Elements format.
package localjson

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"

	"github.com/neverthenetwork/inventa/internal/config"
	"github.com/neverthenetwork/inventa/internal/datastore"

	cy "gonum.org/v1/gonum/graph/formats/cytoscapejs"
)

// Plugin implements discovery.Plugin for static JSON topology files.
type Plugin struct {
	cfg      *config.Conf
	logger   *slog.Logger
	filePath string
}

// New creates a new local JSON plugin.
func New(cfg *config.Conf, logger *slog.Logger) *Plugin {
	return &Plugin{
		cfg:      cfg,
		logger:   logger,
		filePath: cfg.LocalJSONFile,
	}
}

// Name returns the source identifier.
func (p *Plugin) Name() string { return "localjson" }

// Start loads the JSON file into the store and returns immediately.
// It's a one-shot load — no streaming or polling.
func (p *Plugin) Start(_ context.Context, store *datastore.TopologyStore) error {
	if p.filePath == "" {
		return fmt.Errorf("localjson plugin: LocalJSONFile not configured")
	}

	content, err := os.ReadFile(p.filePath)
	if err != nil {
		return fmt.Errorf("reading JSON file %s: %w", p.filePath, err)
	}

	var elements cy.Elements
	if err := json.Unmarshal(content, &elements); err != nil {
		return fmt.Errorf("unmarshaling JSON: %w", err)
	}

	store.Set(elements)
	p.logger.Info("loaded static topology", "file", p.filePath, "nodes", len(elements.Nodes), "edges", len(elements.Edges))

	return nil
}

// Stop is a no-op for the local JSON plugin.
func (p *Plugin) Stop() error { return nil }

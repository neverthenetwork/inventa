// Package discovery defines the plugin interface for topology data sources.
// Each source (BGP-LS, AWS, local JSON, etc.) implements this interface
// and can run concurrently, feeding into a shared TopologyStore.
package discovery

import (
	"context"

	"github.com/neverthenetwork/inventa/internal/datastore"
)

// Plugin is the interface that all topology discovery sources must implement.
type Plugin interface {
	// Name returns a unique identifier for this source (e.g. "bgpls", "aws", "localjson").
	Name() string

	// Start begins discovery and streams/polls topology data into the provided store.
	// This is a blocking call — the caller should run it in a goroutine.
	// It should return when the context is cancelled or an unrecoverable error occurs.
	Start(ctx context.Context, store *datastore.TopologyStore) error

	// Stop gracefully shuts down the discovery source (close connections, cancel subscriptions).
	// It should return quickly and not block indefinitely.
	Stop() error
}

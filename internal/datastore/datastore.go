package datastore

import (
	"sync"

	cy "gonum.org/v1/gonum/graph/formats/cytoscapejs"
)

// TopologyStore holds the graph elements with thread-safe access.
// Written by the BGP watch goroutine, read by HTTP handlers.
type TopologyStore struct {
	mu       sync.RWMutex
	elements cy.Elements
}

// NewTopologyStore creates an empty TopologyStore.
func NewTopologyStore() *TopologyStore {
	return &TopologyStore{
		elements: cy.Elements{
			Nodes: make([]cy.Node, 0),
			Edges: make([]cy.Edge, 0),
		},
	}
}

// Get returns a copy of the current elements (safe for concurrent reads).
func (ts *TopologyStore) Get() cy.Elements {
	ts.mu.RLock()
	defer ts.mu.RUnlock()
	return ts.elements
}

// Set replaces the elements atomically.
func (ts *TopologyStore) Set(elements cy.Elements) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	ts.elements = elements
}

// GetNodeName returns the label for a node ID, or "" if not found.
func (ts *TopologyStore) GetNodeName(nodeID string) string {
	ts.mu.RLock()
	defer ts.mu.RUnlock()
	for _, n := range ts.elements.Nodes {
		if n.Data.ID == nodeID {
			return n.Data.Attributes["label"].(string)
		}
	}
	return ""
}

// NodeCount returns the number of nodes.
func (ts *TopologyStore) NodeCount() int {
	ts.mu.RLock()
	defer ts.mu.RUnlock()
	return len(ts.elements.Nodes)
}

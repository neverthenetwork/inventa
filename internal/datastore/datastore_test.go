package datastore

import (
	"sync"
	"testing"

	cy "gonum.org/v1/gonum/graph/formats/cytoscapejs"
)

func TestNewTopologyStore(t *testing.T) {
	ts := NewTopologyStore()
	if ts == nil {
		t.Fatal("NewTopologyStore() returned nil")
	}
	e := ts.Get()
	if e.Nodes == nil {
		t.Error("expected Nodes to be non-nil")
	}
	if e.Edges == nil {
		t.Error("expected Edges to be non-nil")
	}
	if ts.NodeCount() != 0 {
		t.Errorf("expected 0 nodes, got %d", ts.NodeCount())
	}
}

func TestTopologyStoreSetGet(t *testing.T) {
	ts := NewTopologyStore()
	elements := cy.Elements{
		Nodes: []cy.Node{
			{
				Data: cy.NodeData{
					ID: "node1",
					Attributes: map[string]interface{}{
						"label": "Node 1",
					},
				},
				Selectable: true,
			},
		},
		Edges: []cy.Edge{
			{
				Data: cy.EdgeData{
					ID:     "edge1",
					Source: "node1",
					Target: "node2",
				},
				Selectable: true,
			},
		},
	}

	ts.Set(elements)
	got := ts.Get()

	if len(got.Nodes) != 1 {
		t.Errorf("expected 1 node, got %d", len(got.Nodes))
	}
	if got.Nodes[0].Data.ID != "node1" {
		t.Errorf("expected node ID 'node1', got %q", got.Nodes[0].Data.ID)
	}
	if ts.NodeCount() != 1 {
		t.Errorf("expected NodeCount 1, got %d", ts.NodeCount())
	}
}

func TestTopologyStoreGetNodeName(t *testing.T) {
	ts := NewTopologyStore()
	ts.Set(cy.Elements{
		Nodes: []cy.Node{
			{
				Data: cy.NodeData{
					ID: "abc",
					Attributes: map[string]interface{}{
						"label": "ABC Router",
					},
				},
			},
		},
	})

	if name := ts.GetNodeName("abc"); name != "ABC Router" {
		t.Errorf("expected 'ABC Router', got %q", name)
	}
	if name := ts.GetNodeName("nonexistent"); name != "" {
		t.Errorf("expected empty string for missing node, got %q", name)
	}
}

func TestTopologyStoreConcurrency(_ *testing.T) {
	ts := NewTopologyStore()
	var wg sync.WaitGroup

	// Writer goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			ts.Set(cy.Elements{
				Nodes: []cy.Node{
					{
						Data: cy.NodeData{
							ID: "node",
							Attributes: map[string]interface{}{
								"label": "Node",
							},
						},
					},
				},
			})
		}
	}()

	// Reader goroutines
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				_ = ts.Get()
				_ = ts.NodeCount()
				_ = ts.GetNodeName("node")
			}
		}()
	}

	wg.Wait()
}

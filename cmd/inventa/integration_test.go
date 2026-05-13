package main

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/neverthenetwork/inventa/internal/config"
	"github.com/neverthenetwork/inventa/internal/datastore"
	"github.com/neverthenetwork/inventa/internal/web"

	cy "gonum.org/v1/gonum/graph/formats/cytoscapejs"
)

var testLogger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
	Level: slog.LevelDebug,
}))

func TestIntegration_elementdataEndpoint(t *testing.T) {
	// Set up the same wiring as main, but with test data
	store := datastore.NewTopologyStore()
	store.Set(cy.Elements{
		Nodes: []cy.Node{
			{
				Data: cy.NodeData{
					ID: "router1",
					Attributes: map[string]interface{}{
						"label": "router1",
					},
				},
				Selectable: true,
			},
			{
				Data: cy.NodeData{
					ID: "router2",
					Attributes: map[string]interface{}{
						"label": "router2",
					},
				},
				Selectable: true,
			},
			{
				Data: cy.NodeData{
					ID: "router3",
					Attributes: map[string]interface{}{
						"label": "router3",
					},
				},
				Selectable: true,
			},
		},
		Edges: []cy.Edge{
			{
				Data: cy.EdgeData{
					ID:     "router1-router2",
					Source: "router1",
					Target: "router2",
					Attributes: map[string]interface{}{
						"igp_metric": "10",
					},
				},
				Selectable: true,
			},
			{
				Data: cy.EdgeData{
					ID:     "router2-router3",
					Source: "router2",
					Target: "router3",
					Attributes: map[string]interface{}{
						"igp_metric": "10",
					},
				},
				Selectable: true,
			},
		},
	})

	srv := &web.Server{
		StaticFS: staticFiles,
		Store:    store,
		Cfg:      &config.Conf{HTTPListenPort: 8080},
		Logger:   testLogger,
	}

	// Test /elementdata.json without filters
	req := httptest.NewRequest("GET", "/elementdata.json", nil)
	rec := httptest.NewRecorder()
	srv.JsHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var elements cy.Elements
	if err := json.Unmarshal(rec.Body.Bytes(), &elements); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}

	if len(elements.Nodes) != 3 {
		t.Errorf("expected 3 nodes, got %d", len(elements.Nodes))
	}
	if len(elements.Edges) != 2 {
		t.Errorf("expected 2 edges, got %d", len(elements.Edges))
	}

	// Test with src/dst path filter
	req2 := httptest.NewRequest("GET", "/elementdata.json?src=router1&dst=router3", nil)
	rec2 := httptest.NewRecorder()
	srv.JsHandler(rec2, req2)

	if rec2.Code != http.StatusOK {
		t.Fatalf("expected 200 for path lookup, got %d", rec2.Code)
	}

	var elements2 cy.Elements
	if err := json.Unmarshal(rec2.Body.Bytes(), &elements2); err != nil {
		t.Fatalf("failed to unmarshal path-filtered JSON: %v", err)
	}

	// All 3 nodes should still be returned, but only those on path have show=true
	visibleCount := 0
	for _, n := range elements2.Nodes {
		if show, _ := n.Data.Attributes["show"].(bool); show {
			visibleCount++
		}
	}
	if visibleCount != 3 {
		t.Errorf("expected 3 visible nodes on path router1->router3, got %d", visibleCount)
	}

	// The edge connecting them should be visible
	edgeVisible := 0
	for _, e := range elements2.Edges {
		if show, _ := e.Data.Attributes["show"].(bool); show {
			edgeVisible++
		}
	}
	if edgeVisible != 2 {
		t.Errorf("expected 2 visible edges on path, got %d", edgeVisible)
	}
}

func TestLoadJSON(t *testing.T) {
	dir := t.TempDir()
	jsonPath := dir + "/topology.json"

	topo := `{"nodes": [{"data": {"id": "test1", "label": "Test 1"}, "selectable": true}], "edges": []}`
	if err := os.WriteFile(jsonPath, []byte(topo), 0644); err != nil {
		t.Fatal(err)
	}

	store := datastore.NewTopologyStore()
	if err := loadJSON(jsonPath, store); err != nil {
		t.Fatalf("loadJSON() error = %v", err)
	}

	if store.NodeCount() != 1 {
		t.Errorf("expected 1 node, got %d", store.NodeCount())
	}

	got := store.Get()
	if got.Nodes[0].Data.ID != "test1" {
		t.Errorf("expected node ID 'test1', got %q", got.Nodes[0].Data.ID)
	}
}

func TestLoadJSON_missing(t *testing.T) {
	store := datastore.NewTopologyStore()
	if err := loadJSON("/nonexistent/file.json", store); err == nil {
		t.Error("expected error for missing file")
	}
}

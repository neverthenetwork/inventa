package web

import (
	"embed"
	"encoding/json"
	"io/fs"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/neverthenetwork/inventa/internal/config"
	"github.com/neverthenetwork/inventa/internal/datastore"
	"github.com/neverthenetwork/inventa/internal/spf"

	cy "gonum.org/v1/gonum/graph/formats/cytoscapejs"
)

//go:embed testdata/static/*
var testStaticFS embed.FS

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
}

func testServer() *Server {
	store := datastore.NewTopologyStore()
	store.Set(cy.Elements{
		Nodes: []cy.Node{
			{
				Data: cy.NodeData{
					ID: "A",
					Attributes: map[string]interface{}{
						"label": "Node A",
					},
				},
				Selectable: true,
			},
			{
				Data: cy.NodeData{
					ID: "B",
					Attributes: map[string]interface{}{
						"label": "Node B",
					},
				},
				Selectable: true,
			},
		},
		Edges: []cy.Edge{
			{
				Data: cy.EdgeData{
					ID:     "A-B",
					Source: "A",
					Target: "B",
					Attributes: map[string]interface{}{
						"igp_metric": "10",
					},
				},
				Selectable: true,
			},
		},
	})

	staticFS, err := fs.Sub(testStaticFS, "testdata")
	if err != nil {
		panic(err)
	}

	return &Server{
		StaticFS: staticFS,
		Store:    store,
		Cfg:      &config.Conf{HTTPListenPort: 8080},
		Logger:   testLogger(),
	}
}

func TestIndexHandler(t *testing.T) {
	srv := testServer()
	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()

	srv.IndexHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	contentType := rec.Header().Get("Content-Type")
	if !strings.Contains(contentType, "text/html") {
		t.Errorf("expected text/html content type, got %q", contentType)
	}
	if !strings.Contains(rec.Body.String(), "<html") {
		t.Error("expected HTML content in response body")
	}
}

func TestVRIndexHandler(t *testing.T) {
	srv := testServer()
	req := httptest.NewRequest("GET", "/vr", nil)
	rec := httptest.NewRecorder()

	srv.VRIndexHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	contentType := rec.Header().Get("Content-Type")
	if !strings.Contains(contentType, "text/html") {
		t.Errorf("expected text/html content type, got %q", contentType)
	}
}

func TestThreeDIndexHandler(t *testing.T) {
	srv := testServer()
	req := httptest.NewRequest("GET", "/3d", nil)
	rec := httptest.NewRecorder()

	srv.ThreeDIndexHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	contentType := rec.Header().Get("Content-Type")
	if !strings.Contains(contentType, "text/html") {
		t.Errorf("expected text/html content type, got %q", contentType)
	}
}

func TestJsHandler_noFilter(t *testing.T) {
	srv := testServer()
	req := httptest.NewRequest("GET", "/elementdata.json", nil)
	rec := httptest.NewRecorder()

	srv.JsHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	contentType := rec.Header().Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		t.Errorf("expected application/json content type, got %q", contentType)
	}

	var elements cy.Elements
	if err := json.Unmarshal(rec.Body.Bytes(), &elements); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if len(elements.Nodes) != 2 {
		t.Errorf("expected 2 nodes, got %d", len(elements.Nodes))
	}
	if len(elements.Edges) != 1 {
		t.Errorf("expected 1 edge, got %d", len(elements.Edges))
	}
	// When no filter is active, all nodes should have show=true
	for _, n := range elements.Nodes {
		show, _ := n.Data.Attributes["show"].(bool)
		if !show {
			t.Errorf("expected node %s to have show=true", n.Data.ID)
		}
	}
}

func TestJsHandler_withSrcDst(t *testing.T) {
	srv := testServer()
	req := httptest.NewRequest("GET", "/elementdata.json?src=A&dst=B", nil)
	rec := httptest.NewRecorder()

	srv.JsHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var elements cy.Elements
	if err := json.Unmarshal(rec.Body.Bytes(), &elements); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	// With src=A&dst=B, nodes A and B should be visible
	if len(elements.Nodes) != 2 {
		t.Errorf("expected 2 nodes, got %d", len(elements.Nodes))
	}

	hasTrue := false
	hasFalse := false
	for _, n := range elements.Nodes {
		show, _ := n.Data.Attributes["show"].(bool)
		if show {
			hasTrue = true
		} else {
			hasFalse = true
		}
	}
	if !hasTrue {
		t.Error("expected at least one node with show=true for path")
	}
	_ = hasFalse
}

func TestJsHandler_badSrcDst(t *testing.T) {
	srv := testServer()
	req := httptest.NewRequest("GET", "/elementdata.json?src=A&dst=NOEXIST", nil)
	rec := httptest.NewRecorder()

	srv.JsHandler(rec, req)

	// JsHandler returns 500 when SPF fails on unknown nodes
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 for invalid path lookup, got %d", rec.Code)
	}
}

func TestFilterNodes_noIncludeList(t *testing.T) {
	nodes := []cy.Node{
		{Data: cy.NodeData{ID: "A", Attributes: map[string]interface{}{"label": "A"}}},
		{Data: cy.NodeData{ID: "B", Attributes: map[string]interface{}{"label": "B"}}},
	}
	result := filterNodes(nodes, nil)
	if len(result) != 2 {
		t.Errorf("expected 2 nodes, got %d", len(result))
	}
	for _, n := range result {
		show, _ := n.Data.Attributes["show"].(bool)
		if !show {
			t.Errorf("expected node %s to have show=true", n.Data.ID)
		}
	}
}

func TestFilterNodes_withIncludeList(t *testing.T) {
	nodes := []cy.Node{
		{Data: cy.NodeData{ID: "A", Attributes: map[string]interface{}{"label": "A"}}},
		{Data: cy.NodeData{ID: "B", Attributes: map[string]interface{}{"label": "B"}}},
		{Data: cy.NodeData{ID: "C", Attributes: map[string]interface{}{"label": "C"}}},
	}
	result := filterNodes(nodes, []string{"A", "B"})

	if len(result) != 3 {
		t.Errorf("expected 3 nodes (all returned, some hidden), got %d", len(result))
	}

	visible := 0
	for _, n := range result {
		show, _ := n.Data.Attributes["show"].(bool)
		if show {
			visible++
		}
	}
	if visible != 2 {
		t.Errorf("expected 2 visible nodes, got %d", visible)
	}
}

func TestFilterNodes_empty(t *testing.T) {
	result := filterNodes(nil, nil)
	if len(result) != 0 {
		t.Errorf("expected 0 nodes, got %d", len(result))
	}
}

func TestFilterEdges_noPathPairs(t *testing.T) {
	edges := []cy.Edge{
		{Data: cy.EdgeData{ID: "e1", Source: "A", Target: "B", Attributes: map[string]interface{}{}}},
		{Data: cy.EdgeData{ID: "e2", Source: "B", Target: "C", Attributes: map[string]interface{}{}}},
	}
	result := filterEdges(edges, nil)
	if len(result) != 2 {
		t.Errorf("expected 2 edges, got %d", len(result))
	}
	for _, e := range result {
		show, _ := e.Data.Attributes["show"].(bool)
		if !show {
			t.Errorf("expected edge %s to have show=true", e.Data.ID)
		}
	}
}

func TestFilterEdges_withPathPairs(t *testing.T) {
	edges := []cy.Edge{
		{Data: cy.EdgeData{ID: "e1", Source: "A", Target: "B", Attributes: map[string]interface{}{}}},
		{Data: cy.EdgeData{ID: "e2", Source: "B", Target: "C", Attributes: map[string]interface{}{}}},
		{Data: cy.EdgeData{ID: "e3", Source: "C", Target: "D", Attributes: map[string]interface{}{}}},
	}
	result := filterEdges(edges, []spf.PathSegment{
		{Src: "A", Dst: "B"},
		{Src: "B", Dst: "C"},
	})

	if len(result) != 3 {
		t.Errorf("expected 3 edges, got %d", len(result))
	}

	visible := 0
	for _, e := range result {
		show, _ := e.Data.Attributes["show"].(bool)
		if show {
			visible++
		}
	}
	if visible != 2 {
		t.Errorf("expected 2 visible edges, got %d", visible)
	}
}

func TestFilterEdges_empty(t *testing.T) {
	result := filterEdges(nil, nil)
	if len(result) != 0 {
		t.Errorf("expected 0 edges, got %d", len(result))
	}
}

func TestCollapsePathPairs_single(t *testing.T) {
	pairs := []spf.PathSegment{
		{Src: "A", Dst: "B"},
		{Src: "B", Dst: "C"},
	}
	nodes := collapsePathPairs(pairs)
	expected := []string{"A", "B", "C"}
	if len(nodes) != len(expected) {
		t.Fatalf("expected %d nodes, got %d", len(expected), len(nodes))
	}
	for _, want := range expected {
		found := false
		for _, got := range nodes {
			if got == want {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected %q in collapsed list, got %v", want, nodes)
		}
	}
}

func TestCollapsePathPairs_duplicates(t *testing.T) {
	pairs := []spf.PathSegment{
		{Src: "A", Dst: "B"},
		{Src: "B", Dst: "A"},
	}
	nodes := collapsePathPairs(pairs)
	if len(nodes) != 2 {
		t.Errorf("expected 2 unique nodes, got %d", len(nodes))
	}
}

func TestCollapsePathPairs_empty(t *testing.T) {
	nodes := collapsePathPairs(nil)
	if len(nodes) != 0 {
		t.Errorf("expected 0 nodes, got %d", len(nodes))
	}
}

package spf

import (
	"log/slog"
	"os"
	"testing"

	cy "gonum.org/v1/gonum/graph/formats/cytoscapejs"
)

// testLogger returns a logger for testing that writes to stderr at debug level
func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
}

// simpleTriangle returns a 3-node triangle topology: A-B, B-C, A-C
func simpleTriangle() cy.Elements {
	return cy.Elements{
		Nodes: []cy.Node{
			makeNode("A", "Node A"),
			makeNode("B", "Node B"),
			makeNode("C", "Node C"),
		},
		Edges: []cy.Edge{
			makeEdge("A-B", "A", "B", "10"),
			makeEdge("B-C", "B", "C", "10"),
			makeEdge("A-C", "A", "C", "100"),
		},
	}
}

// lineTopology returns a 3-node line: A-B-C
func lineTopology() cy.Elements {
	return cy.Elements{
		Nodes: []cy.Node{
			makeNode("A", "Node A"),
			makeNode("B", "Node B"),
			makeNode("C", "Node C"),
		},
		Edges: []cy.Edge{
			makeEdge("A-B", "A", "B", "5"),
			makeEdge("B-C", "B", "C", "10"),
		},
	}
}

func makeNode(id, label string) cy.Node {
	return cy.Node{
		Data: cy.NodeData{
			ID: id,
			Attributes: map[string]interface{}{
				"label": label,
			},
		},
		Selectable: true,
	}
}

func makeEdge(id, src, dst, metric string) cy.Edge {
	return cy.Edge{
		Data: cy.EdgeData{
			ID:     id,
			Source: src,
			Target: dst,
			Attributes: map[string]interface{}{
				"igp_metric": metric,
			},
		},
		Selectable: true,
	}
}

func TestFindNode_found(t *testing.T) {
	nodes := []cy.Node{
		makeNode("A", "A"),
		makeNode("B", "B"),
	}
	idx, found := FindNode("B", nodes)
	if !found {
		t.Error("expected to find node B")
	}
	if idx != 1 {
		t.Errorf("expected index 1, got %d", idx)
	}
}

func TestFindNode_notFound(t *testing.T) {
	nodes := []cy.Node{
		makeNode("A", "A"),
	}
	_, found := FindNode("X", nodes)
	if found {
		t.Error("expected node X to not be found")
	}
}

func TestFindNode_empty(t *testing.T) {
	_, found := FindNode("A", nil)
	if found {
		t.Error("expected not found in nil slice")
	}
}

func TestMakeDijkstra_triangle(t *testing.T) {
	elements := simpleTriangle()
	graph, err := makeDijkstra(elements, testLogger())
	if err != nil {
		t.Fatalf("makeDijkstra() error = %v", err)
	}
	if graph == nil {
		t.Fatal("expected non-nil graph")
	}

	// Verify all three nodes are mapped
	for _, id := range []string{"A", "B", "C"} {
		if _, err := graph.GetMapping(id); err != nil {
			t.Errorf("expected node %s to exist in graph", id)
		}
	}
}

func TestMakeDijkstra_empty(t *testing.T) {
	graph, err := makeDijkstra(cy.Elements{}, testLogger())
	if err != nil {
		t.Fatalf("makeDijkstra() error = %v", err)
	}
	if graph == nil {
		t.Fatal("expected non-nil graph for empty input")
	}
}

func TestMakeDijkstra_noEdges(t *testing.T) {
	elements := cy.Elements{
		Nodes: []cy.Node{
			makeNode("X", "X"),
			makeNode("Y", "Y"),
		},
	}
	graph, err := makeDijkstra(elements, testLogger())
	if err != nil {
		t.Fatalf("makeDijkstra() error = %v", err)
	}
	// Nodes should be mapped even without edges
	for _, id := range []string{"X", "Y"} {
		if _, err := graph.GetMapping(id); err != nil {
			t.Errorf("expected node %s to exist in graph", id)
		}
	}
}

func TestGetBestPathNames_direct(t *testing.T) {
	elements := lineTopology()
	paths, err := GetBestPathNames(elements, "A", "C", testLogger())
	if err != nil {
		t.Fatalf("GetBestPathNames() error = %v", err)
	}
	if len(paths.Paths) != 1 {
		t.Fatalf("expected 1 path, got %d", len(paths.Paths))
	}
	expected := []string{"A", "B", "C"}
	for i, name := range paths.Paths[0].Path {
		if name != expected[i] {
			t.Errorf("path[%d] = %q, want %q", i, name, expected[i])
		}
	}
}

func TestGetBestPathNames_shortest(t *testing.T) {
	// Triangle: A-B (10), B-C (10), A-C (100)
	// A→C should go A-B-C (20) not A-C (100)
	elements := simpleTriangle()
	paths, err := GetBestPathNames(elements, "A", "C", testLogger())
	if err != nil {
		t.Fatalf("GetBestPathNames() error = %v", err)
	}
	if len(paths.Paths) != 1 {
		t.Fatalf("expected 1 path, got %d", len(paths.Paths))
	}
	path := paths.Paths[0].Path
	if len(path) != 3 {
		t.Fatalf("expected 3-node path, got %v", path)
	}
	if path[0] != "A" || path[1] != "B" || path[2] != "C" {
		t.Errorf("expected A->B->C, got %v", path)
	}
}

func TestGetBestPathNames_self(t *testing.T) {
	// Self-paths are not supported by the dijkstra library — it returns an error.
	elements := simpleTriangle()
	_, err := GetBestPathNames(elements, "A", "A", testLogger())
	if err == nil {
		t.Error("expected error for self-path")
	}
}

func TestGetBestPathNames_missingNode(t *testing.T) {
	elements := simpleTriangle()
	_, err := GetBestPathNames(elements, "A", "NOEXIST", testLogger())
	if err == nil {
		t.Error("expected error for nonexistent destination")
	}
}

func TestGetBestPathNames_disconnected(t *testing.T) {
	// Two separate nodes with no edge between them
	elements := cy.Elements{
		Nodes: []cy.Node{
			makeNode("X", "X"),
			makeNode("Y", "Y"),
		},
	}
	_, err := GetBestPathNames(elements, "X", "Y", testLogger())
	if err == nil {
		t.Error("expected error for unreachable destination")
	}
}

func TestGetPathSegments_singlePath(t *testing.T) {
	bp := BestPaths{
		Paths: []BestPath{
			{Path: []string{"A", "B", "C"}},
		},
	}
	segments := GetPathSegments(bp)
	if len(segments) != 2 {
		t.Fatalf("expected 2 segments, got %d", len(segments))
	}
	if segments[0].Src != "A" || segments[0].Dst != "B" {
		t.Errorf("segment[0]: expected A->B, got %s->%s", segments[0].Src, segments[0].Dst)
	}
	if segments[1].Src != "B" || segments[1].Dst != "C" {
		t.Errorf("segment[1]: expected B->C, got %s->%s", segments[1].Src, segments[1].Dst)
	}
}

func TestGetPathSegments_empty(t *testing.T) {
	segments := GetPathSegments(BestPaths{})
	if len(segments) != 0 {
		t.Errorf("expected 0 segments, got %d", len(segments))
	}
}

func TestGetPathSegments_singleNode(t *testing.T) {
	bp := BestPaths{
		Paths: []BestPath{
			{Path: []string{"A"}},
		},
	}
	segments := GetPathSegments(bp)
	if len(segments) != 0 {
		t.Errorf("expected 0 segments for single-node path, got %d", len(segments))
	}
}

func TestGetPathSegments_multiplePaths(t *testing.T) {
	bp := BestPaths{
		Paths: []BestPath{
			{Path: []string{"A", "B", "C"}},
			{Path: []string{"D", "E"}},
		},
	}
	segments := GetPathSegments(bp)
	if len(segments) != 3 {
		t.Fatalf("expected 3 segments, got %d", len(segments))
	}
}

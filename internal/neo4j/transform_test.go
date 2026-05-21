package neo4j

import (
	"strings"
	"testing"

	neodriver "github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j/db"

	cy "gonum.org/v1/gonum/graph/formats/cytoscapejs"
)

// newTestNode is a helper that creates a neo4j.Node for testing.
func newTestNode(id string, labels []string, props map[string]any) neodriver.Node {
	return neodriver.Node{
		ElementId: id,
		Labels:    labels,
		Props:     props,
	}
}

// newTestRel is a helper that creates a neo4j.Relationship for testing.
func newTestRel(id, relType, startID, endID string, props map[string]any) neodriver.Relationship {
	return neodriver.Relationship{
		ElementId:      id,
		Type:           relType,
		StartElementId: startID,
		EndElementId:   endID,
		Props:          props,
	}
}

// newTestEagerResult creates a minimal EagerResult for testing Transform.
//
// The neo4j.Record type is `db.Record`. We construct records directly.
func newTestEagerResult(n, m neodriver.Node, rel neodriver.Relationship) *neodriver.EagerResult {
	keys := []string{"n", "r", "m"}
	rec := &db.Record{Keys: keys, Values: []any{n, rel, m}}
	return &neodriver.EagerResult{
		Keys:    keys,
		Records: []*db.Record{rec},
	}
}

func TestTransformEmpty(t *testing.T) {
	result := &neodriver.EagerResult{
		Keys:    []string{"n", "r", "m"},
		Records: nil,
	}
	elements, err := Transform(result, "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(elements.Nodes) != 0 {
		t.Errorf("expected 0 nodes, got %d", len(elements.Nodes))
	}
	if len(elements.Edges) != 0 {
		t.Errorf("expected 0 edges, got %d", len(elements.Edges))
	}
}

func TestTransformSingleRelationship(t *testing.T) {
	n1 := newTestNode("n1", []string{"EC2Instance", "ComputeInstance"}, map[string]any{
		"instanceid": "i-abc123",
		"state":      "running",
	})
	n2 := newTestNode("n2", []string{"EC2Subnet"}, map[string]any{
		"subnetid":   "subnet-xyz",
		"cidr_block": "10.0.1.0/24",
	})
	rel := newTestRel("r1", "PART_OF_SUBNET", "n1", "n2", nil)

	result := newTestEagerResult(n1, n2, rel)
	elements, err := Transform(result, "neo4j")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(elements.Nodes) != 2 {
		t.Fatalf("expected 2 nodes, got %d", len(elements.Nodes))
	}
	if len(elements.Edges) != 1 {
		t.Fatalf("expected 1 edge, got %d", len(elements.Edges))
	}

	// Check edge.
	edge := elements.Edges[0]
	if edge.Data.ID != "r1" {
		t.Errorf("edge ID: expected r1, got %s", edge.Data.ID)
	}
	if edge.Data.Source != "n1" {
		t.Errorf("edge source: expected n1, got %s", edge.Data.Source)
	}
	if edge.Data.Target != "n2" {
		t.Errorf("edge target: expected n2, got %s", edge.Data.Target)
	}
	if edge.Data.Attributes["type"] != "PART_OF_SUBNET" {
		t.Errorf("edge type: expected PART_OF_SUBNET, got %v", edge.Data.Attributes["type"])
	}

	// Find nodes by ID.
	nodesByID := make(map[string]cy.Node)
	for _, n := range elements.Nodes {
		nodesByID[n.Data.ID] = n
	}

	instance, ok := nodesByID["n1"]
	if !ok {
		t.Fatal("node n1 not found")
	}
	if instance.Data.Attributes["group"] != "EC2Instance" {
		t.Errorf("group: expected EC2Instance, got %v", instance.Data.Attributes["group"])
	}
	if instance.Data.Attributes["instanceid"] != "i-abc123" {
		t.Errorf("instanceid: expected i-abc123, got %v", instance.Data.Attributes["instanceid"])
	}
	if instance.Data.Attributes["state"] != "running" {
		t.Errorf("state: expected running, got %v", instance.Data.Attributes["state"])
	}
	labels, _ := instance.Data.Attributes["labels"].(string)
	if !strings.Contains(labels, "EC2Instance") || !strings.Contains(labels, "ComputeInstance") {
		t.Errorf("labels: expected 'EC2Instance,ComputeInstance', got %v", labels)
	}

	subnet, ok := nodesByID["n2"]
	if !ok {
		t.Fatal("node n2 not found")
	}
	if subnet.Data.Attributes["group"] != "EC2Subnet" {
		t.Errorf("group: expected EC2Subnet, got %v", subnet.Data.Attributes["group"])
	}
	if subnet.Data.Attributes["source"] != "neo4j" {
		t.Errorf("source: expected neo4j, got %v", subnet.Data.Attributes["source"])
	}
}

func TestTransformNodeDeduplication(t *testing.T) {
	// Create a star topology: central node connected to 3 leaf nodes.
	center := newTestNode("center", []string{"AWSVpc"}, map[string]any{"vpcid": "vpc-1"})
	leaf1 := newTestNode("leaf1", []string{"EC2Subnet"}, map[string]any{"subnetid": "subnet-a"})
	leaf2 := newTestNode("leaf2", []string{"EC2Subnet"}, map[string]any{"subnetid": "subnet-b"})
	leaf3 := newTestNode("leaf3", []string{"EC2Subnet"}, map[string]any{"subnetid": "subnet-c"})

	// Build result with 3 records: center→leaf1, center→leaf2, center→leaf3.
	// The center node appears 3 times but should only produce 1 cytoscape node.
	keys := []string{"n", "r", "m"}
	records := []*db.Record{
		{Keys: keys, Values: []any{center, newTestRel("r1", "RESOURCE", "center", "leaf1", nil), leaf1}},
		{Keys: keys, Values: []any{center, newTestRel("r2", "RESOURCE", "center", "leaf2", nil), leaf2}},
		{Keys: keys, Values: []any{leaf3, newTestRel("r3", "RESOURCE", "leaf3", "center", nil), center}},
	}
	result := &neodriver.EagerResult{Keys: keys, Records: records}

	elements, err := Transform(result, "neo4j")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 4 unique nodes, 3 edges.
	if len(elements.Nodes) != 4 {
		t.Errorf("expected 4 deduplicated nodes, got %d", len(elements.Nodes))
	}
	if len(elements.Edges) != 3 {
		t.Errorf("expected 3 edges, got %d", len(elements.Edges))
	}

	// Verify center node appears exactly once.
	centerCount := 0
	for _, n := range elements.Nodes {
		if n.Data.ID == "center" {
			centerCount++
		}
	}
	if centerCount != 1 {
		t.Errorf("center node appeared %d times, expected 1", centerCount)
	}
}

func TestDefaultCypherQuery(t *testing.T) {
	query := DefaultCypherQuery()

	// Must contain the key structural elements.
	if !strings.Contains(query, "MATCH (n)-[r]-(m)") {
		t.Error("query missing MATCH clause")
	}
	if !strings.Contains(query, "RETURN n, r, m") {
		t.Error("query missing RETURN clause")
	}
	if !strings.Contains(query, "LIMIT 10000") {
		t.Error("query missing LIMIT")
	}

	// Must contain at least one known label.
	if !strings.Contains(query, "'EC2Subnet'") {
		t.Error("query missing EC2Subnet label")
	}
	if !strings.Contains(query, "'EC2Instance'") {
		t.Error("query missing EC2Instance label")
	}
}

func TestDefaultLabelFilter(t *testing.T) {
	filter := defaultLabelFilter()
	if !strings.Contains(filter, "'AWSVpc'") {
		t.Error("filter missing AWSVpc")
	}
	if !strings.Contains(filter, "'EC2Subnet'") {
		t.Error("filter missing EC2Subnet")
	}
}

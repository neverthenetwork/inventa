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
func newTestEagerResult(n, m neodriver.Node, rel neodriver.Relationship) *neodriver.EagerResult {
	keys := []string{"n", "r", "m"}
	rec := &db.Record{Keys: keys, Values: []any{n, rel, m}}
	return &neodriver.EagerResult{Keys: keys, Records: []*db.Record{rec}}
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
		"instanceid":         "i-abc123",
		"private_ip_address": "10.0.1.5",
		"state":              "running",
		"instance_type":      "t3.micro",
		"subnet_id":          "subnet-xyz",
	})
	n2 := newTestNode("n2", []string{"EC2Subnet"}, map[string]any{
		"subnetid":   "subnet-xyz",
		"cidr_block": "10.0.1.0/24",
		"vpc_id":     "vpc-1",
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
	if edge.Data.Attributes["type"] != "parent" {
		t.Errorf("edge type: expected parent, got %v", edge.Data.Attributes["type"])
	}

	// Find nodes by ID.
	nodesByID := make(map[string]cy.Node)
	for _, n := range elements.Nodes {
		nodesByID[n.Data.ID] = n
	}

	// Check instance node.
	instance, ok := nodesByID["n1"]
	if !ok {
		t.Fatal("node n1 not found")
	}
	if instance.Data.Attributes["group"] != "instance" {
		t.Errorf("group: expected instance, got %v", instance.Data.Attributes["group"])
	}
	if instance.Data.Attributes["label"] != "i-abc123" {
		t.Errorf("label: expected i-abc123, got %v", instance.Data.Attributes["label"])
	}
	// Property name mapping.
	if instance.Data.Attributes["privateIp"] != "10.0.1.5" {
		t.Errorf("privateIp: expected 10.0.1.5, got %v", instance.Data.Attributes["privateIp"])
	}
	if instance.Data.Attributes["instanceType"] != "t3.micro" {
		t.Errorf("instanceType: expected t3.micro, got %v", instance.Data.Attributes["instanceType"])
	}
	if instance.Data.Attributes["subnetId"] != "subnet-xyz" {
		t.Errorf("subnetId: expected subnet-xyz, got %v", instance.Data.Attributes["subnetId"])
	}

	// Check subnet node.
	subnet, ok := nodesByID["n2"]
	if !ok {
		t.Fatal("node n2 not found")
	}
	if subnet.Data.Attributes["group"] != "subnet" {
		t.Errorf("group: expected subnet, got %v", subnet.Data.Attributes["group"])
	}
	if subnet.Data.Attributes["label"] != "subnet-xyz" {
		t.Errorf("label: expected subnet-xyz, got %v", subnet.Data.Attributes["label"])
	}
	if subnet.Data.Attributes["cidr"] != "10.0.1.0/24" {
		t.Errorf("cidr: expected 10.0.1.0/24, got %v", subnet.Data.Attributes["cidr"])
	}
	if subnet.Data.Attributes["vpcId"] != "vpc-1" {
		t.Errorf("vpcId: expected vpc-1, got %v", subnet.Data.Attributes["vpcId"])
	}
}

func TestTransformNodeDeduplication(t *testing.T) {
	center := newTestNode("center", []string{"AWSVpc"}, map[string]any{
		"vpcid":              "vpc-1",
		"primary_cidr_block": "10.0.0.0/16",
	})
	leaf1 := newTestNode("leaf1", []string{"EC2Subnet"}, map[string]any{
		"subnetid": "subnet-a",
		"vpc_id":   "vpc-1",
	})
	leaf2 := newTestNode("leaf2", []string{"EC2Subnet"}, map[string]any{
		"subnetid": "subnet-b",
		"vpc_id":   "vpc-1",
	})
	leaf3 := newTestNode("leaf3", []string{"EC2Subnet"}, map[string]any{
		"subnetid": "subnet-c",
		"vpc_id":   "vpc-1",
	})

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

	if len(elements.Nodes) != 4 {
		t.Errorf("expected 4 deduplicated nodes, got %d", len(elements.Nodes))
	}
	if len(elements.Edges) != 3 {
		t.Errorf("expected 3 edges, got %d", len(elements.Edges))
	}

	centerCount := 0
	for _, n := range elements.Nodes {
		if n.Data.ID == "center" {
			centerCount++
		}
	}
	if centerCount != 1 {
		t.Errorf("center node appeared %d times, expected 1", centerCount)
	}

	// Verify VPC node has correct group and label.
	var vpc cy.Node
	for _, n := range elements.Nodes {
		if n.Data.ID == "center" {
			vpc = n
			break
		}
	}
	if vpc.Data.Attributes["group"] != "vpc" {
		t.Errorf("VPC group: expected vpc, got %v", vpc.Data.Attributes["group"])
	}
	if vpc.Data.Attributes["label"] != "vpc-1" {
		t.Errorf("VPC label: expected vpc-1, got %v", vpc.Data.Attributes["label"])
	}
	if vpc.Data.Attributes["cidr"] != "10.0.0.0/16" {
		t.Errorf("VPC cidr: expected 10.0.0.0/16, got %v", vpc.Data.Attributes["cidr"])
	}
}

func TestTransformFiltersNonTopologyNodes(t *testing.T) {
	// IpRule and AWSIpRange should be filtered out.
	ipRule := newTestNode("ip1", []string{"IpRule", "AWSIpRule"}, map[string]any{
		"fromport": float64(80),
	})
	ipRange := newTestNode("range1", []string{"AWSIpRange", "IpRange"}, map[string]any{
		"cidrip": "0.0.0.0/0",
	})
	subnet := newTestNode("sub1", []string{"EC2Subnet"}, map[string]any{
		"subnetid": "subnet-1",
	})

	// Edge between IpRule and IpRange (both non-topology).
	keys := []string{"n", "r", "m"}
	records := []*db.Record{
		{Keys: keys, Values: []any{ipRule, newTestRel("r1", "MEMBER_OF_IP_RULE", "ip1", "range1", nil), ipRange}},
		{Keys: keys, Values: []any{ipRule, newTestRel("r2", "MEMBER_OF_IP_RULE", "ip1", "sub1", nil), subnet}},
	}
	result := &neodriver.EagerResult{Keys: keys, Records: records}

	elements, err := Transform(result, "neo4j")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Only the subnet node should appear.
	if len(elements.Nodes) != 1 {
		t.Errorf("expected 1 node (subnet only), got %d", len(elements.Nodes))
	}
	if len(elements.Edges) != 0 {
		t.Errorf("expected 0 edges (all filtered), got %d", len(elements.Edges))
	}
}

func TestEdgeTypeMapping(t *testing.T) {
	n1 := newTestNode("n1", []string{"EC2Instance"}, map[string]any{"instanceid": "i-1"})
	n2 := newTestNode("n2", []string{"EC2SecurityGroup"}, map[string]any{"groupid": "sg-1"})
	rel := newTestRel("r1", "MEMBER_OF_EC2_SECURITY_GROUP", "n1", "n2", nil)

	result := newTestEagerResult(n1, n2, rel)
	elements, err := Transform(result, "neo4j")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(elements.Edges) != 1 {
		t.Fatal("expected 1 edge")
	}
	if elements.Edges[0].Data.Attributes["type"] != "member" {
		t.Errorf("edge type: expected member, got %v", elements.Edges[0].Data.Attributes["type"])
	}
}

func TestDefaultCypherQuery(t *testing.T) {
	query := DefaultCypherQuery()

	if !strings.Contains(query, "MATCH (n)-[r]->(m)") {
		t.Error("query missing MATCH clause")
	}
	if !strings.Contains(query, "RETURN n, r, m") {
		t.Error("query missing RETURN clause")
	}
	if !strings.Contains(query, "LIMIT 10000") {
		t.Error("query missing LIMIT")
	}
	if !strings.Contains(query, "'EC2Subnet'") {
		t.Error("query missing EC2Subnet label")
	}
	if !strings.Contains(query, "'EC2Instance'") {
		t.Error("query missing EC2Instance label")
	}
	// Non-topology labels should be excluded.
	if strings.Contains(query, "'IpRule'") || strings.Contains(query, "'AWSIpRange'") {
		t.Error("query should NOT include non-topology labels like IpRule or AWSIpRange")
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

func TestELBLabel(t *testing.T) {
	// ELB should use its name as label.
	n := newTestNode("elb1", []string{"AWSLoadBalancerV2", "LoadBalancer"}, map[string]any{
		"name":    "prod-alb",
		"dnsname": "prod-alb.elb.localhost",
		"scheme":  "internet-facing",
	})
	nd := nodeToCyElement(n, "elb", "test")
	if nd.Data.Attributes["label"] != "prod-alb" {
		t.Errorf("label: expected prod-alb, got %v", nd.Data.Attributes["label"])
	}
	if nd.Data.Attributes["group"] != "elb" {
		t.Errorf("group: expected elb, got %v", nd.Data.Attributes["group"])
	}
	if nd.Data.Attributes["dns"] != "prod-alb.elb.localhost" {
		t.Errorf("dns: expected prod-alb.elb.localhost, got %v", nd.Data.Attributes["dns"])
	}
}

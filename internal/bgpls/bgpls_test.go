package bgpls

import (
	"testing"

	"github.com/neverthenetwork/inventa/internal/config"
)

func TestMakePeerConfiguration(t *testing.T) {
	peer := MakePeerConfiguration("10.0.0.1", 65001)

	if peer == nil {
		t.Fatal("expected non-nil peer")
	}
	if peer.Conf == nil {
		t.Fatal("expected non-nil Conf")
	}
	if peer.Conf.NeighborAddress != "10.0.0.1" {
		t.Errorf("expected NeighborAddress 10.0.0.1, got %s", peer.Conf.NeighborAddress)
	}
	if peer.Conf.PeerAsn != 65001 {
		t.Errorf("expected PeerAsn 65001, got %d", peer.Conf.PeerAsn)
	}
	if peer.ApplyPolicy == nil {
		t.Fatal("expected non-nil ApplyPolicy")
	}
	if peer.ApplyPolicy.ImportPolicy == nil || peer.ApplyPolicy.ImportPolicy.DefaultAction.String() != "ACCEPT" {
		t.Error("expected ImportPolicy DefaultAction ACCEPT")
	}
	if peer.ApplyPolicy.ExportPolicy == nil || peer.ApplyPolicy.ExportPolicy.DefaultAction.String() != "REJECT" {
		t.Error("expected ExportPolicy DefaultAction REJECT")
	}
	if len(peer.AfiSafis) != 1 {
		t.Fatalf("expected 1 AfiSafi, got %d", len(peer.AfiSafis))
	}
	if !peer.AfiSafis[0].Config.Enabled {
		t.Error("expected AfiSafi enabled")
	}
}

func TestBuildTopology_singleNode(t *testing.T) {
	nodeMap := map[string]nodeNLRI{
		"10.0.0.1": {nodeName: "router1", localRouterID: "10.0.0.1"},
	}
	cfg := &config.Conf{}

	elements := buildTopology(nodeMap, nil, cfg)

	if len(elements.Nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(elements.Nodes))
	}
	node := elements.Nodes[0]
	if node.Data.ID != "router1" {
		t.Errorf("expected node ID 'router1', got %q", node.Data.ID)
	}
	label, _ := node.Data.Attributes["label"].(string)
	if label != "router1" {
		t.Errorf("expected label 'router1', got %q", label)
	}
	group, _ := node.Data.Attributes["group"].(string)
	if group != "noGroup" {
		t.Errorf("expected group 'noGroup', got %q", group)
	}
	if len(elements.Edges) != 0 {
		t.Errorf("expected 0 edges, got %d", len(elements.Edges))
	}
}

func TestBuildTopology_multipleNodes(t *testing.T) {
	nodeMap := map[string]nodeNLRI{
		"10.0.0.1": {nodeName: "router1", localRouterID: "10.0.0.1"},
		"10.0.0.2": {nodeName: "router2", localRouterID: "10.0.0.2"},
		"10.0.0.3": {nodeName: "router3", localRouterID: "10.0.0.3"},
	}
	cfg := &config.Conf{}

	elements := buildTopology(nodeMap, nil, cfg)

	if len(elements.Nodes) != 3 {
		t.Fatalf("expected 3 nodes, got %d", len(elements.Nodes))
	}
	// Verify all nodes have group "noGroup" when no split char
	for _, n := range elements.Nodes {
		group, _ := n.Data.Attributes["group"].(string)
		if group != "noGroup" {
			t.Errorf("node %s: expected 'noGroup', got %q", n.Data.ID, group)
		}
	}
}

func TestBuildTopology_withGroupSplit(t *testing.T) {
	nodeMap := map[string]nodeNLRI{
		"10.0.0.1": {nodeName: "area0-router1", localRouterID: "10.0.0.1"},
		"10.0.0.2": {nodeName: "area0-router2", localRouterID: "10.0.0.2"},
		"10.0.0.3": {nodeName: "area1-router1", localRouterID: "10.0.0.3"},
	}
	cfg := &config.Conf{
		GroupSplitChar:  "-",
		GroupSplitIndex: 0,
	}

	elements := buildTopology(nodeMap, nil, cfg)

	if len(elements.Nodes) != 3 {
		t.Fatalf("expected 3 nodes, got %d", len(elements.Nodes))
	}

	// First two nodes should be in group "area0", last in "area1"
	area0Count := 0
	area1Count := 0
	for _, n := range elements.Nodes {
		group, _ := n.Data.Attributes["group"].(string)
		switch group {
		case "area0":
			area0Count++
		case "area1":
			area1Count++
		default:
			t.Errorf("unexpected group %q for node %s", group, n.Data.ID)
		}
	}
	if area0Count != 2 {
		t.Errorf("expected 2 nodes in area0, got %d", area0Count)
	}
	if area1Count != 1 {
		t.Errorf("expected 1 node in area1, got %d", area1Count)
	}
}

func TestBuildTopology_withEdges(t *testing.T) {
	nodeMap := map[string]nodeNLRI{
		"10.0.0.1": {nodeName: "router1", localRouterID: "10.0.0.1"},
		"10.0.0.2": {nodeName: "router2", localRouterID: "10.0.0.2"},
	}
	linkTable := []linkNLRI{
		{
			localRouterID:    "10.0.0.1",
			remoteRouterID:   "10.0.0.2",
			localIPv4Addr:    "192.168.1.1",
			neighborIPv4Addr: "192.168.1.2",
			igpMetric:        100,
			srAdjacencySid:   5001,
		},
	}
	cfg := &config.Conf{}

	elements := buildTopology(nodeMap, linkTable, cfg)

	if len(elements.Nodes) != 2 {
		t.Errorf("expected 2 nodes, got %d", len(elements.Nodes))
	}
	if len(elements.Edges) != 1 {
		t.Fatalf("expected 1 edge, got %d", len(elements.Edges))
	}

	edge := elements.Edges[0]
	if edge.Data.Source != "router1" {
		t.Errorf("expected source 'router1', got %q", edge.Data.Source)
	}
	if edge.Data.Target != "router2" {
		t.Errorf("expected target 'router2', got %q", edge.Data.Target)
	}

	adjSid, _ := edge.Data.Attributes["adjacency_sid"].(string)
	if adjSid != "5001" {
		t.Errorf("expected adjacency_sid '5001', got %q", adjSid)
	}
	igpMetric, _ := edge.Data.Attributes["igp_metric"].(string)
	if igpMetric != "100" {
		t.Errorf("expected igp_metric '100', got %q", igpMetric)
	}

	// Verify edge ID format
	expectedID := "router1:router2_5001"
	if edge.Data.ID != expectedID {
		t.Errorf("expected edge ID %q, got %q", expectedID, edge.Data.ID)
	}
}

func TestBuildTopology_empty(t *testing.T) {
	cfg := &config.Conf{}
	elements := buildTopology(nil, nil, cfg)

	if len(elements.Nodes) != 0 {
		t.Errorf("expected 0 nodes, got %d", len(elements.Nodes))
	}
	if len(elements.Edges) != 0 {
		t.Errorf("expected 0 edges, got %d", len(elements.Edges))
	}
}

func TestBuildTopology_noNodesForEdge(t *testing.T) {
	// Edge referencing a routerID not in nodeMap — Go maps return the zero value.
	// The edge will be created with empty Source/Target since the missing node's
	// nodeName is empty string. This tests the code doesn't panic.
	nodeMap := map[string]nodeNLRI{
		"10.0.0.1": {nodeName: "router1", localRouterID: "10.0.0.1"},
	}
	linkTable := []linkNLRI{
		{
			localRouterID:  "10.0.0.1",
			remoteRouterID: "10.0.0.99", // not in nodeMap
		},
	}
	cfg := &config.Conf{}
	elements := buildTopology(nodeMap, linkTable, cfg)

	if len(elements.Edges) != 1 {
		t.Fatalf("expected 1 edge (with empty target), got %d", len(elements.Edges))
	}
	// Target should be empty string since nodeMap["10.0.0.99"] is zero value
	if elements.Edges[0].Data.Target != "" {
		t.Errorf("expected empty target for unknown router, got %q", elements.Edges[0].Data.Target)
	}
}

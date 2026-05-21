// Package neo4j provides a discovery plugin that loads topology data
// from a Neo4j graph database (e.g. populated by Cartography).
package neo4j

import (
	"fmt"
	"strings"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"

	cy "gonum.org/v1/gonum/graph/formats/cytoscapejs"
)

// defaultCypherQuery returns the set of interesting topology labels as a
// comma-separated string of label literals for embedding in Cypher.
//
// This targets the labels Cartography creates for AWS infrastructure:
//
//	VPCs, subnets, instances, security groups, load balancers, listeners,
//	network interfaces, IP ranges, and IP rules.
func defaultLabelFilter() string {
	labels := []string{
		"AWSVpc",
		"EC2Subnet",
		"EC2Instance",
		"EC2SecurityGroup",
		"AWSLoadBalancerV2",
		"ELBV2Listener",
		"NetworkInterface",
		"AWSIpRange",
		"IpRule",
	}
	quoted := make([]string, len(labels))
	for i, l := range labels {
		quoted[i] = fmt.Sprintf("'%s'", l)
	}
	return strings.Join(quoted, ",")
}

// DefaultCypherQuery returns a Cypher query that fetches all nodes and
// relationships matching the default topology labels, with a 10 000 row limit.
func DefaultCypherQuery() string {
	return fmt.Sprintf(
		`MATCH (n)-[r]-(m)
		 WHERE any(label IN labels(n) WHERE label IN [%s])
		   AND any(label IN labels(m) WHERE label IN [%s])
		 RETURN n, r, m
		 LIMIT 10000`,
		defaultLabelFilter(),
		defaultLabelFilter(),
	)
}

// Transform converts a Neo4j EagerResult into cytoscape.js Elements.
//
// Each record is expected to have keys "n" (node), "r" (relationship),
// and "m" (node). Nodes are deduplicated by their Neo4j element ID.
// Node labels are joined into a comma-separated string stored in
// data.attributes.labels. The primary label is used for data.group.
// Relationship type is stored as data.type.
func Transform(result *neo4j.EagerResult, sourceName string) (*cy.Elements, error) {
	seenNodes := make(map[string]bool)
	var elements cy.Elements

	for _, record := range result.Records {
		nodeN, _, err := neo4j.GetRecordValue[neo4j.Node](record, "n")
		if err != nil {
			return nil, fmt.Errorf("extracting node n: %w", err)
		}
		nodeM, _, err := neo4j.GetRecordValue[neo4j.Node](record, "m")
		if err != nil {
			return nil, fmt.Errorf("extracting node m: %w", err)
		}
		rel, _, err := neo4j.GetRecordValue[neo4j.Relationship](record, "r")
		if err != nil {
			return nil, fmt.Errorf("extracting relationship r: %w", err)
		}

		// Add node N if not seen.
		if !seenNodes[nodeN.ElementId] {
			seenNodes[nodeN.ElementId] = true
			elements.Nodes = append(elements.Nodes, nodeToCyElement(nodeN, sourceName))
		}

		// Add node M if not seen.
		if !seenNodes[nodeM.ElementId] {
			seenNodes[nodeM.ElementId] = true
			elements.Nodes = append(elements.Nodes, nodeToCyElement(nodeM, sourceName))
		}

		// Add the relationship as an edge.
		elements.Edges = append(elements.Edges, relToCyEdge(rel, nodeN.ElementId, nodeM.ElementId))
	}

	return &elements, nil
}

// nodeToCyElement converts a Neo4j node into a cytoscape.js Node element.
//
// The node's element ID becomes the cytoscape Data.ID. Labels are joined
// with commas and stored in Data.Attributes["labels"]. The primary label
// is used as Data.Attributes["group"] for cytoscape styling.
// All Neo4j properties are copied into Data.Attributes.
// The source plugin name is stored in Data.Attributes["source"].
func nodeToCyElement(n neo4j.Node, sourceName string) cy.Node {
	attrs := make(map[string]interface{})

	// Copy all Neo4j properties first (so explicit keys below take precedence).
	for k, v := range n.Props {
		attrs[k] = v
	}

	// Set cytoscape metadata.
	attrs["labels"] = strings.Join(n.Labels, ",")
	attrs["source"] = sourceName
	if len(n.Labels) > 0 {
		attrs["group"] = n.Labels[0]
	}

	return cy.Node{
		Data: cy.NodeData{
			ID:         n.ElementId,
			Attributes: attrs,
		},
	}
}

// relToCyEdge converts a Neo4j relationship into a cytoscape.js Edge element.
//
// The relationship's element ID is used as Data.ID. Start/end element IDs
// map to Data.Source / Data.Target. The relationship type is stored in
// Data.Attributes["type"]. All Neo4j properties are copied into
// Data.Attributes.
func relToCyEdge(r neo4j.Relationship, sourceID, targetID string) cy.Edge {
	attrs := make(map[string]interface{})

	for k, v := range r.Props {
		attrs[k] = v
	}

	attrs["type"] = r.Type

	return cy.Edge{
		Data: cy.EdgeData{
			ID:         r.ElementId,
			Source:     sourceID,
			Target:     targetID,
			Attributes: attrs,
		},
	}
}

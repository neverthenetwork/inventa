// Package neo4j provides a discovery plugin that loads topology data
// from a Neo4j graph database (e.g. populated by Cartography).
package neo4j

import (
	"fmt"
	"strings"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"

	cy "gonum.org/v1/gonum/graph/formats/cytoscapejs"
)

// labelToGroup maps Cartography Neo4j labels to frontend cytoscape groups.
// Nodes with labels not in this map are treated as non-topology metadata
// and filtered out (e.g. IpRule, AWSIpRange).
var labelToGroup = map[string]string{
	"AWSAccount":        "vpc", // top-level container — styled as VPC
	"AWSVpc":            "vpc",
	"EC2Subnet":         "subnet",
	"EC2Instance":       "instance",
	"EC2SecurityGroup":  "security_group",
	"AWSLoadBalancerV2": "elb",
	"LoadBalancer":      "elb",
	"LoadBalancerV2":    "elb",
	"ELBV2Listener":     "elb",
	// AWSIpRange, IpRule, AWSIpPermissionInbound, NetworkInterface,
	// EC2Reservation, AWSRootPrincipal — filtered out; metadata only.
}

// relTypeToEdgeType maps Cartography Neo4j relationship types to frontend
// cytoscape edge types (which drive styling and detail panels).
var relTypeToEdgeType = map[string]string{
	"RESOURCE":                     "parent",
	"PART_OF_SUBNET":               "parent",
	"MEMBER_OF_EC2_SECURITY_GROUP": "member",
	"MEMBER_OF_EC2_RESERVATION":    "parent",
	"NETWORK_INTERFACE":            "attached",
	"ELBV2_LISTENER":               "parent",
}

// primaryLabel returns the primary (most specific) label for a Neo4j node
// from the Cartography label set.
func primaryLabel(labels []string) string {
	// Priority order: most specific topology labels first.
	priority := []string{
		"AWSAccount", "AWSVpc", "EC2Subnet", "EC2Instance", "EC2SecurityGroup",
		"AWSLoadBalancerV2", "ELBV2Listener", "NetworkInterface",
	}
	for _, want := range priority {
		for _, have := range labels {
			if have == want {
				return want
			}
		}
	}
	// Fallback: first label that maps to a known group.
	for _, l := range labels {
		if _, ok := labelToGroup[l]; ok {
			return l
		}
	}
	return ""
}

// nodeLabel generates a human-readable label for a Neo4j topology node.
// Falls back to the Neo4j element ID if nothing better is available.
func nodeLabel(n neo4j.Node) string {
	pLabel := primaryLabel(n.Labels)

	// Label-specific identifiers take priority over generic name.
	switch pLabel {
	case "AWSAccount":
		if v := stringProp(n.Props, "id"); v != "" {
			return "Account: " + v
		}
		return "AWS Account"
	case "AWSVpc":
		if v := stringProp(n.Props, "vpcid"); v != "" {
			return v
		}
		if v := stringProp(n.Props, "vpc_id"); v != "" {
			return v
		}
	case "EC2Subnet":
		if v := stringProp(n.Props, "subnetid"); v != "" {
			return v
		}
		if v := stringProp(n.Props, "subnet_id"); v != "" {
			return v
		}
	case "EC2Instance":
		if v := stringProp(n.Props, "instanceid"); v != "" {
			return v
		}
	case "EC2SecurityGroup":
		if v := stringProp(n.Props, "groupid"); v != "" {
			return v
		}
		if v := stringProp(n.Props, "group_id"); v != "" {
			return v
		}
	case "AWSLoadBalancerV2", "LoadBalancer", "LoadBalancerV2":
		if v := stringProp(n.Props, "name"); v != "" {
			return v
		}
		if v := stringProp(n.Props, "dnsname"); v != "" {
			return v
		}
	case "ELBV2Listener":
		return fmt.Sprintf("%s:%v",
			stringProp(n.Props, "protocol"),
			n.Props["port"],
		)
	case "NetworkInterface":
		if v := stringProp(n.Props, "private_ip_address"); v != "" {
			return v
		}
	}

	// Generic fallbacks.
	if v := stringProp(n.Props, "name"); v != "" {
		return v
	}
	if v := stringProp(n.Props, "groupname"); v != "" {
		return v
	}
	if v := stringProp(n.Props, "id"); v != "" {
		return v
	}
	return n.ElementId
}

// nodeGroup returns the frontend group for a Neo4j node.
// Returns empty string if the node should be filtered out.
func nodeGroup(labels []string) string {
	p := primaryLabel(labels)
	if p == "" {
		return ""
	}
	return labelToGroup[p]
}

// edgeType returns the frontend edge type for a Neo4j relationship.
func edgeType(relType string) string {
	if t, ok := relTypeToEdgeType[relType]; ok {
		return t
	}
	// Default: use the raw Neo4j type, lowercased and underscored for
	// generic display (won't get special styling but won't break either).
	return strings.ToLower(strings.ReplaceAll(relType, "_", "-"))
}

// defaultCypherQuery returns the set of interesting topology labels as a
// comma-separated string of label literals for embedding in Cypher.
func defaultLabelFilter() string {
	labels := []string{
		"AWSAccount",
		"AWSVpc",
		"EC2Subnet",
		"EC2Instance",
		"EC2SecurityGroup",
		"AWSLoadBalancerV2",
		"ELBV2Listener",
		"NetworkInterface",
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
		`MATCH (n)-[r]->(m)
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
//
// Node labels are mapped to frontend cytoscape groups (vpc, subnet,
// instance, elb, security_group) and human-readable labels are generated
// from identifying properties. Relationship types are mapped to frontend
// edge types (parent, member, attached, etc.).
func Transform(result *neo4j.EagerResult, sourceName string) (*cy.Elements, error) {
	seenNodes := make(map[string]bool)
	nodeMap := make(map[string]cy.Node) // elementId → node
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

		// Only include nodes with a known group.
		groupN := nodeGroup(nodeN.Labels)
		groupM := nodeGroup(nodeM.Labels)
		if groupN == "" && groupM == "" {
			continue
		}

		// Deduplicate and store nodes.
		if groupN != "" && !seenNodes[nodeN.ElementId] {
			seenNodes[nodeN.ElementId] = true
			nd := nodeToCyElement(nodeN, groupN, sourceName)
			nodeMap[nodeN.ElementId] = nd
		}
		if groupM != "" && !seenNodes[nodeM.ElementId] {
			seenNodes[nodeM.ElementId] = true
			nd := nodeToCyElement(nodeM, groupM, sourceName)
			nodeMap[nodeM.ElementId] = nd
		}

		// Only add edge if BOTH endpoints are topology nodes.
		if groupN != "" && groupM != "" {
			elements.Edges = append(elements.Edges,
				relToCyEdge(rel, nodeN.ElementId, nodeM.ElementId))
		}
	}

	// Second pass: set parent on child nodes for compound layout.
	// Subnets → parent VPC, Instances → parent Subnet, Listeners → parent ELB.
	for _, nd := range nodeMap {
		switch nd.Data.Attributes["group"] {
		case "subnet":
			if vpcID := stringProp(nd.Data.Attributes, "vpc_id"); vpcID != "" {
				nd.Data.Parent = vpcID
			}
		case "instance":
			if subnetID := stringProp(nd.Data.Attributes, "subnet_id"); subnetID != "" {
				nd.Data.Parent = subnetID
			}
		case "elb":
			// ELB/Listener — parent is set via edge traversal, not here.
		}
		elements.Nodes = append(elements.Nodes, nd)
	}

	return &elements, nil
}

// nodeToCyElement converts a Neo4j node into a cytoscape.js Node element.
//
// Sets Data.ID to the Neo4j element ID, generates a human-readable
// Data.Attributes["label"], maps the group, and copies/re-maps Neo4j
// properties to match the frontend's expected fields.
func nodeToCyElement(n neo4j.Node, group, sourceName string) cy.Node {
	attrs := make(map[string]any)

	// Copy all raw Neo4j properties.
	for k, v := range n.Props {
		attrs[k] = v
	}

	// Frontend-expected metadata.
	attrs["labels"] = strings.Join(n.Labels, ",")
	attrs["group"] = group
	attrs["source"] = sourceName
	attrs["label"] = nodeLabel(n)

	// --- Property name mapping (snake_case Cartography → camelCase frontend) ---

	// VPC fields.
	mapProp(attrs, "vpc_id", "vpcId")
	mapProp(attrs, "primary_cidr_block", "cidr")
	mapProp(attrs, "is_default", "isDefault")

	// Subnet fields.
	mapProp(attrs, "subnet_id", "subnetId")
	mapProp(attrs, "cidr_block", "cidr")
	mapProp(attrs, "availability_zone", "az")
	mapProp(attrs, "map_public_ip_on_launch", "public")

	// Instance fields.
	mapProp(attrs, "instance_type", "instanceType")
	mapProp(attrs, "private_ip_address", "privateIp")
	mapProp(attrs, "public_ip_address", "publicIp")
	mapProp(attrs, "state", "state")
	mapProp(attrs, "subnet_id", "subnetId")

	// Security group fields.
	mapProp(attrs, "group_name", "groupName")
	mapProp(attrs, "description", "description")
	mapProp(attrs, "group_id", "groupId")

	// ELB fields.
	mapProp(attrs, "dns_name", "dns")
	mapProp(attrs, "dnsname", "dns")
	mapProp(attrs, "scheme", "scheme")
	mapProp(attrs, "type", "type")

	// Generic identifier.
	mapProp(attrs, "name", "name")

	return cy.Node{
		Data: cy.NodeData{
			ID:         n.ElementId,
			Attributes: attrs,
		},
		Selectable: true,
	}
}

// relToCyEdge converts a Neo4j relationship into a cytoscape.js Edge element.
//
// Maps the Neo4j relationship type to the frontend edge type system
// (parent, member, attached, target, lb-subnet, egress).
func relToCyEdge(r neo4j.Relationship, sourceID, targetID string) cy.Edge {
	attrs := make(map[string]any)

	// Copy all Neo4j properties.
	for k, v := range r.Props {
		attrs[k] = v
	}

	attrs["type"] = edgeType(r.Type)

	return cy.Edge{
		Data: cy.EdgeData{
			ID:         r.ElementId,
			Source:     sourceID,
			Target:     targetID,
			Attributes: attrs,
		},
		Selectable: true,
	}
}

// stringProp extracts a string value from a property map.
func stringProp(props map[string]any, key string) string {
	if v, ok := props[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// mapProp copies a value from srcKey to dstKey in attrs if srcKey exists.
func mapProp(attrs map[string]any, srcKey, dstKey string) {
	if v, ok := attrs[srcKey]; ok {
		attrs[dstKey] = v
	}
}

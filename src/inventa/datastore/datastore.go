package datastore

import (
	cy "gonum.org/v1/gonum/graph/formats/cytoscapejs"
)

type myElements struct {
	Elements cy.Elements
}

func (elements *myElements) getName(nodeID string) string {
	for _, n := range elements.Elements.Nodes {
		if n.Data.ID == nodeID {
			return n.Data.Attributes["label"].(string)
		}
	}
	return ""
}

// FindNode checks whether a node exists in the list of nodes
func FindNode(what string, where []cy.Node) (idx int, found bool) {
	for i, v := range where {
		if v.Data.ID == what {
			return i, true
		}
	}
	return 0, false
}

// Elements holds all our graph nodes/verteces
var Elements cy.Elements

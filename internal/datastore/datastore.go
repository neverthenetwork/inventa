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

// Elements holds all our graph nodes/verteces
var Elements cy.Elements

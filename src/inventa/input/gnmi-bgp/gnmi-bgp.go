package bgpls

import (
	"fmt"
	"strconv"

	"github.com/shelson/inventa/src/inventa/datastore"

	cy "gonum.org/v1/gonum/graph/formats/cytoscapejs"
)

type bgpNeighbor struct {
	NeighborAddress string `json:"neighbor_address"`
	NeighborAS      int64  `json:"neighbor_as"`
	NeighborID      string `json:"neighbor_id"`
	NeighborState   string `json:"neighbor_state"`
	NeighborType    string `json:"neighbor_type"`
}

type bgpNeighborList struct {
	Neighbors []bgpNeighbor
}

var bgpData map[string]bgpNeighborList

func MakeSomeNeighbors() {
	// Make some fake neighbors on some fake hosts
	bgpData = make(map[string]bgpNeighborList)
	var neighborList bgpNeighborList
	neighborList.Neighbors = append(neighborList.Neighbors, bgpNeighbor{
		NeighborAddress: "23.24.25.2",
		NeighborAS:      1234,
		NeighborID:      "Foo",
		NeighborState:   "Established",
		NeighborType:    "External",
	})
	neighborList.Neighbors = append(neighborList.Neighbors, bgpNeighbor{
		NeighborAddress: "23.24.25.3",
		NeighborAS:      7224,
		NeighborID:      "Foo",
		NeighborState:   "Established",
		NeighborType:    "External",
	})
	neighborList.Neighbors = append(neighborList.Neighbors, bgpNeighbor{
		NeighborAddress: "23.24.25.4",
		NeighborAS:      36549,
		NeighborID:      "Foo",
		NeighborState:   "Established",
		NeighborType:    "External",
	})
	neighborList.Neighbors = append(neighborList.Neighbors, bgpNeighbor{
		NeighborAddress: "23.24.25.34",
		NeighborAS:      12354,
		NeighborID:      "Foo",
		NeighborState:   "Established",
		NeighborType:    "External",
	})

	bgpData["router1"] = neighborList

	neighborList.Neighbors = []bgpNeighbor{}

	neighborList.Neighbors = append(neighborList.Neighbors, bgpNeighbor{
		NeighborAddress: "45.45.45.1",
		NeighborAS:      1234,
		NeighborID:      "Foo",
		NeighborState:   "Established",
		NeighborType:    "External",
	})
	neighborList.Neighbors = append(neighborList.Neighbors, bgpNeighbor{
		NeighborAddress: "45.45.45.2",
		NeighborAS:      7224,
		NeighborID:      "Foo",
		NeighborState:   "Established",
		NeighborType:    "External",
	})
	neighborList.Neighbors = append(neighborList.Neighbors, bgpNeighbor{
		NeighborAddress: "45.45.45.3",
		NeighborAS:      36549,
		NeighborID:      "Foo",
		NeighborState:   "Established",
		NeighborType:    "External",
	})
	neighborList.Neighbors = append(neighborList.Neighbors, bgpNeighbor{
		NeighborAddress: "45.45.45.4",
		NeighborAS:      12354,
		NeighborID:      "Foo",
		NeighborState:   "Established",
		NeighborType:    "External",
	})

	bgpData["router2"] = neighborList

	neighborList.Neighbors = []bgpNeighbor{}

	neighborList.Neighbors = append(neighborList.Neighbors, bgpNeighbor{
		NeighborAddress: "53.2.1.3",
		NeighborAS:      36549,
		NeighborID:      "Foo",
		NeighborState:   "Established",
		NeighborType:    "External",
	})

	bgpData["router3"] = neighborList

}

func AddBgpNeighbors() {
	// Add the neighbors to the datastore
	for host := range bgpData {
		for _, neighbor := range bgpData[host].Neighbors {
			generatedNeighborID := fmt.Sprintf("%d", neighbor.NeighborAS)
			if _, found := datastore.FindNode(generatedNeighborID, datastore.Elements.Nodes); !found {
				node := cy.Node{
					Data: cy.NodeData{
						ID: generatedNeighborID,
						Attributes: map[string]interface{}{
							"label":   generatedNeighborID,
							"address": neighbor.NeighborAddress,
							"asn":     neighbor.NeighborAS,
							"group":   "bgp",
							"cluster": 10,
						},
					},
					Selectable: true,
				}
				datastore.Elements.Nodes = append(datastore.Elements.Nodes, node)
			}

			// Create an edge to the new node
			edge := cy.Edge{
				Data: cy.EdgeData{
					ID:     fmt.Sprintf("%s:%s_%s", host, generatedNeighborID, "1"),
					Source: host,
					Target: generatedNeighborID,
					Attributes: map[string]interface{}{
						"asn":        strconv.FormatInt(int64(neighbor.NeighborAS), 10),
						"igp_metric": "170", // stealing Junos route preference for now
					},
				},
				Selectable: true,
			}
			datastore.Elements.Edges = append(datastore.Elements.Edges, edge)
		}
	}
}

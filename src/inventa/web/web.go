package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/neverthenetwork/inventa/src/inventa/datastore"
	"github.com/neverthenetwork/inventa/src/inventa/spf"
	"github.com/neverthenetwork/inventa/src/inventa/utils"

	cy "gonum.org/v1/gonum/graph/formats/cytoscapejs"
)

// IndexHandler is the handler for the index page
func IndexHandler(w http.ResponseWriter, r *http.Request) {
	content, _ := os.ReadFile("../../static/index.html")
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, "%s", string(content))
}

// VRIndexHandler is the handler for the VR index page
func VRIndexHandler(w http.ResponseWriter, r *http.Request) {
	content, _ := os.ReadFile("../../static/vrindex.html")
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, "%s", string(content))
}

// ThreeDIndexHandler is the handler for the VR index page
func ThreeDIndexHandler(w http.ResponseWriter, r *http.Request) {
	content, _ := os.ReadFile("../../static/3dindex.html")
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, "%s", string(content))
}

// JsHandler is the handler for returning the json data
func JsHandler(w http.ResponseWriter, r *http.Request) {
	var includeList = []string{}
	var pathPairs = []spf.PathSegment{}
	src := r.URL.Query().Get("src")
	dst := r.URL.Query().Get("dst")
	w.Header().Set("Content-Type", "application/json")
	if src != "" && dst != "" {
		bestPathNames, _ := spf.GetBestPathNames(datastore.Elements, src, dst)
		pathPairs = spf.GetPathSegments(bestPathNames)
		includeList = collapsePathPairs(pathPairs)
	}
	var filteredElements = cy.Elements{
		Nodes: make([]cy.Node, 0),
		Edges: make([]cy.Edge, 0),
	}
	filteredElements.Nodes = filterNodes(datastore.Elements.Nodes, includeList)
	filteredElements.Edges = filterEdges(datastore.Elements.Edges, pathPairs)
	if err := json.NewEncoder(w).Encode(filteredElements); err != nil {
		fmt.Printf("%s\n", err)
	}
}

func filterNodes(nodes []cy.Node, includeList []string) []cy.Node {
	var nodeList = make([]cy.Node, 0)
	for _, n := range nodes {
		if len(includeList) == 0 {
			n.Data.Attributes["show"] = true
			nodeList = append(nodeList, n)
		} else {
			_, found := utils.FindInArray(n.Data.ID, includeList)
			if found {
				n.Data.Attributes["show"] = true
				nodeList = append(nodeList, n)
			} else {
				n.Data.Attributes["show"] = false
				nodeList = append(nodeList, n)
			}
		}
	}
	return nodeList
}

func filterEdges(edges []cy.Edge, pathPairs []spf.PathSegment) []cy.Edge {
	var edgeList = make([]cy.Edge, 0)
	if len(pathPairs) == 0 {
		for _, e := range edges {
			e.Data.Attributes["show"] = true
			edgeList = append(edgeList, e)
		}
	} else {
		for _, e := range edges {
			var found = false
			for _, pp := range pathPairs {
				// fmt.Printf("%s:%s vs %s:%s\n", e.Data.Source, e.Data.Target, pp.src, pp.dst)
				if e.Data.Source == pp.Src && e.Data.Target == pp.Dst {
					// fmt.Printf("FOUND: %s:%s vs %s:%s\n", e.Data.Source, e.Data.Target, pp.src, pp.dst)
					found = true
				}
			}
			if found {
				e.Data.Attributes["show"] = true
				edgeList = append(edgeList, e)
			} else {
				e.Data.Attributes["show"] = false
				edgeList = append(edgeList, e)
			}
		}
	}
	return edgeList
}

// return a list of nodes that exist in the pathpairs list
func collapsePathPairs(pathPairs []spf.PathSegment) []string {
	var nodeList = make([]string, 0)
	for _, pp := range pathPairs {
		if _, f := utils.FindInArray(pp.Src, nodeList); !f {
			nodeList = append(nodeList, pp.Src)
		}
		if _, f := utils.FindInArray(pp.Dst, nodeList); !f {
			nodeList = append(nodeList, pp.Dst)
		}
	}
	return nodeList
}

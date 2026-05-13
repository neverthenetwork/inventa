package web

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"

	"github.com/neverthenetwork/inventa/internal/config"
	"github.com/neverthenetwork/inventa/internal/datastore"
	"github.com/neverthenetwork/inventa/internal/spf"

	cy "gonum.org/v1/gonum/graph/formats/cytoscapejs"
)

// Server handles HTTP requests for the web UI.
type Server struct {
	StaticFS fs.FS
	Store    *datastore.TopologyStore
	Cfg      *config.Conf
	Logger   *slog.Logger
}

// IndexHandler serves the 2D cytoscape view.
func (s *Server) IndexHandler(w http.ResponseWriter, _ *http.Request) {
	content, err := fs.ReadFile(s.StaticFS, "web-dist/index.html")
	if err != nil {
		http.Error(w, "index not found", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprint(w, string(content))
}

// VRIndexHandler serves the VR 3D force graph view.
func (s *Server) VRIndexHandler(w http.ResponseWriter, _ *http.Request) {
	content, err := fs.ReadFile(s.StaticFS, "web-dist/vrindex.html")
	if err != nil {
		http.Error(w, "VR view not found", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprint(w, string(content))
}

// ThreeDIndexHandler serves the 3D force graph view.
func (s *Server) ThreeDIndexHandler(w http.ResponseWriter, _ *http.Request) {
	content, err := fs.ReadFile(s.StaticFS, "web-dist/3dindex.html")
	if err != nil {
		http.Error(w, "3D view not found", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprint(w, string(content))
}

// JsHandler returns topology data as JSON for Cytoscape.
func (s *Server) JsHandler(w http.ResponseWriter, r *http.Request) {
	var includeList []string
	var pathPairs []spf.PathSegment
	src := r.URL.Query().Get("src")
	dst := r.URL.Query().Get("dst")
	w.Header().Set("Content-Type", "application/json")

	if src != "" && dst != "" {
		elements := s.Store.Get()
		bestPathNames, err := spf.GetBestPathNames(elements, src, dst, s.Logger)
		if err != nil {
			http.Error(w, fmt.Sprintf("path computation error: %v", err), http.StatusInternalServerError)
			return
		}
		pathPairs = spf.GetPathSegments(bestPathNames)
		includeList = collapsePathPairs(pathPairs)
	}

	elements := s.Store.Get()
	filteredElements := cy.Elements{
		Nodes: make([]cy.Node, 0),
		Edges: make([]cy.Edge, 0),
	}
	filteredElements.Nodes = filterNodes(elements.Nodes, includeList)
	filteredElements.Edges = filterEdges(elements.Edges, pathPairs)
	if err := json.NewEncoder(w).Encode(filteredElements); err != nil {
		s.Logger.Error("encoding json response", "error", err)
	}
}

func filterNodes(nodes []cy.Node, includeList []string) []cy.Node {
	nodeList := make([]cy.Node, 0)
	for _, n := range nodes {
		if len(includeList) == 0 {
			n.Data.Attributes["show"] = true
			nodeList = append(nodeList, n)
		} else {
			_, found := config.FindInArray(n.Data.ID, includeList)
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
	edgeList := make([]cy.Edge, 0)
	if len(pathPairs) == 0 {
		for _, e := range edges {
			e.Data.Attributes["show"] = true
			edgeList = append(edgeList, e)
		}
	} else {
		for _, e := range edges {
			var found bool
			for _, pp := range pathPairs {
				if e.Data.Source == pp.Src && e.Data.Target == pp.Dst {
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

// collapsePathPairs returns a deduplicated list of all nodes in the path pairs.
func collapsePathPairs(pathPairs []spf.PathSegment) []string {
	var nodeList []string
	for _, pp := range pathPairs {
		if _, f := config.FindInArray(pp.Src, nodeList); !f {
			nodeList = append(nodeList, pp.Src)
		}
		if _, f := config.FindInArray(pp.Dst, nodeList); !f {
			nodeList = append(nodeList, pp.Dst)
		}
	}
	return nodeList
}

package spf

import (
	"github.com/RyanCarrier/dijkstra"
	cy "gonum.org/v1/gonum/graph/formats/cytoscapejs"
)

// PathSegment is a pair of nodes
type PathSegment struct {
	Src string
	Dst string
}

// BestPath is a list of nodes in a shortest path
type BestPath struct {
	Path []string
}

// BestPaths is a list of BestPath objects
type BestPaths struct {
	Paths []BestPath
}

func makeDijkstra(elements cy.Elements) (*dijkstra.Graph, error) {
	graph := dijkstra.NewGraph()
	for _, n := range elements.Nodes {
		graph.AddMappedVertex(n.Data.ID)
	}

	for _, v := range elements.Edges {
		if err := graph.AddMappedArc(v.Data.Source, v.Data.Target, 10); err != nil {
			return nil, err
		} // TODO add metric
	}

	return graph, nil
}

func makeNameList(graph *dijkstra.Graph, paths dijkstra.BestPath) []string {
	var names []string
	for _, p := range paths.Path {
		name, _ := graph.GetMapped(p)
		names = append(names, name)
	}
	return names
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

// GetBestPathNames finds the shortest path(s) from src to dst, converting them to names
func GetBestPathNames(elements cy.Elements, src string, dst string) (BestPaths, error) {
	graph, _ := makeDijkstra(elements)
	srcIdx, _ := graph.GetMapping(src)
	dstIdx, _ := graph.GetMapping(dst)
	bestPathList, err := graph.ShortestAll(srcIdx, dstIdx)
	if err != nil {
		return BestPaths{
			Paths: nil,
		}, err
	}

	bestPathNames := BestPaths{
		Paths: nil,
	}
	for _, b := range bestPathList {
		bp := BestPath{
			Path: makeNameList(graph, b),
		}
		bestPathNames.Paths = append(bestPathNames.Paths, bp)
	}

	return bestPathNames, nil
}

// GetPathSegments breaks a list of paths into a list of two-node segments
func GetPathSegments(paths BestPaths) []PathSegment {
	var pathPairs []PathSegment
	for _, b := range paths.Paths {
		prev := ""
		for idx, p := range b.Path {
			if idx == 0 {
				prev = p
			} else {
				srcName := prev
				dstName := p
				var ps = PathSegment{
					Src: srcName,
					Dst: dstName,
				}
				pathPairs = append(pathPairs, ps)
				prev = p
			}
		}
	}
	return pathPairs
}

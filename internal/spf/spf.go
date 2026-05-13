package spf

import (
	"log/slog"
	"strconv"

	"github.com/RyanCarrier/dijkstra"
	cy "gonum.org/v1/gonum/graph/formats/cytoscapejs"
)

// PathSegment is a pair of nodes.
type PathSegment struct {
	Src string
	Dst string
}

// BestPath is a list of nodes in a shortest path.
type BestPath struct {
	Path []string
}

// BestPaths is a list of BestPath objects.
type BestPaths struct {
	Paths []BestPath
}

func makeDijkstra(elements cy.Elements, logger *slog.Logger) (*dijkstra.Graph, error) {
	graph := dijkstra.NewGraph()
	for _, n := range elements.Nodes {
		logger.Debug("adding node", "id", n.Data.ID)
		graph.AddMappedVertex(n.Data.ID)
	}

	for _, v := range elements.Edges {
		var metricInt int64
		metricString, ok := v.Data.Attributes["igp_metric"]
		if !ok {
			metricInt = 10 // default metric
		} else {
			metric, err := strconv.ParseInt(metricString.(string), 10, 64)
			if err != nil {
				metricInt = 10
			} else {
				metricInt = metric
			}
		}
		logger.Debug("adding edge", "src", v.Data.Source, "dst", v.Data.Target, "metric", metricInt)
		if err := graph.AddMappedArc(v.Data.Source, v.Data.Target, metricInt); err != nil {
			return nil, err
		}
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

// FindNode checks whether a node exists in the list of nodes.
func FindNode(what string, where []cy.Node) (idx int, found bool) {
	for i, v := range where {
		if v.Data.ID == what {
			return i, true
		}
	}
	return 0, false
}

// GetBestPathNames finds the shortest path(s) from src to dst.
func GetBestPathNames(elements cy.Elements, src string, dst string, logger *slog.Logger) (BestPaths, error) {
	graph, err := makeDijkstra(elements, logger)
	if err != nil {
		return BestPaths{}, err
	}
	srcIdx, err := graph.GetMapping(src)
	if err != nil {
		return BestPaths{}, err
	}
	dstIdx, err := graph.GetMapping(dst)
	if err != nil {
		return BestPaths{}, err
	}
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

// GetPathSegments breaks a list of paths into two-node segments.
func GetPathSegments(paths BestPaths) []PathSegment {
	var pathPairs []PathSegment
	for _, b := range paths.Paths {
		prev := ""
		for idx, p := range b.Path {
			if idx == 0 {
				prev = p
			} else {
				ps := PathSegment{
					Src: prev,
					Dst: p,
				}
				pathPairs = append(pathPairs, ps)
				prev = p
			}
		}
	}
	return pathPairs
}

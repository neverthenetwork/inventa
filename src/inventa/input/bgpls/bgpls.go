package bgpls

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/shelson/inventa/src/inventa/datastore"
	"github.com/shelson/inventa/src/inventa/logging"
	"github.com/shelson/inventa/src/inventa/utils"

	api "github.com/osrg/gobgp/v3/api"
	"github.com/osrg/gobgp/v3/pkg/server"
	cy "gonum.org/v1/gonum/graph/formats/cytoscapejs"
	"google.golang.org/protobuf/encoding/protojson"
)

type nodeNLRI struct {
	nodeName      string
	localRouterID string
}

type linkNLRI struct {
	localRouterID    string
	remoteRouterID   string
	localIPv4Addr    string
	neighborIPv4Addr string
	igpMetric        uint32
	srAdjacencySid   uint32
}

type prefixEntry struct {
	prefix      string
	srPrefixSid uint32
}

type prefixNLRI struct {
	prefixesReachable []prefixEntry
}

// MakePeerConfiguration returns a peer configuration for gobgp
func MakePeerConfiguration(peerIPv4Address string, peerASN int) *api.Peer {
	return &api.Peer{
		Conf: &api.PeerConf{
			NeighborAddress: peerIPv4Address,
			PeerAsn:         uint32(peerASN),
		},
		ApplyPolicy: &api.ApplyPolicy{
			ImportPolicy: &api.PolicyAssignment{
				DefaultAction: api.RouteAction_ACCEPT,
			},
			ExportPolicy: &api.PolicyAssignment{
				DefaultAction: api.RouteAction_REJECT,
			},
		},
		AfiSafis: []*api.AfiSafi{
			{
				Config: &api.AfiSafiConfig{
					Family: &api.Family{
						Afi:  api.Family_AFI_LS,
						Safi: api.Family_SAFI_LS,
					},
					Enabled: true,
				},
			},
		},
	}
}

// ProcessBGPUpdates processes the updates, dumps the current table
// and updates our datastore
func ProcessBGPUpdates(r *api.WatchEventResponse, count int, s *server.BgpServer) {

	marshaller := protojson.MarshalOptions{
		Indent:        "  ",
		UseProtoNames: true,
	}

	if p := r.GetPeer(); p != nil && p.Type == api.WatchEventResponse_PeerEvent_STATE {
		logging.Log.Info(p)
	} else if t := r.GetTable(); t != nil {
		newNodeMap := map[string]nodeNLRI{}
		newReachMap := map[string]prefixNLRI{}
		newLinkTable := []linkNLRI{}

		family := &api.Family{Afi: api.Family_AFI_LS, Safi: api.Family_SAFI_LS}

		if count > 50 { // only start doing this once we're more stable
			if err := s.ListPath(context.Background(),
				&api.ListPathRequest{TableType: api.TableType_LOCAL, Family: family}, func(d *api.Destination) {
					for _, p := range d.Paths {
						var nlriType = "unknown"
						var nodeName string
						var nodelocalRouterID string
						var igpRouterID string
						var igpMetric uint32
						var localRouterID string
						var remoteRouterID string
						var localIPv4Addr string
						var neighborIPv4Addr string
						var prefixesReachable []string
						var srAdjacencySid uint32
						var srPrefixSid uint32

						m, _ := (p.Nlri).UnmarshalNew()
						switch m := m.(type) {
						case *api.LsAddrPrefix:
							n, _ := (m.Nlri).UnmarshalNew()
							switch n := n.(type) {
							case *api.LsNodeNLRI:
								nlriType = "LsnodeNLRI"
								igpRouterID = n.LocalNode.IgpRouterId
							case *api.LsLinkNLRI:
								nlriType = "LslinkNLRI"
								localRouterID = n.LocalNode.IgpRouterId
								remoteRouterID = n.RemoteNode.IgpRouterId
								localIPv4Addr = n.LinkDescriptor.InterfaceAddrIpv4
								neighborIPv4Addr = n.LinkDescriptor.NeighborAddrIpv4
							case *api.LsPrefixV4NLRI:
								nlriType = "LsPrefixV4NLRI"
								localRouterID = n.LocalNode.IgpRouterId
								prefixesReachable = n.PrefixDescriptor.IpReachability
							case *api.LsPrefixV6NLRI:
								nlriType = "LsPrefixV6NLRI"
								localRouterID = n.LocalNode.IgpRouterId
								prefixesReachable = n.PrefixDescriptor.IpReachability
							}
						}
						for _, pattr := range p.Pattrs {
							pattrObj, _ := pattr.UnmarshalNew()
							switch pattrObj := pattrObj.(type) {
							case *api.LsAttribute:
								nodeName = utils.StripUnwanted(pattrObj.Node.Name)
								nodelocalRouterID = pattrObj.Node.LocalRouterId
								igpMetric = pattrObj.Link.IgpMetric
								srAdjacencySid = pattrObj.Link.SrAdjacencySid
								srPrefixSid = pattrObj.Prefix.SrPrefixSid
							}
						}
						if nlriType == "LsnodeNLRI" {
							newNodeNLRI := nodeNLRI{nodeName: nodeName, localRouterID: nodelocalRouterID}
							newNodeMap[igpRouterID] = newNodeNLRI
						}
						if nlriType == "LslinkNLRI" {
							newLinkNLRI := linkNLRI{
								localRouterID:    localRouterID,
								remoteRouterID:   remoteRouterID,
								localIPv4Addr:    localIPv4Addr,
								neighborIPv4Addr: neighborIPv4Addr,
								igpMetric:        igpMetric,
								srAdjacencySid:   srAdjacencySid,
							}
							newLinkTable = append(newLinkTable, newLinkNLRI)
						}
						if nlriType == "LsPrefixV4NLRI" || nlriType == "LsPrefixV6NLRI" {
							reachablePrefixes := []prefixEntry{}
							for _, prefix := range prefixesReachable {
								newPrefix := prefixEntry{prefix: prefix, srPrefixSid: srPrefixSid}
								reachablePrefixes = append(reachablePrefixes, newPrefix)
							}
							newReachMap[localRouterID] = prefixNLRI{prefixesReachable: reachablePrefixes}
						}
						if nlriType == "unknown" {
							pathdump, _ := marshaller.Marshal(p.Nlri)
							logging.Log.Info(string(pathdump))
						}
					}
				}); err != nil {
				logging.Log.Fatal(err)
			}
			datastore.Elements = cy.Elements{
				Nodes: make([]cy.Node, 0),
				Edges: make([]cy.Edge, 0),
			}
			nodeGroups := []string{}
			for _, val := range newNodeMap {
				nodeGroup := "noGroup"
				if utils.Configs.GroupSplitChar != "" {
					nodeParts := strings.Split(val.nodeName, utils.Configs.GroupSplitChar)
					nodeGroup := nodeParts[utils.Configs.GroupSplitIndex]
					if _, found := utils.FindInArray(nodeGroup, nodeGroups); !found {
						nodeGroups = append(nodeGroups, nodeGroup)
					}
				} else {
					nodeGroups = []string{"NoGroup"}
				}
				nodeGroupIndex, _ := utils.FindInArray(nodeGroup, nodeGroups)
				node := cy.Node{
					Data: cy.NodeData{
						ID: val.nodeName,
						Attributes: map[string]interface{}{
							"label":   val.nodeName,
							"group":   nodeGroup,
							"cluster": nodeGroupIndex,
						},
					},
					Selectable: true,
				}
				datastore.Elements.Nodes = append(datastore.Elements.Nodes, node)
			}
			for _, link := range newLinkTable {
				edge := cy.Edge{
					Data: cy.EdgeData{
						ID:     fmt.Sprintf("%s:%s_%s", newNodeMap[link.localRouterID].nodeName, newNodeMap[link.remoteRouterID].nodeName, strconv.FormatInt(int64(link.srAdjacencySid), 10)),
						Source: newNodeMap[link.localRouterID].nodeName,
						Target: newNodeMap[link.remoteRouterID].nodeName,
						Attributes: map[string]interface{}{
							"adjacency_sid": strconv.FormatInt(int64(link.srAdjacencySid), 10),
							"igp_metric":    strconv.FormatInt(int64(link.igpMetric), 10),
						},
					},
					Selectable: true,
				}
				datastore.Elements.Edges = append(datastore.Elements.Edges, edge)
			}
		}
	}
}

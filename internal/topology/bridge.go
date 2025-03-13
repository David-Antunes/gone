package topology

import (
	"github.com/David-Antunes/gone/internal"
	"github.com/David-Antunes/gone/internal/network"
)

type Bridge struct {
	Id             string
	NetworkBridge  *network.Bridge
	Router         *Router
	RouterLink     *BiLink
	ConnectedNodes map[string]*Node
	NodeLinks      map[string]*BiLink
	MachineId      string
}

func (bridge *Bridge) ID() string {
	return bridge.Id
}

func (bridge *Bridge) AddNode(node *Node, link *BiLink) {
	bridge.ConnectedNodes[node.ID()] = node
	bridge.NodeLinks[node.ID()] = link
}

func (bridge *Bridge) RemoveNode(nodeId string) {

	if link, ok := bridge.NodeLinks[nodeId]; ok {
		n := bridge.ConnectedNodes[nodeId]
		bridge.NetworkBridge.RemoveNode([]byte(n.NetworkNode.GetMac()))
		link.NetworkBILink.Close()
		delete(bridge.ConnectedNodes, nodeId)
		delete(bridge.NodeLinks, nodeId)
	}
}

func (bridge *Bridge) SetRouter(router *Router, link *BiLink) {
	bridge.Router = router
	bridge.RouterLink = link
}
func (bridge *Bridge) RemoveRouter() {
	if bridge.Router != nil {
		bridge.Router = nil
		bridge.RouterLink.NetworkBILink.Close()
		bridge.RouterLink = nil
		bridge.NetworkBridge.SetGateway(internal.GetNullChan())
	}
}

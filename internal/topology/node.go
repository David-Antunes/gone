package topology

import "github.com/David-Antunes/gone/internal/network"

type Node struct {
	Id          string
	NetworkNode *network.Node
	Link        *BiLink
	Bridge      *Bridge
	MachineId   string
}

func (node *Node) ID() string {
	return node.Id
}

func (node *Node) SetBridge(bridge *Bridge, link *BiLink) *Node {
	node.Bridge = bridge
	node.Link = link
	return node
}

// Clears the data regarding a topology.Bridge. Doesn't affect the networkNode
func (node *Node) RemoveBridge() {

	if node.Bridge != nil {
		node.Link.NetworkBILink.Close()
		node.NetworkNode.SetLink(nil)
		node.Bridge.RemoveNode(node.Id)
		node.Bridge = nil
	}

}

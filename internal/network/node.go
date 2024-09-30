package network

import "github.com/David-Antunes/gone-proxy/xdp"

type Node struct {
	macAddr  string
	incoming chan *xdp.Frame
	outgoing chan *xdp.Frame
	link     *BiLink
}

func CreateNode(macAddr string, incoming chan *xdp.Frame, outgoing chan *xdp.Frame, link *BiLink) *Node {
	return &Node{macAddr: macAddr, incoming: incoming, outgoing: outgoing, link: link}
}

func (node *Node) GetIncoming() chan *xdp.Frame {
	return node.incoming
}

func (node *Node) GetOutgoing() chan *xdp.Frame {
	return node.outgoing
}

func (node *Node) GetMac() string {
	return node.macAddr
}

func (node *Node) GetLink() *BiLink {
	return node.link
}

func (node *Node) SetLink(link *BiLink) {
	node.link = link
}

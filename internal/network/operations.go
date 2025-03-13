package network

import (
	"github.com/David-Antunes/gone-proxy/xdp"
	"github.com/David-Antunes/gone/internal"
)

type disruptLogic struct {
	disrupted bool
	ctx       chan struct{}
}

func ConnectNodeToBridge(node *Node, bridge *Bridge, props LinkProps) *BiLink {

	bridgeOutgoingChannel := make(chan *xdp.Frame, internal.QueueSize)
	toLink := CreateLink(node.incoming, bridge.incomingChannel, props)
	fromLink := CreateLink(bridgeOutgoingChannel, node.outgoing, props)
	link := CreateBILink(toLink, fromLink)
	node.SetLink(link)
	bridge.AddNode([]byte(node.macAddr), bridgeOutgoingChannel)
	return link
}

func ConnectBridgeToRouter(bridge *Bridge, router *Router, props LinkProps) *BiLink {

	gateway := make(chan *xdp.Frame, internal.QueueSize)
	bridgeChannel := make(chan *xdp.Frame, internal.QueueSize)
	bridge.SetGateway(gateway)

	toLink := CreateLink(bridge.gateway, router.incomingChannel, props)
	fromLink := CreateLink(bridgeChannel, bridge.incomingChannel, props)
	link := CreateBILink(toLink, fromLink)

	bridge.SetLink(link)

	return link
}

func ConnectRouterToRouter(router1 *Router, router2 *Router, props LinkProps) *BiLink {
	router1_to_router2_channel := make(chan *xdp.Frame, internal.QueueSize)
	router2_to_router1_channel := make(chan *xdp.Frame, internal.QueueSize)

	to_link := CreateLink(router1_to_router2_channel, router2.incomingChannel, props)
	from_link := CreateLink(router2_to_router1_channel, router1.incomingChannel, props)
	BI_link := CreateBILink(to_link, from_link)

	return BI_link
}

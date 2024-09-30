package routing

import "github.com/David-Antunes/gone-proxy/xdp"

/*
Routing package to expose an entrypoint to insert abstracted logic
to manage routing events in the network router.

This package starts with a dummy protocol which does nothing.

To insert a new protocol, implement the protocol interface and then
override the routing with SetRouting method.
*/
type protocol interface {
	HandleNewMac(*xdp.Frame, string)
}

var handleRouting protocol

func Init() {
	SetRouting(createDummyRouting())
}

func SetRouting(proto protocol) {
	handleRouting = proto
}

func HandleNewMac(frame *xdp.Frame, routerId string) {
	handleRouting.HandleNewMac(frame, routerId)
}

package routing

import "github.com/David-Antunes/gone-proxy/xdp"

/*
Dummy routing logic. This interface is placeholder until a complete implementation
of proper routing protocol is set.
*/
type dummyRouting struct {
}

func (d dummyRouting) HandleNewMac(frame *xdp.Frame, routerId string) {
}

func createDummyRouting() dummyRouting {
	return dummyRouting{}
}

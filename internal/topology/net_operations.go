package topology

import (
	"github.com/David-Antunes/gone-proxy/xdp"
)

// TODO Add route to the other router
// TODO Update weight to the real weight
func AddNewMacBetweenRouters(src *Router, dest *Router, mac string, weight int) int {
	biLink := src.RouterLinks[dest.ID()]
	if dest.MachineId != src.MachineId {
		src.Weights[mac] = Weight{Router: dest.ID(), Weight: weight}
		src.NetworkRouter.AddNode([]byte(mac), biLink.ConnectsTo.NetworkLink.GetOriginChan())
	} else {
		src.Weights[mac] = Weight{Router: dest.ID(), Weight: weight}
		src.NetworkRouter.AddNode([]byte(mac), GetOriginChanFromLink(src.ID(), biLink))
	}
	return weight - biLink.ConnectsTo.NetworkLink.GetProps().Weight
}

func GetOriginChanFromLink(src string, link *BiLink) chan *xdp.Frame {
	if link.From.ID() == src {
		return link.ConnectsFrom.NetworkLink.GetOriginChan()
	} else {
		return link.ConnectsTo.NetworkLink.GetOriginChan()
	}
}

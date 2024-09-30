package topology

import "github.com/David-Antunes/gone/internal/network"

type Router struct {
	Id               string
	NetworkRouter    *network.Router
	ConnectedRouters map[string]*Router
	RouterLinks      map[string]*BiLink
	ConnectedBridges map[string]*Bridge
	BridgeLinks      map[string]*BiLink
	Weights          map[string]Weight
	MachineId        string
}

type Weight struct {
	Router string
	Weight int
}

func (router *Router) ID() string {
	return router.Id
}

func (router *Router) AddBridge(bridge *Bridge, link *BiLink) {
	router.ConnectedBridges[bridge.ID()] = bridge
	router.BridgeLinks[bridge.ID()] = link
}

func (router *Router) RemoveBridge(bridgeId string) {

	if link, ok := router.BridgeLinks[bridgeId]; ok {
		link.NetworkBILink.Stop()
		delete(router.ConnectedBridges, bridgeId)
		delete(router.BridgeLinks, bridgeId)
	}
}

func (router *Router) RemoveRouter(routerId string) {

	if link, ok := router.RouterLinks[routerId]; ok {
		link.NetworkBILink.Stop()
		delete(router.ConnectedRouters, routerId)
		delete(router.RouterLinks, routerId)
	}
}

func (router *Router) AddRouter(r *Router, link *BiLink) {
	router.ConnectedRouters[r.ID()] = r
	router.RouterLinks[r.ID()] = link
}

func (router *Router) AddWeight(mac string, routerId string, weight int) {
	router.Weights[mac] = Weight{Router: routerId, Weight: weight}
}

func (router *Router) RemoveWeight(mac string) {
	delete(router.Weights, mac)
}

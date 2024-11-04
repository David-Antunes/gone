package application

import (
	"fmt"
	"github.com/David-Antunes/gone/api"
	"github.com/David-Antunes/gone/internal/network"
	"github.com/David-Antunes/gone/internal/topology"
)

const _REMOTE_QUEUESIZE = 1000

func sniffSocketPath(id string) string {
	return "/tmp/" + id + ".sniff"
}
func interceptSocketPath(id string) string {
	return "/tmp/" + id + ".intercept"
}

//func getInterceptId(id string, direction bool) string {
//	if direction {
//		return id + "-tx"
//	} else {
//		return id + "-rx"
//	}
//}

func isSpecialLink(link *topology.Link) bool {
	if link.NetworkLink != nil {
		_, ok := link.NetworkLink.GetShaper().(*network.NetworkShaper)
		return !ok
	}
	return true
}

func convertToAPINode(n *topology.Node) api.Node {
	b := ""
	link := &api.Link{
		To:        "",
		From:      "",
		LinkProps: api.LinkProps{},
	}
	if n.Bridge != nil {
		b = n.Bridge.ID()
		link = &api.Link{
			To:   n.Link.To.ID(),
			From: n.Link.From.ID(),
			LinkProps: api.LinkProps{
				Latency:   int(n.Link.NetworkBILink.Left.GetProps().Latency),
				Bandwidth: n.Link.NetworkBILink.Left.GetProps().Bandwidth,
				Jitter:    n.Link.NetworkBILink.Left.GetProps().Jitter,
				DropRate:  n.Link.NetworkBILink.Left.GetProps().DropRate,
				Weight:    n.Link.NetworkBILink.Left.GetProps().Weight,
			},
		}
	}

	a := api.Node{
		Id:        n.Id,
		MachineId: n.MachineId,
		Bridge:    b,
		Link:      *link,
	}
	fmt.Println(a)
	return a
}

func convertToAPIBridge(b *topology.Bridge) api.Bridge {
	nodes := make([]api.Node, 0, len(b.ConnectedNodes))

	for _, n := range b.ConnectedNodes {
		nodes = append(nodes, convertToAPINode(n))
	}
	r := ""
	link := &api.Link{}
	if b.Router != nil {
		r = b.Router.ID()
		link = &api.Link{
			To:   b.Router.Id,
			From: b.ID(),
			LinkProps: api.LinkProps{
				Latency:   int(b.RouterLink.NetworkBILink.Left.GetProps().Latency),
				Bandwidth: b.RouterLink.NetworkBILink.Left.GetProps().Bandwidth,
				Jitter:    b.RouterLink.NetworkBILink.Left.GetProps().Jitter,
				DropRate:  b.RouterLink.NetworkBILink.Left.GetProps().DropRate,
				Weight:    b.RouterLink.NetworkBILink.Left.GetProps().Weight,
			},
		}
	}
	return api.Bridge{
		Id:        b.Id,
		MachineId: b.MachineId,
		Router:    r,
		Link:      *link,
		Nodes:     nodes,
	}
}

func convertToAPIRouter(r *topology.Router) api.Router {
	links := make(map[string]api.Link)

	for k, _ := range r.ConnectedRouters {
		links[k] = api.Link{
			To:   r.RouterLinks[k].To.ID(),
			From: r.RouterLinks[k].From.ID(),
			LinkProps: api.LinkProps{
				Latency:   int(r.RouterLinks[k].NetworkBILink.Left.GetProps().Latency),
				Bandwidth: r.RouterLinks[k].NetworkBILink.Left.GetProps().Bandwidth,
				Jitter:    r.RouterLinks[k].NetworkBILink.Left.GetProps().Jitter,
				DropRate:  r.RouterLinks[k].NetworkBILink.Left.GetProps().DropRate,
				Weight:    r.RouterLinks[k].NetworkBILink.Left.GetProps().Weight,
			},
		}
	}

	bridges := make([]api.Bridge, 0, len(r.ConnectedBridges))

	for k, _ := range r.ConnectedBridges {
		bridges = append(bridges, convertToAPIBridge(r.ConnectedBridges[k]))
	}

	weights := make(map[string]map[string]int)
	weights[r.Id] = make(map[string]int)

	for k, _ := range r.ConnectedRouters {
		weights[k] = make(map[string]int)
	}

	for k, v := range r.Weights {
		weights[v.Router][k] = v.Weight
	}

	return api.Router{
		Id:        r.Id,
		MachineId: r.MachineId,
		Routers:   links,
		Bridges:   bridges,
		Weights:   weights,
	}
}

package application

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/David-Antunes/gone-proxy/xdp"
	api "github.com/David-Antunes/gone/api/Add"
	connectApi "github.com/David-Antunes/gone/api/Connect"
	disconnectApi "github.com/David-Antunes/gone/api/Disconnect"
	opApi "github.com/David-Antunes/gone/api/Operations"
	removeApi "github.com/David-Antunes/gone/api/Remove"
	internal "github.com/David-Antunes/gone/internal/api"
	"github.com/David-Antunes/gone/internal/cluster"
	"github.com/David-Antunes/gone/internal/docker"
	"github.com/David-Antunes/gone/internal/graphDB"
	"github.com/David-Antunes/gone/internal/network"
	"github.com/David-Antunes/gone/internal/proxy"
	redirecttraffic "github.com/David-Antunes/gone/internal/redirect-traffic"
	"github.com/David-Antunes/gone/internal/topology"
	"net"
	"net/http"
	"strings"
	"time"
)

type Leader struct {
	cl         *cluster.Cluster
	dm         *docker.DockerManager
	proxy      *proxy.Proxy
	topo       *topology.Topology
	icm        *cluster.InterCommunicationManager
	rm         *RttManager
	sniffers   map[string]*redirecttraffic.RedirectionSocket
	intercepts map[string]*redirecttraffic.RedirectionSocket
}

func NewLeader(cl *cluster.Cluster, dm *docker.DockerManager, proxy *proxy.Proxy, icm *cluster.InterCommunicationManager, rm *RttManager) *Leader {
	return &Leader{
		cl:         cl,
		dm:         dm,
		proxy:      proxy,
		topo:       topology.CreateTopology(dm.GetMachineId(), proxy, rm.GetRtt()),
		icm:        icm,
		rm:         rm,
		sniffers:   make(map[string]*redirecttraffic.RedirectionSocket),
		intercepts: make(map[string]*redirecttraffic.RedirectionSocket),
	}
}

func (app *Leader) GetMachineId() string {
	return app.dm.GetMachineId()
}

func (app *Leader) GetNode(id string) (*topology.Node, bool) {
	return app.topo.GetNode(id)
}
func (app *Leader) GetBridge(id string) (*topology.Bridge, bool) {
	return app.topo.GetBridge(id)
}
func (app *Leader) GetRouter(id string) (*topology.Router, bool) {
	return app.topo.GetRouter(id)
}

func (app *Leader) GetRouterWeights(id string) map[string]topology.Weight {
	r, _ := app.topo.GetRouter(id)
	return r.Weights
}

func (app *Leader) specialLinkCleanup(link *network.BiLink) {

	if link.Left != nil {

		if s, ok := link.Left.GetShaper().(*network.SniffShaper); ok {
			delete(app.sniffers, s.GetRtID())
			return
		}
		if s, ok := link.Right.GetShaper().(*network.SniffShaper); ok {
			delete(app.sniffers, s.GetRtID())
			return
		}
	}
	if link.Right != nil {

		if s, ok := link.Left.GetShaper().(*network.InterceptShaper); ok {
			delete(app.intercepts, s.GetRtID())
			return
		}
		if s, ok := link.Right.GetShaper().(*network.InterceptShaper); ok {
			delete(app.intercepts, s.GetRtID())
			return
		}
	}
}

func (app *Leader) HandleNewMac(frame *xdp.Frame, routerId string) {

	dest := frame.MacDestination

	r, _ := app.topo.GetRouter(routerId)

	path, distance := graphDB.FindPathToRouter(routerId, dest)
	fmt.Println(routerId, ":", net.HardwareAddr(dest), ":", path)

	if len(path) > 0 {
		if net.HardwareAddr(dest).String() == path[len(path)-1] {
			path = path[:len(path)-1]
		}
		app.topo.InsertNewPath(path, frame, distance)
		r.NetworkRouter.InjectFrame(frame)
	}
}

func (app *Leader) execInMachine(machineId string, dockerCmd []string) (string, string, string, error) {
	body := &api.AddNodeRequest{
		DockerCmd: dockerCmd,
		MachineId: machineId,
	}

	resp, err := app.cl.SendMsg(machineId, body, "addNode")

	if err != nil {
		return "", "", "", err
	}

	d := json.NewDecoder(resp.Body)

	result := &api.AddNodeResponse{}
	err = d.Decode(&result)
	if err != nil {
		return "", "", "", err
	}

	if result.Error.ErrCode != 0 {
		return "", "", "", errors.New(result.Error.ErrMsg)
	}

	return result.Id, result.Mac, result.Ip, err
}

func (app *Leader) AddNode(machineId string, dockerCmd []string) (string, string, string, error) {
	if !app.cl.Contains(machineId) {
		return "", "", "", errors.New("invalid machine id")
	}

	if machineId != "" && app.dm.GetMachineId() != machineId {
		id, mac, ip, err := app.execInMachine(machineId, dockerCmd)
		if err != nil {
			return "", "", "", err
		}
		return id, mac, ip, nil
	}

	id, mac, ip, err := app.dm.ExecContainer(dockerCmd)
	if err != nil {
		return "", "", "", err
	}
	err = app.dm.RegisterContainer(machineId, id, mac, ip)
	if err != nil {
		return "", "", "", err
	}

	err = app.dm.BootstrapContainer(id)
	if err != nil {
		return "", "", "", err
	}

	err = app.dm.PropagateArp(ip, mac)
	if err != nil {
		return "", "", "", err
	}

	_, err = app.cl.Broadcast(&internal.RegisterNodeRequest{
		Id:        id,
		Ip:        ip,
		Mac:       mac,
		MachineId: machineId,
	}, http.MethodPost, "registerNode")

	_, err = app.topo.RegisterNode(id, mac, machineId)
	if err != nil {
		return "", "", "", err
	}

	return id, mac, ip, nil
}

func (app *Leader) RegisterNode(id string, mac string, ip string, machineId string) error {

	if !app.cl.Contains(machineId) {
		return errors.New("invalid machine id")
	}
	err := app.dm.RegisterContainer(machineId, id, mac, ip)
	if err != nil {
		return err
	}

	err = app.dm.PropagateArp(ip, mac)
	if err != nil {
		return err
	}

	_, err = app.topo.RegisterNode(id, mac, machineId)
	if err != nil {
		return err
	}
	return nil
}

func (app *Leader) AddBridge(machineId string, id string) (string, error) {

	if !app.cl.Contains(machineId) {
		return "", errors.New("invalid machine id")
	}

	_, err := app.topo.RegisterBridge(id, machineId)
	if err != nil {
		return "", err
	}

	return "", nil
}

func (app *Leader) AddRouter(machineId string, id string) (string, error) {

	if !app.cl.Contains(machineId) {
		return "", errors.New("invalid machine id")
	}

	r, err := app.topo.RegisterRouter(id, machineId)
	if err != nil {
		return "", err
	}

	if r.MachineId == app.GetMachineId() {
		graphDB.AddRouter(id)
	}

	return "", nil

}

func (app *Leader) ConnectNodeToBridge(nodeID string, bridgeID string, linkProps network.LinkProps) error {

	var n *topology.Node
	var b *topology.Bridge
	var ok bool

	if n, ok = app.topo.GetNode(nodeID); !ok {
		return errors.New(nodeID + " ID doesn't exist")
	}

	if b, ok = app.topo.GetBridge(bridgeID); !ok {
		return errors.New(bridgeID + " ID doesn't exist")
	}
	if n.MachineId != b.MachineId {
		return errors.New("can't connect a node and bridge in different machines")
	}
	if n.MachineId == app.GetMachineId() {
		_, err := app.topo.ConnectNodeToBridge(nodeID, bridgeID, linkProps)
		if err != nil {
			return err
		}
		if b.Router != nil {
			graphDB.AddNode(n.NetworkNode.GetMac(), b.Router.ID())
			fmt.Println("Added", nodeID, "to router from", bridgeID, "to router", b.Router.ID())
		}
		return nil
	} else {
		body := &connectApi.ConnectNodeToBridgeRequest{
			Node:      nodeID,
			Bridge:    bridgeID,
			Latency:   int(linkProps.Latency/time.Millisecond) * 2,
			Jitter:    linkProps.Jitter,
			DropRate:  linkProps.DropRate,
			Bandwidth: linkProps.Bandwidth,
			Weight:    linkProps.Weight,
		}

		resp, err := app.cl.SendMsg(n.MachineId, body, "connectNodeToBridge")
		if err != nil {
			return err
		}
		d := json.NewDecoder(resp.Body)
		req := &connectApi.ConnectNodeToBridgeResponse{}
		err = d.Decode(&req)
		if err != nil {
			return err
		}
		if req.Error.ErrCode != 0 {
			return errors.New(req.Error.ErrMsg)
		}
	}

	return nil
}

func (app *Leader) ConnectBridgeToRouter(bridgeID string, routerID string, linkProps network.LinkProps) error {

	var b *topology.Bridge
	var r *topology.Router
	var ok bool

	if b, ok = app.topo.GetBridge(bridgeID); !ok {
		return errors.New(bridgeID + " ID doesn't exist")
	}

	if r, ok = app.topo.GetRouter(routerID); !ok {
		return errors.New(bridgeID + " ID doesn't exist")
	}
	if b.MachineId != r.MachineId {
		return errors.New("can't connect a bridge and a router in different machines")
	}

	if app.GetMachineId() == b.MachineId {
		_, err := app.topo.ConnectBridgeToRouter(bridgeID, routerID, linkProps)

		if err != nil {
			return err
		}

		for _, netNode := range b.ConnectedNodes {
			graphDB.AddNode(netNode.NetworkNode.GetMac(), routerID)
			fmt.Println("Added", netNode.ID(), "to router", routerID)
		}
	} else {

		body := &connectApi.ConnectBridgeToRouterRequest{
			Bridge:    bridgeID,
			Router:    routerID,
			Latency:   int(linkProps.Latency/time.Millisecond) * 2,
			Jitter:    linkProps.Jitter,
			DropRate:  linkProps.DropRate,
			Bandwidth: linkProps.Bandwidth,
			Weight:    linkProps.Weight,
		}

		resp, err := app.cl.SendMsg(b.MachineId, body, "connectBridgeToRouter")
		if err != nil {
			return err
		}

		d := json.NewDecoder(resp.Body)
		req := &connectApi.ConnectBridgeToRouterResponse{}
		err = d.Decode(&req)
		if err != nil {
			return err
		}
		if req.Error.ErrCode != 0 {
			return errors.New(req.Error.ErrMsg)
		}
	}
	return nil
}

func (app *Leader) ConnectRouterToRouter(router1ID string, router2ID string, linkProps network.LinkProps) error {

	r1, ok := app.topo.GetRouter(router1ID)

	if !ok {
		return errors.New("router not found")
	}
	r2, ok := app.topo.GetRouter(router2ID)

	if !ok {
		return errors.New("router not found")
	}

	if r1.MachineId == app.GetMachineId() {
		if r2.MachineId == app.GetMachineId() {
			l, err := app.topo.ConnectRouterToRouterLocal(router1ID, router2ID, linkProps)

			if err != nil {
				return err
			}

			graphDB.AddPath(router1ID, router2ID, l.ID(), linkProps.Weight)

			app.PropagateNewRoutes(r1)
			return nil
		} else {
			return app.connectRouterToRouterRemote(r1, r2, linkProps)
		}
	} else if r2.MachineId == app.GetMachineId() {
		return app.connectRouterToRouterRemote(r2, r1, linkProps)
	} else {
		return app.RedirectConnection(r1, r2, linkProps)
	}
}

func (app *Leader) connectRouterToRouterRemote(r1 *topology.Router, r2 *topology.Router, linkProps network.LinkProps) error {

	if _, ok := r1.ConnectedRouters[r2.ID()]; ok {
		return errors.New(r1.ID() + " is already connected to " + r2.ID())
	}

	if _, ok := r2.ConnectedRouters[r2.ID()]; ok {
		return errors.New(r1.ID() + " is already connected to " + r2.ID())
	}

	router1Channel := make(chan *xdp.Frame, _REMOTE_QUEUESIZE)
	conn := app.cl.Endpoints[r2.MachineId]
	app.icm.AddConnection(r2.ID(), conn, r1.ID(), r1.NetworkRouter)
	toLink := network.CreateLink(router1Channel, nil, linkProps)
	topoLink := &topology.Link{
		Id:          r1.ID() + "-RemoteLink-" + r2.ID(),
		NetworkLink: toLink,
		From:        r1,
		To:          r2,
	}

	BiLink := &topology.BiLink{
		Id:            r1.ID() + "-RemoteBiLink-" + r2.ID(),
		NetworkBILink: nil,
		ConnectsTo:    topoLink,
		ConnectsFrom:  nil,
		To:            r2,
		From:          r1,
	}

	r1.AddRouter(r2, BiLink)
	r2.AddRouter(r1, BiLink)

	s := network.CreateRemoteShaper(r2.ID(), r1.ID(), router1Channel, app.icm.GetoutQueue(), linkProps)
	d, _ := app.cl.GetNodeDelay(r2.MachineId)
	s.SetDelay(d)
	toLink.SetShaper(s)
	toLink.Start()
	b := &internal.ConnectRouterToRouterRequest{
		R1:        r2.ID(),
		R2:        r1.ID(),
		MachineID: r1.MachineId,
		Latency:   linkProps.Latency * 2,
		Jitter:    linkProps.Jitter,
		DropRate:  linkProps.DropRate,
		Bandwidth: linkProps.Bandwidth,
		Weight:    linkProps.Weight,
	}
	_, err := app.cl.SendMsg(r2.MachineId, b, "connectRouterToRouterRemote")
	if err != nil {
		return errors.New("couldn't contact machine")
	}

	graphDB.AddPath(r1.ID(), r2.ID(), BiLink.ID(), linkProps.Weight)
	app.PropagateNewRoutes(r1)
	return nil
}

func (app *Leader) RedirectConnection(r1 *topology.Router, r2 *topology.Router, linkProps network.LinkProps) error {

	b := &internal.ConnectRouterToRouterRequest{
		R1:        r1.ID(),
		R2:        r2.ID(),
		MachineID: r2.MachineId,
		Latency:   linkProps.Latency * 2,
		Jitter:    linkProps.Jitter,
		DropRate:  linkProps.DropRate,
		Bandwidth: linkProps.Bandwidth,
		Weight:    linkProps.Weight,
	}
	resp, err := app.cl.SendMsg(r1.MachineId, b, "connectRouterToRouter")
	if err != nil {
		return errors.New("couldn't contact machine")
	}

	d := json.NewDecoder(resp.Body)
	req := &internal.ConnectRouterToRouterResponse{}
	err = d.Decode(&req)

	if err != nil {
		return errors.New("couldn't decode weights response")
	}
	return nil
}

func (app *Leader) PropagateNewRoutes(r *topology.Router) {

	visitedRouters := make(map[string]*topology.Router, app.topo.GetRouterNumber())

	toVisit := make([]*topology.Router, 0)

	toVisit = append(toVisit, r)

	for _, router := range r.ConnectedRouters {
		if router.MachineId == app.GetMachineId() {
			toVisit = append(toVisit, router)
		}
	}

	for len(toVisit) > 0 {

		currRouter := toVisit[0]
		toVisit = toVisit[1:]
		if _, ok := visitedRouters[currRouter.ID()]; !ok {

			for _, adjacentRouter := range currRouter.ConnectedRouters {
				if adjacentRouter.MachineId != app.GetMachineId() {
					err := app.TradeRoutesRemote(currRouter, adjacentRouter)
					if err != nil {
						fmt.Println("Failed to trade routes")
					}
				} else {
					app.TradeRoutes(currRouter, adjacentRouter)
					toVisit = append(toVisit, adjacentRouter)
				}
			}
			visitedRouters[currRouter.ID()] = currRouter
		}
	}
}

func (app *Leader) TradeRoutes(r1 *topology.Router, r2 *topology.Router) {

	biLink := r1.RouterLinks[r2.ID()]
	newWeight := biLink.NetworkBILink.Left.GetProps().Weight

	for mac, weight := range r1.Weights {

		if existingWeight, ok := r2.Weights[mac]; ok && newWeight+weight.Weight < existingWeight.Weight {
			r2.Weights[mac] = topology.Weight{Router: r1.ID(), Weight: newWeight + weight.Weight}
			r2.NetworkRouter.AddNode([]byte(mac), topology.GetOriginChanFromLink(r2.ID(), biLink))
			fmt.Println(r2.ID(), "updated weight of", net.HardwareAddr(mac), "from", r1.ID(), "with weight", newWeight+weight.Weight, "before:", existingWeight)
		} else if _, ok = r2.Weights[mac]; !ok {
			r2.Weights[mac] = topology.Weight{Router: r1.ID(), Weight: newWeight + weight.Weight}
			r2.NetworkRouter.AddNode([]byte(mac), topology.GetOriginChanFromLink(r2.ID(), biLink))
			fmt.Println(r2.ID(), "added weight of", net.HardwareAddr(mac), "from", r1.ID(), "with weight", newWeight+weight.Weight)
		}
	}

	for mac, weight := range r2.Weights {

		if existingWeight, ok := r1.Weights[mac]; ok && newWeight+weight.Weight < existingWeight.Weight {
			r1.Weights[mac] = topology.Weight{Router: r2.ID(), Weight: newWeight + weight.Weight}
			r1.NetworkRouter.AddNode([]byte(mac), topology.GetOriginChanFromLink(r1.ID(), biLink))
			fmt.Println(r1.ID(), "updated weight of", net.HardwareAddr(mac), "from", r2.ID(), "with weight", newWeight+weight.Weight, "before:", existingWeight)
		} else if _, ok = r1.Weights[mac]; !ok {
			r1.Weights[mac] = topology.Weight{Router: r2.ID(), Weight: newWeight + weight.Weight}
			r1.NetworkRouter.AddNode([]byte(mac), topology.GetOriginChanFromLink(r1.ID(), biLink))
			fmt.Println(r1.ID(), "added weight of", net.HardwareAddr(mac), "from", r2.ID(), "with weight", newWeight+weight.Weight)
		}
	}
}

func (app *Leader) TradeRoutesRemote(r1 *topology.Router, r2 *topology.Router) error {
	b := &internal.GetRouterWeightsRequest{
		Router: r2.ID(),
	}
	resp, err := app.cl.SendMsg(r2.MachineId, b, "weights")
	if err != nil {
		return errors.New("couldn't contact machine")
	}

	d := json.NewDecoder(resp.Body)
	req := &internal.GetRouterWeightsResponse{}
	err = d.Decode(&req)

	if err != nil {
		return errors.New("couldn't decode weights response")
	}

	biLink := r1.RouterLinks[r2.ID()]
	newWeight := biLink.ConnectsTo.NetworkLink.GetProps().Weight

	for mac, weight := range req.Weights {

		if existingWeight, ok := r1.Weights[mac]; ok && newWeight+weight.Weight < existingWeight.Weight {
			r1.Weights[mac] = topology.Weight{Router: r2.ID(), Weight: newWeight + weight.Weight}
			r1.NetworkRouter.AddNode([]byte(mac), biLink.ConnectsTo.NetworkLink.GetOriginChan())
			fmt.Println(r1.ID(), "updated weight of", net.HardwareAddr(mac), "from", r2.ID(), "with weight", newWeight+weight.Weight, "before:", existingWeight)
		} else if _, ok = r1.Weights[mac]; !ok {
			r1.Weights[mac] = topology.Weight{Router: r2.ID(), Weight: newWeight + weight.Weight}
			r1.NetworkRouter.AddNode([]byte(mac), biLink.ConnectsTo.NetworkLink.GetOriginChan())
			fmt.Println(r1.ID(), "added weight of", net.HardwareAddr(mac), "from", r2.ID(), "with weight", newWeight+weight.Weight)
		}
	}

	body := &internal.TradeRoutesRequest{
		To:      r2.ID(),
		From:    r1.ID(),
		Weights: r1.Weights,
	}

	resp, err = app.cl.SendMsg(r2.MachineId, body, "trade")
	if err != nil {
		return errors.New("couldn't contact machine")
	}

	d = json.NewDecoder(resp.Body)
	tradeReq := &internal.TradeRoutesResponse{}
	err = d.Decode(&tradeReq)

	if err != nil {
		return errors.New("couldn't decode weights response")
	}
	return nil
}

func (app *Leader) ApplyRoutes(to string, from string, weights map[string]topology.Weight) {
	r, ok := app.topo.GetRouter(to)

	if !ok {
		return
	}

	biLink := r.RouterLinks[from]
	newWeight := biLink.ConnectsTo.NetworkLink.GetProps().Weight
	for mac, weight := range weights {

		if existingWeight, ok := r.Weights[mac]; ok && newWeight+weight.Weight < existingWeight.Weight {
			r.Weights[mac] = topology.Weight{Router: from, Weight: newWeight + weight.Weight}
			r.NetworkRouter.AddNode([]byte(mac), biLink.ConnectsTo.NetworkLink.GetOriginChan())
			fmt.Println(r.ID(), "updated weight of", net.HardwareAddr(mac), "from", from, "with weight", newWeight+weight.Weight, "before:", existingWeight)
		} else if _, ok = r.Weights[mac]; !ok {
			r.Weights[mac] = topology.Weight{Router: from, Weight: newWeight + weight.Weight}
			r.NetworkRouter.AddNode([]byte(mac), biLink.ConnectsTo.NetworkLink.GetOriginChan())
			fmt.Println(r.ID(), "added weight of", net.HardwareAddr(mac), "from", from, "with weight", newWeight+weight.Weight)
		}
	}

}
func (app *Leader) RemoveNode(nodeId string) error {

	n, ok := app.topo.GetNode(nodeId)

	if !ok {
		return errors.New(nodeId + " ID doesn't exist")
	}

	if n.MachineId == app.GetMachineId() {
		graphDB.RemoveNode(n.NetworkNode.GetMac())
	}

	link := n.Link

	_, err := app.topo.RemoveNode(nodeId)

	if err != nil {
		return err
	}
	if link != nil {
		app.specialLinkCleanup(link.NetworkBILink)
	}

	if n.MachineId == app.GetMachineId() {
		err = app.dm.RemoveNode(nodeId)
		if err != nil {
			return err
		}
	}

	if n.MachineId == app.GetMachineId() {

		body := &internal.ClearNodeRequest{
			Id: nodeId,
		}

		resp, err := app.cl.Broadcast(body, http.MethodPost, "clearNode")
		if err != nil {
			return err
		}

		for _, result := range resp {

			d := json.NewDecoder(result.Body)
			req := &internal.ClearNodeResponse{}
			err = d.Decode(&req)

			if err != nil {
				return err
			}

			if req.Error.ErrCode != 0 {
				return errors.New(req.Error.ErrMsg)
			}
		}

		return nil
	}

	if n.MachineId != app.GetMachineId() {

		body := &removeApi.RemoveNodeRequest{
			Name: nodeId,
		}

		resp, err := app.cl.SendMsg(n.MachineId, body, "removeNode")
		if err != nil {
			return err
		}

		d := json.NewDecoder(resp.Body)
		req := &removeApi.RemoveNodeResponse{}
		err = d.Decode(&req)

		if err != nil {
			return err
		}

		if req.Error.ErrCode != 0 {
			return errors.New(req.Error.ErrMsg)
		}

		return nil
	}

	return nil
}

func (app *Leader) ClearNode(id string) error {
	err := app.dm.RemoveNode(id)
	if err != nil {
		return err
	}
	return nil
}

func (app *Leader) RemoveBridge(bridgeId string) error {

	b, ok := app.topo.GetBridge(bridgeId)

	if !ok {
		return errors.New(bridgeId + " ID doesn't exist")
	}

	if b.MachineId == app.GetMachineId() {
		for _, node := range b.ConnectedNodes {
			graphDB.RemoveNode(node.NetworkNode.GetMac())
		}
	}

	link := b.RouterLink

	_, err := app.topo.RemoveBridge(bridgeId)

	if err != nil {
		return err
	}

	if link != nil {
		app.specialLinkCleanup(link.NetworkBILink)
	}

	if b.MachineId != app.GetMachineId() {

		body := &removeApi.RemoveBridgeRequest{
			Name: bridgeId,
		}

		resp, err := app.cl.SendMsg(b.MachineId, body, "removeBridge")

		if err != nil {
			return err
		}

		d := json.NewDecoder(resp.Body)
		req := &removeApi.RemoveBridgeResponse{}
		err = d.Decode(&req)

		if err != nil {
			return err
		}

		if req.Error.ErrCode != 0 {
			return errors.New(req.Error.ErrMsg)
		}

		return nil
	}
	return nil
}

func (app *Leader) RemoveRouter(routerId string) error {

	r, ok := app.topo.GetRouter(routerId)

	if !ok {
		return errors.New(routerId + " ID doesn't exist")
	}

	if r.MachineId == app.GetMachineId() {
		graphDB.RemoveRouter(routerId)
	}

	for _, router := range r.ConnectedRouters {
		if router.MachineId != app.GetMachineId() {
			body := &disconnectApi.DisconnectRoutersRequest{
				First:  routerId,
				Second: router.ID(),
			}
			resp, err := app.cl.SendMsg(router.MachineId, body, "disconnectRouters")
			if err != nil {
				return err
			}

			d := json.NewDecoder(resp.Body)
			req := &disconnectApi.DisconnectRoutersResponse{}
			err = d.Decode(&req)

			if err != nil {
				return err
			}

			if req.Error.ErrCode != 0 {
				return errors.New(req.Error.ErrMsg)
			}
		}
	}

	links := make([]*topology.BiLink, len(r.RouterLinks))

	for _, link := range r.RouterLinks {
		links = append(links, link)
	}

	_, err := app.topo.RemoveRouter(routerId)

	if err != nil {
		return err
	}

	for _, link := range links {
		app.specialLinkCleanup(link.NetworkBILink)
	}

	if r.MachineId != app.GetMachineId() {
		body := &removeApi.RemoveRouterRequest{
			Name: routerId,
		}

		resp, err := app.cl.SendMsg(r.MachineId, body, "removeRouter")

		if err != nil {
			return err
		}

		d := json.NewDecoder(resp.Body)
		req := &removeApi.RemoveRouterResponse{}
		err = d.Decode(&req)

		if err != nil {
			return err
		}

		if req.Error.ErrCode != 0 {
			return errors.New(req.Error.ErrMsg)
		}

		return nil
	}
	return nil
}

func (app *Leader) DisconnectNode(id string) error {

	n, ok := app.topo.GetNode(id)

	if !ok {
		return errors.New(id + " ID doesn't exist")
	}

	if n.MachineId == app.GetMachineId() {
		graphDB.RemoveNode(n.NetworkNode.GetMac())
		if n.Bridge != nil {
			link := n.Link
			err := app.topo.DisconnectNode(id)
			if err != nil {
				return err
			}
			if link != nil {
				app.specialLinkCleanup(link.NetworkBILink)
			}
			return nil
		} else {
			return errors.New(id + "is not connected to anything")
		}
	}
	b := &disconnectApi.DisconnectNodeRequest{
		Name: id,
	}
	resp, err := app.cl.SendMsg(n.MachineId, b, "disconnectNode")
	if err != nil {
		return err
	}

	d := json.NewDecoder(resp.Body)
	req := &disconnectApi.DisconnectNodeResponse{}
	err = d.Decode(&req)

	if err != nil {
		return err
	}

	if req.Error.ErrCode != 0 {
		return errors.New(req.Error.ErrMsg)
	}

	return nil
}

func (app *Leader) DisconnectBridge(id string) error {

	b, ok := app.topo.GetBridge(id)

	if !ok {
		return errors.New(id + " ID doesn't exist")
	}
	if b.MachineId == app.GetMachineId() {

		for _, node := range b.ConnectedNodes {
			graphDB.RemoveNode(node.NetworkNode.GetMac())
		}
		link := b.RouterLink
		err := app.topo.DisconnectBridge(id)
		if err != nil {
			return err
		}
		if link != nil {
			app.specialLinkCleanup(link.NetworkBILink)
		}
		return nil
	}
	body := &disconnectApi.DisconnectBridgeRequest{
		Name: id,
	}
	resp, err := app.cl.SendMsg(b.MachineId, body, "disconnectBridge")
	if err != nil {
		return err
	}

	d := json.NewDecoder(resp.Body)
	req := &disconnectApi.DisconnectBridgeResponse{}
	err = d.Decode(&req)

	if err != nil {
		return err
	}

	if req.Error.ErrCode != 0 {
		return errors.New(req.Error.ErrMsg)
	}

	return nil
}
func (app *Leader) DisconnectRouters(router1 string, router2 string) error {

	r1, ok := app.topo.GetRouter(router1)

	if !ok {
		return errors.New(router1 + " ID doesn't exist")
	}
	r2, ok := app.topo.GetRouter(router2)

	if !ok {
		return errors.New(router2 + " ID doesn't exist")
	}

	if r1.MachineId == app.GetMachineId() {
		graphDB.RemovePath(router1, router2)

		link := r1.RouterLinks[router2]
		err := app.topo.DisconnectRouters(router1, router2)
		if err != nil {
			return err
		}
		if link != nil {
			app.specialLinkCleanup(link.NetworkBILink)
		}
		if r2.MachineId != app.GetMachineId() {
			body := &disconnectApi.DisconnectRoutersRequest{
				First:  router2,
				Second: router1,
			}
			resp, err := app.cl.SendMsg(r2.MachineId, body, "disconnectRouters")
			if err != nil {
				return err
			}

			d := json.NewDecoder(resp.Body)
			req := &disconnectApi.DisconnectRoutersResponse{}
			err = d.Decode(&req)

			if err != nil {
				return err
			}

			if req.Error.ErrCode != 0 {
				return errors.New(req.Error.ErrMsg)
			}

			return nil
		}
		return nil
	} else {
		if r2.MachineId == app.GetMachineId() {
			graphDB.RemovePath(router1, router2)

			link := r2.RouterLinks[router1]
			err := app.topo.DisconnectRouters(router2, router1)
			if err != nil {
				return err
			}
			if link != nil {
				app.specialLinkCleanup(link.NetworkBILink)
			}
			body := &disconnectApi.DisconnectRoutersRequest{
				First:  router1,
				Second: router2,
			}
			resp, err := app.cl.SendMsg(r1.MachineId, body, "disconnectRouters")
			if err != nil {
				return err
			}

			d := json.NewDecoder(resp.Body)
			req := &disconnectApi.DisconnectRoutersResponse{}
			err = d.Decode(&req)

			if err != nil {
				return err
			}

			if req.Error.ErrCode != 0 {
				return errors.New(req.Error.ErrMsg)
			}

			return nil
		} else {
			body := &disconnectApi.DisconnectRoutersRequest{
				First:  router1,
				Second: router2,
			}
			resp, err := app.cl.SendMsg(r1.MachineId, body, "disconnectRouters")
			if err != nil {
				return err
			}

			d := json.NewDecoder(resp.Body)
			req := &disconnectApi.DisconnectRoutersResponse{}
			err = d.Decode(&req)

			if err != nil {
				return err
			}

			if req.Error.ErrCode != 0 {
				return errors.New(req.Error.ErrMsg)
			}

			resp, err = app.cl.SendMsg(r2.MachineId, body, "disconnectRouters")
			if err != nil {
				return err
			}

			d = json.NewDecoder(resp.Body)
			req = &disconnectApi.DisconnectRoutersResponse{}
			err = d.Decode(&req)

			if err != nil {
				return err
			}

			if req.Error.ErrCode != 0 {
				return errors.New(req.Error.ErrMsg)
			}
			return nil
		}
	}
}

func (app *Leader) Propagate(routerId string) error {

	r, ok := app.topo.GetRouter(routerId)

	if !ok {
		return errors.New(routerId + " ID doesn't exist")
	}

	if r.MachineId == app.GetMachineId() {
		app.PropagateNewRoutes(r)
	} else {
		body := &opApi.PropagateRequest{
			Name: routerId,
		}
		resp, err := app.cl.SendMsg(r.MachineId, body, "propagate")
		if err != nil {
			return err
		}

		d := json.NewDecoder(resp.Body)
		req := &opApi.PropagateResponse{}
		err = d.Decode(&req)

		if err != nil {
			return err
		}

		if req.Error.ErrCode != 0 {
			return errors.New(req.Error.ErrMsg)
		}
	}
	return nil
}
func (app *Leader) Forget(routerId string) error {

	r, ok := app.topo.GetRouter(routerId)

	if !ok {
		return errors.New(routerId + " ID doesn't exist")
	}

	if r.MachineId == app.GetMachineId() {
		return app.topo.ForgetRoutes(routerId)
	} else {
		body := &opApi.ForgetRequest{
			Name: routerId,
		}
		resp, err := app.cl.SendMsg(r.MachineId, body, "forget")
		if err != nil {
			return err
		}

		d := json.NewDecoder(resp.Body)
		req := &opApi.ForgetResponse{}
		err = d.Decode(&req)

		if err != nil {
			return err
		}

		if req.Error.ErrCode != 0 {
			return errors.New(req.Error.ErrMsg)
		}
	}
	return nil
}

func (app *Leader) ListSniffers() []string {

	broadcast, err := app.cl.Broadcast(nil, http.MethodGet, "listSniffers")
	if err != nil {
		return nil
	}
	snifferSize := 0
	sniffersList := make([]*opApi.ListSniffersResponse, 0, len(broadcast))
	for _, res := range broadcast {
		response := &opApi.ListSniffersResponse{}

		d := json.NewDecoder(res.Body)
		err = d.Decode(&response)
		snifferSize += len(response.Sniffers)
		sniffersList = append(sniffersList, response)
	}

	ids := make([]string, 0, snifferSize)

	for _, sniffer := range sniffersList {
		for _, id := range sniffer.Sniffers {
			ids = append(ids, id)
		}
	}

	return ids
}

func (app *Leader) StopSniffNode(id string) error {

	n, ok := app.topo.GetNode(id)

	if !ok {
		return errors.New(id + " ID doesn't exist")
	}

	if n.MachineId == app.GetMachineId() {
		var redirect *redirecttraffic.RedirectionSocket
		if redirect, ok = app.sniffers[id]; !ok {
			return errors.New("not sniffing traffic in this node")
		}

		if n.Bridge == nil {
			app.sniffers[id].Stop()
			delete(app.sniffers, id)
			return errors.New(id + "is not connected to any bridge")
		}

		n.Link.NetworkBILink.Left.Stop()
		n.Link.NetworkBILink.Right.Stop()

		newSniff := n.Link.NetworkBILink.Left.GetShaper().(*network.SniffShaper).ConvertToNetworkShaper()
		n.Link.NetworkBILink.Left.SetShaper(newSniff)
		n.Link.NetworkBILink.Left.Start()

		newSniff = n.Link.NetworkBILink.Right.GetShaper().(*network.SniffShaper).ConvertToNetworkShaper()
		n.Link.NetworkBILink.Right.SetShaper(newSniff)
		n.Link.NetworkBILink.Right.Start()

		delete(app.sniffers, id)
		redirect.Stop()

		return nil

	} else {
		body := &opApi.StopSniffRequest{
			Id: id,
		}
		resp, err := app.cl.SendMsg(n.MachineId, body, "stopSniffNode")
		if err != nil {
			return err
		}

		d := json.NewDecoder(resp.Body)
		req := &opApi.StopSniffResponse{}
		err = d.Decode(&req)

		if err != nil {
			return err
		}

		if req.Error.ErrCode != 0 {
			return errors.New(req.Error.ErrMsg)
		}

		return nil
	}
}
func (app *Leader) StopSniffBridge(id string) error {

	b, ok := app.topo.GetBridge(id)

	if !ok {
		return errors.New(id + " ID doesn't exist")
	}

	if b.MachineId == app.GetMachineId() {
		var redirect *redirecttraffic.RedirectionSocket
		if redirect, ok = app.sniffers[id]; !ok {
			return errors.New("not sniffing traffic in this bridge")
		}

		if b.Router == nil {
			app.sniffers[id].Stop()
			delete(app.sniffers, id)
			return errors.New(id + "is not connected to any router")
		}

		b.RouterLink.NetworkBILink.Left.Stop()
		b.RouterLink.NetworkBILink.Right.Stop()

		newSniff := b.RouterLink.NetworkBILink.Left.GetShaper().(*network.SniffShaper).ConvertToNetworkShaper()
		b.RouterLink.NetworkBILink.Left.SetShaper(newSniff)
		b.RouterLink.NetworkBILink.Left.Start()

		newSniff = b.RouterLink.NetworkBILink.Right.GetShaper().(*network.SniffShaper).ConvertToNetworkShaper()
		b.RouterLink.NetworkBILink.Right.SetShaper(newSniff)
		b.RouterLink.NetworkBILink.Right.Start()

		delete(app.sniffers, id)
		redirect.Stop()

		return nil

	} else {
		body := &opApi.StopSniffRequest{
			Id: id,
		}
		resp, err := app.cl.SendMsg(b.MachineId, body, "stopSniffBridge")
		if err != nil {
			return err
		}

		d := json.NewDecoder(resp.Body)
		req := &opApi.StopSniffResponse{}
		err = d.Decode(&req)

		if err != nil {
			return err
		}

		if req.Error.ErrCode != 0 {
			return errors.New(req.Error.ErrMsg)
		}

		return nil
	}

}
func (app *Leader) StopSniffRouters(id string) error {
	routers := strings.Split(id, "-")

	if len(routers) != 2 {
		return errors.New("invalid id")
	}

	r1, ok := app.topo.GetRouter(routers[0])

	if !ok {
		return errors.New(routers[0] + " ID doesn't exist")
	}

	r2, ok := app.topo.GetRouter(routers[1])

	if !ok {
		return errors.New(routers[1] + " ID doesn't exist")
	}

	if r1.MachineId == app.GetMachineId() {
		if r2.MachineId == app.GetMachineId() {
			var redirect *redirecttraffic.RedirectionSocket
			if redirect, ok = app.sniffers[r1.ID()+"-"+r2.ID()]; !ok {
				return errors.New("not sniffing traffic between" + r1.ID() + " and " + r2.ID())

			} else {
				delete(app.sniffers, r1.ID()+"-"+r2.ID())
				redirect.Stop()
			}
			if redirect, ok = app.sniffers[r2.ID()+"-"+r1.ID()]; !ok {
				return errors.New("not sniffing traffic between" + r1.ID() + " and " + r2.ID())

			} else {
				delete(app.sniffers, r2.ID()+"-"+r1.ID())
				redirect.Stop()
			}

			link, ok := r1.RouterLinks[r2.ID()]
			if !ok {
				return errors.New(r1.ID() + " and " + r2.ID() + "are not connected")
			}

			link.NetworkBILink.Left.Stop()
			link.NetworkBILink.Right.Stop()

			newSniff := link.NetworkBILink.Left.GetShaper().(*network.SniffShaper).ConvertToNetworkShaper()
			link.NetworkBILink.Left.SetShaper(newSniff)
			link.NetworkBILink.Left.Start()

			newSniff = link.NetworkBILink.Right.GetShaper().(*network.SniffShaper).ConvertToNetworkShaper()
			link.NetworkBILink.Right.SetShaper(newSniff)
			link.NetworkBILink.Right.Start()

			return nil
		} else {
			return errors.New("can't sniff traffic between remote routers")
		}

	} else {
		if r2.MachineId == app.GetMachineId() {
			return errors.New("can't sniff traffic between remote routers")
		} else {

			body := &opApi.StopSniffRequest{
				Id: id,
			}
			resp, err := app.cl.SendMsg(r1.MachineId, body, "stopSniffRouters")
			if err != nil {
				return err
			}

			d := json.NewDecoder(resp.Body)
			req := &opApi.StopSniffResponse{}
			err = d.Decode(&req)

			if err != nil {
				return err
			}

			if req.Error.ErrCode != 0 {
				return errors.New(req.Error.ErrMsg)
			}

			return nil
		}
	}
}

func (app *Leader) SniffNode(id string) (string, string, string, error) {

	n, ok := app.topo.GetNode(id)

	if !ok {
		return "", "", "", errors.New(id + " ID doesn't exist")
	}

	if n.MachineId == app.GetMachineId() {

		if _, ok = app.sniffers[id]; ok {
			return "", "", "", errors.New("Already sniffing" + id)
		}

		if n.Bridge == nil {
			return "", "", "", errors.New(id + "is not connected to any bridge")
		}

		if _, ok = n.Link.NetworkBILink.Left.GetShaper().(*network.NetworkShaper); !ok {
			return "", "", "", errors.New("already performing an operation on this link")
		} else if _, ok = n.Link.NetworkBILink.Right.GetShaper().(*network.NetworkShaper); !ok {
			return "", "", "", errors.New("already performing an operation on this link")
		}

		redirect, err := redirecttraffic.NewRedirectionSocket(id, sniffSocketPath(id))
		if err != nil {
			return "", "", "", err

		}

		n.Link.NetworkBILink.Left.Stop()
		n.Link.NetworkBILink.Right.Stop()

		newSniff := n.Link.NetworkBILink.Left.GetShaper().(*network.NetworkShaper).ConvertToSniffShaper(redirect)
		n.Link.NetworkBILink.Left.SetShaper(newSniff)
		n.Link.NetworkBILink.Left.Start()

		newSniff = n.Link.NetworkBILink.Right.GetShaper().(*network.NetworkShaper).ConvertToSniffShaper(redirect)
		n.Link.NetworkBILink.Right.SetShaper(newSniff)
		n.Link.NetworkBILink.Right.Start()

		app.sniffers[id] = redirect
		go redirect.Start()

		return id, redirect.GetSocketPath(), n.MachineId, nil

	} else {
		body := &opApi.SniffNodeRequest{
			Name: id,
		}
		resp, err := app.cl.SendMsg(n.MachineId, body, "sniffNode")
		if err != nil {
			return "", "", "", err
		}

		d := json.NewDecoder(resp.Body)
		req := &opApi.SniffNodeResponse{}
		err = d.Decode(&req)

		if err != nil {
			return "", "", "", err
		}

		if req.Error.ErrCode != 0 {
			return "", "", "", errors.New(req.Error.ErrMsg)
		}

		return req.Id, req.Path, req.MachineId, nil
	}
}

func (app *Leader) SniffBridge(id string) (string, string, string, error) {

	b, ok := app.topo.GetBridge(id)

	if !ok {
		return "", "", "", errors.New(id + " ID doesn't exist")
	}

	if b.MachineId == app.GetMachineId() {
		if _, ok = app.sniffers[id]; ok {
			return "", "", "", errors.New("Already sniffing" + id)
		}

		if b.Router == nil {
			return "", "", "", errors.New(id + "is not connected to any router")
		}

		if _, ok = b.RouterLink.NetworkBILink.Left.GetShaper().(*network.NetworkShaper); !ok {
			return "", "", "", errors.New("already performing an operation on this link")
		} else if _, ok = b.RouterLink.NetworkBILink.Right.GetShaper().(*network.NetworkShaper); !ok {
			return "", "", "", errors.New("already performing an operation on this link")
		}

		redirect, err := redirecttraffic.NewRedirectionSocket(id, sniffSocketPath(id))
		if err != nil {
			return "", "", "", err

		}

		b.RouterLink.NetworkBILink.Left.Stop()
		b.RouterLink.NetworkBILink.Right.Stop()

		newSniff := b.RouterLink.NetworkBILink.Left.GetShaper().(*network.NetworkShaper).ConvertToSniffShaper(redirect)
		b.RouterLink.NetworkBILink.Left.SetShaper(newSniff)
		b.RouterLink.NetworkBILink.Left.Start()

		newSniff = b.RouterLink.NetworkBILink.Right.GetShaper().(*network.NetworkShaper).ConvertToSniffShaper(redirect)
		b.RouterLink.NetworkBILink.Right.SetShaper(newSniff)
		b.RouterLink.NetworkBILink.Right.Start()

		app.sniffers[id] = redirect
		go redirect.Start()

		return id, redirect.GetSocketPath(), b.MachineId, nil

	} else {
		body := &opApi.SniffBridgeRequest{
			Name: id,
		}
		resp, err := app.cl.SendMsg(b.MachineId, body, "sniffBridge")
		if err != nil {
			return "", "", "", err
		}

		d := json.NewDecoder(resp.Body)
		req := &opApi.SniffBridgeResponse{}
		err = d.Decode(&req)

		if err != nil {
			return "", "", "", err
		}

		if req.Error.ErrCode != 0 {
			return "", "", "", errors.New(req.Error.ErrMsg)
		}

		return req.Id, req.Path, req.MachineId, nil
	}
}
func (app *Leader) SniffRouters(router1 string, router2 string) (string, string, string, error) {

	r1, ok := app.topo.GetRouter(router1)

	if !ok {
		return "", "", "", errors.New(router1 + " ID doesn't exist")
	}

	r2, ok := app.topo.GetRouter(router2)

	if !ok {
		return "", "", "", errors.New(router2 + " ID doesn't exist")
	}

	if r1.MachineId == app.GetMachineId() {
		if r2.MachineId == app.GetMachineId() {
			if _, ok = app.sniffers[router1+"-"+router2]; ok {
				return "", "", "", errors.New("already sniffing " + router1 + " and " + router2)
			}
			if _, ok = app.sniffers[router2+"-"+router1]; ok {
				return "", "", "", errors.New("already sniffing " + router1 + " and " + router2)
			}
			link, ok := r1.RouterLinks[router2]
			if !ok {
				return "", "", "", errors.New(router1 + " and " + router2 + "are not connected")
			}

			if _, ok = link.NetworkBILink.Left.GetShaper().(*network.NetworkShaper); !ok {
				return "", "", "", errors.New("already performing an operation on this link")
			} else if _, ok = link.NetworkBILink.Right.GetShaper().(*network.NetworkShaper); !ok {
				return "", "", "", errors.New("already performing an operation on this link")
			}

			redirect, err := redirecttraffic.NewRedirectionSocket(router1+"-"+router2, sniffSocketPath(router1+"-"+router2))
			if err != nil {
				return "", "", "", err

			}

			link.NetworkBILink.Left.Stop()
			link.NetworkBILink.Right.Stop()

			newSniff := link.NetworkBILink.Left.GetShaper().(*network.NetworkShaper).ConvertToSniffShaper(redirect)
			link.NetworkBILink.Left.SetShaper(newSniff)
			link.NetworkBILink.Left.Start()

			newSniff = link.NetworkBILink.Right.GetShaper().(*network.NetworkShaper).ConvertToSniffShaper(redirect)
			link.NetworkBILink.Right.SetShaper(newSniff)
			link.NetworkBILink.Right.Start()

			app.sniffers[router1+"-"+router2] = redirect
			go redirect.Start()

			return router1 + "-" + router2, redirect.GetSocketPath(), r1.MachineId, nil
		} else {
			return "", "", "", errors.New("can't sniff traffic between remote routers")
		}

	} else {
		if r2.MachineId == app.GetMachineId() {
			return "", "", "", errors.New("can't sniff traffic between remote routers")
		} else {

			body := &opApi.SniffRoutersRequest{
				Router1: router1,
				Router2: router2,
			}
			resp, err := app.cl.SendMsg(r1.MachineId, body, "sniffRouters")
			if err != nil {
				return "", "", "", err
			}

			d := json.NewDecoder(resp.Body)
			req := &opApi.SniffRoutersResponse{}
			err = d.Decode(&req)

			if err != nil {
				return "", "", "", err
			}

			if req.Error.ErrCode != 0 {
				return "", "", "", errors.New(req.Error.ErrMsg)
			}

			return req.Id, req.Path, req.MachineId, nil
		}
	}
}

func (app *Leader) InterceptNode(id string, direction bool) (string, string, string, error) {

	n, ok := app.topo.GetNode(id)

	if !ok {
		return "", "", "", errors.New(id + " ID doesn't exist")
	}

	interceptId := getInterceptId(id, direction)

	if n.MachineId == app.GetMachineId() {

		if n.Bridge == nil {
			return "", "", "", errors.New(id + "is not connected to any bridge")
		}

		if _, ok = app.intercepts[interceptId]; ok {
			return "", "", "", errors.New("Already intercepting" + interceptId)
		}
		redirect, err := redirecttraffic.NewRedirectionSocket(interceptId, interceptSocketPath(interceptId))
		if err != nil {
			return "", "", "", err

		}

		if direction {

			n.Link.NetworkBILink.Left.Stop()

			intercept := n.Link.NetworkBILink.Left.GetShaper().(*network.NetworkShaper).ConvertToInterceptShaper(redirect)
			n.Link.NetworkBILink.Left.SetShaper(intercept)
			n.Link.NetworkBILink.Left.Start()
		} else {
			n.Link.NetworkBILink.Right.Stop()
			intercept := n.Link.NetworkBILink.Right.GetShaper().(*network.NetworkShaper).ConvertToInterceptShaper(redirect)
			n.Link.NetworkBILink.Right.SetShaper(intercept)
			n.Link.NetworkBILink.Right.Start()

		}

		app.intercepts[interceptId] = redirect
		go redirect.Start()

		return id, redirect.GetSocketPath(), n.MachineId, nil

	} else {
		body := &opApi.InterceptNodeRequest{
			Name: id,
		}
		resp, err := app.cl.SendMsg(n.MachineId, body, "interceptNode")
		if err != nil {
			return "", "", "", err
		}

		d := json.NewDecoder(resp.Body)
		req := &opApi.InterceptNodeResponse{}
		err = d.Decode(&req)

		if err != nil {
			return "", "", "", err
		}

		if req.Error.ErrCode != 0 {
			return "", "", "", errors.New(req.Error.ErrMsg)
		}

		return req.Id, req.Path, req.MachineId, nil
	}
}

func (app *Leader) InterceptBridge(id string, direction bool) (string, string, string, error) {

	b, ok := app.topo.GetBridge(id)

	if !ok {
		return "", "", "", errors.New(id + " ID doesn't exist")
	}

	interceptId := getInterceptId(id, direction)

	if b.MachineId == app.GetMachineId() {
		if b.Router == nil {
			return "", "", "", errors.New(id + "is not connected to any router")
		}

		if _, ok = app.intercepts[interceptId]; ok {
			return "", "", "", errors.New("Already intercepting" + interceptId)
		}
		redirect, err := redirecttraffic.NewRedirectionSocket(interceptId, interceptSocketPath(interceptId))
		if err != nil {
			return "", "", "", err

		}

		if direction {

			b.RouterLink.NetworkBILink.Left.Stop()

			intercept := b.RouterLink.NetworkBILink.Left.GetShaper().(*network.NetworkShaper).ConvertToInterceptShaper(redirect)
			b.RouterLink.NetworkBILink.Left.SetShaper(intercept)
			b.RouterLink.NetworkBILink.Left.Start()

		} else {
			b.RouterLink.NetworkBILink.Right.Stop()

			intercept := b.RouterLink.NetworkBILink.Right.GetShaper().(*network.NetworkShaper).ConvertToInterceptShaper(redirect)
			b.RouterLink.NetworkBILink.Right.SetShaper(intercept)
			b.RouterLink.NetworkBILink.Right.Start()
		}

		app.intercepts[interceptId] = redirect
		go redirect.Start()

		return id, redirect.GetSocketPath(), b.MachineId, nil

	} else {
		body := &opApi.InterceptBridgeRequest{
			Name: id,
		}
		resp, err := app.cl.SendMsg(b.MachineId, body, "interceptBridge")
		if err != nil {
			return "", "", "", err
		}

		d := json.NewDecoder(resp.Body)
		req := &opApi.InterceptBridgeResponse{}
		err = d.Decode(&req)

		if err != nil {
			return "", "", "", err
		}

		if req.Error.ErrCode != 0 {
			return "", "", "", errors.New(req.Error.ErrMsg)
		}

		return req.Id, req.Path, req.MachineId, nil
	}
}

func (app *Leader) InterceptRouters(router1 string, router2 string, direction bool) (string, string, string, error) {

	r1, ok := app.topo.GetRouter(router1)

	if !ok {
		return "", "", "", errors.New(router1 + " ID doesn't exist")
	}

	r2, ok := app.topo.GetRouter(router2)

	if !ok {
		return "", "", "", errors.New(router2 + " ID doesn't exist")
	}

	if r1.MachineId == app.GetMachineId() {
		if r2.MachineId == app.GetMachineId() {
			if _, ok = app.intercepts[getInterceptId(router1+"-"+router2, direction)]; ok {
				return "", "", "", errors.New("already intercepting " + getInterceptId(router1+"-"+router2, direction))
			}
			if _, ok = app.intercepts[getInterceptId(router2+"-"+router1, direction)]; ok {
				return "", "", "", errors.New("already intercepting " + getInterceptId(router1+"-"+router2, direction))
			}
			link, ok := r1.RouterLinks[router2]
			if !ok {
				return "", "", "", errors.New(router1 + " and " + router2 + "are not connected")
			}

			interceptId := getInterceptId(router1+"-"+router2, direction)

			redirect, err := redirecttraffic.NewRedirectionSocket(interceptId, interceptSocketPath(interceptId))
			if err != nil {
				return "", "", "", err

			}

			if link.To.ID() == r2.ID() && direction {

				link.ConnectsTo.NetworkLink.Stop()

				intercept := link.ConnectsTo.NetworkLink.GetShaper().(*network.NetworkShaper).ConvertToInterceptShaper(redirect)
				link.ConnectsTo.NetworkLink.SetShaper(intercept)
				link.ConnectsTo.NetworkLink.Start()
			} else {
				link.ConnectsFrom.NetworkLink.Stop()
				intercept := link.ConnectsFrom.NetworkLink.GetShaper().(*network.NetworkShaper).ConvertToInterceptShaper(redirect)
				link.ConnectsFrom.NetworkLink.SetShaper(intercept)
				link.ConnectsTo.NetworkLink.Start()
			}

			app.intercepts[interceptId] = redirect
			go redirect.Start()

			return router1 + "-" + router2, redirect.GetSocketPath(), r1.MachineId, nil
		} else {
			return "", "", "", errors.New("can't intercept traffic between remote routers")
		}

	} else {
		if r2.MachineId == app.GetMachineId() {
			return "", "", "", errors.New("can't intercept traffic between remote routers")
		} else {

			body := &opApi.InterceptRoutersRequest{
				Router1: router1,
				Router2: router2,
			}
			resp, err := app.cl.SendMsg(r1.MachineId, body, "interceptRouters")
			if err != nil {
				return "", "", "", err
			}

			d := json.NewDecoder(resp.Body)
			req := &opApi.InterceptRoutersResponse{}
			err = d.Decode(&req)

			if err != nil {
				return "", "", "", err
			}

			if req.Error.ErrCode != 0 {
				return "", "", "", errors.New(req.Error.ErrMsg)
			}

			return req.Id, req.Path, req.MachineId, nil
		}
	}
}

func (app *Leader) StopInterceptNode(id string, direction bool) error {

	n, ok := app.topo.GetNode(id)

	if !ok {
		return errors.New(id + " ID doesn't exist")
	}

	if n.MachineId == app.GetMachineId() {
		var redirect *redirecttraffic.RedirectionSocket
		if redirect, ok = app.intercepts[getInterceptId(id, direction)]; !ok {
			return errors.New("not intercepting traffic in this node")
		}

		if n.Bridge == nil {
			app.intercepts[getInterceptId(id, direction)].Stop()
			delete(app.intercepts, getInterceptId(id, direction))
			return errors.New(id + "is not connected to any bridge")
		}

		if direction {
			n.Link.NetworkBILink.Left.Stop()

			networkShaper := n.Link.NetworkBILink.Left.GetShaper().(*network.InterceptShaper).ConvertToNetworkShaper()
			n.Link.NetworkBILink.Left.SetShaper(networkShaper)
			n.Link.NetworkBILink.Left.Start()
		} else {
			n.Link.NetworkBILink.Right.Stop()

			networkShaper := n.Link.NetworkBILink.Right.GetShaper().(*network.InterceptShaper).ConvertToNetworkShaper()
			n.Link.NetworkBILink.Right.SetShaper(networkShaper)
			n.Link.NetworkBILink.Right.Start()

		}

		delete(app.intercepts, getInterceptId(id, direction))
		redirect.Stop()

		return nil

	} else {
		body := &opApi.StopInterceptRequest{
			Id: id,
		}
		resp, err := app.cl.SendMsg(n.MachineId, body, "stopInterceptNode")
		if err != nil {
			return err
		}

		d := json.NewDecoder(resp.Body)
		req := &opApi.StopInterceptResponse{}
		err = d.Decode(&req)

		if err != nil {
			return err
		}

		if req.Error.ErrCode != 0 {
			return errors.New(req.Error.ErrMsg)
		}

		return nil
	}
}

func (app *Leader) StopInterceptBridge(id string, direction bool) error {
	b, ok := app.topo.GetBridge(id)

	if !ok {
		return errors.New(id + " ID doesn't exist")
	}

	if b.MachineId == app.GetMachineId() {
		var redirect *redirecttraffic.RedirectionSocket
		if redirect, ok = app.intercepts[getInterceptId(id, direction)]; !ok {
			return errors.New("not intercepting traffic in this bridge")
		}

		if b.Router == nil {
			app.intercepts[getInterceptId(id, direction)].Stop()
			delete(app.intercepts, getInterceptId(id, direction))
			return errors.New(id + "is not connected to any router")
		}

		if direction {
			b.RouterLink.NetworkBILink.Left.Stop()

			networkShaper := b.RouterLink.NetworkBILink.Left.GetShaper().(*network.InterceptShaper).ConvertToNetworkShaper()
			b.RouterLink.NetworkBILink.Left.SetShaper(networkShaper)
			b.RouterLink.NetworkBILink.Left.Start()

		} else {
			b.RouterLink.NetworkBILink.Right.Stop()

			networkShaper := b.RouterLink.NetworkBILink.Right.GetShaper().(*network.InterceptShaper).ConvertToNetworkShaper()
			b.RouterLink.NetworkBILink.Right.SetShaper(networkShaper)
			b.RouterLink.NetworkBILink.Right.Start()
		}

		delete(app.intercepts, getInterceptId(id, direction))
		redirect.Stop()

		return nil

	} else {
		body := &opApi.StopInterceptRequest{
			Id: id,
		}
		resp, err := app.cl.SendMsg(b.MachineId, body, "stopInterceptBridge")
		if err != nil {
			return err
		}

		d := json.NewDecoder(resp.Body)
		req := &opApi.StopInterceptResponse{}
		err = d.Decode(&req)

		if err != nil {
			return err
		}

		if req.Error.ErrCode != 0 {
			return errors.New(req.Error.ErrMsg)
		}

		return nil

	}
}

func (app *Leader) StopInterceptRouters(id string, direction bool) error {

	routers := strings.Split(id, "-")

	if len(routers) != 2 {
		return errors.New("invalid id")
	}

	r1, ok := app.topo.GetRouter(routers[0])

	if !ok {
		return errors.New(routers[0] + " ID doesn't exist")
	}

	r2, ok := app.topo.GetRouter(routers[1])

	if !ok {
		return errors.New(routers[1] + " ID doesn't exist")
	}

	if r1.MachineId == app.GetMachineId() {
		if r2.MachineId == app.GetMachineId() {
			var redirect *redirecttraffic.RedirectionSocket
			if redirect, ok = app.intercepts[getInterceptId(r1.ID()+"-"+r2.ID(), direction)]; !ok {
				return errors.New("not intercepting traffic between" + r1.ID() + " and " + r2.ID())

			} else {
				delete(app.intercepts, getInterceptId(r1.ID()+"-"+r2.ID(), direction))
				redirect.Stop()
			}
			if redirect, ok = app.intercepts[getInterceptId(r2.ID()+"-"+r1.ID(), direction)]; !ok {
				return errors.New("not intercepting traffic between" + r1.ID() + " and " + r2.ID())

			} else {
				delete(app.intercepts, getInterceptId(r2.ID()+"-"+r1.ID(), direction))
				redirect.Stop()
			}

			link, ok := r1.RouterLinks[r2.ID()]
			if !ok {
				return errors.New(r1.ID() + " and " + r2.ID() + "are not connected")
			}

			if link.To.ID() == r2.ID() && direction {

				link.ConnectsTo.NetworkLink.Stop()

				networkShaper := link.ConnectsTo.NetworkLink.GetShaper().(*network.InterceptShaper).ConvertToNetworkShaper()
				link.ConnectsTo.NetworkLink.SetShaper(networkShaper)
				link.ConnectsTo.NetworkLink.Start()
			} else {
				link.ConnectsFrom.NetworkLink.Stop()

				networkShaper := link.ConnectsFrom.NetworkLink.GetShaper().(*network.InterceptShaper).ConvertToNetworkShaper()
				link.ConnectsFrom.NetworkLink.SetShaper(networkShaper)
				link.ConnectsTo.NetworkLink.Start()
			}
			return nil
		} else {
			return errors.New("can't intercept traffic between remote routers")
		}

	} else {
		if r2.MachineId == app.GetMachineId() {
			return errors.New("can't intercept traffic between remote routers")
		} else {

			body := &opApi.StopInterceptRequest{
				Id: id,
			}
			resp, err := app.cl.SendMsg(r1.MachineId, body, "stopInterceptRouters")
			if err != nil {
				return err
			}

			d := json.NewDecoder(resp.Body)
			req := &opApi.StopInterceptResponse{}
			err = d.Decode(&req)

			if err != nil {
				return err
			}

			if req.Error.ErrCode != 0 {
				return errors.New(req.Error.ErrMsg)
			}

			return nil
		}
	}
}

func (app *Leader) ListIntercepts() []string {

	broadcast, err := app.cl.Broadcast(nil, http.MethodGet, "listIntercepts")
	if err != nil {
		return nil
	}
	interceptSize := 0
	interceptList := make([]*opApi.ListInterceptsResponse, 0, len(broadcast))
	for _, res := range broadcast {
		response := &opApi.ListInterceptsResponse{}

		d := json.NewDecoder(res.Body)
		err = d.Decode(&response)
		interceptSize += len(response.Intercepts)
		interceptList = append(interceptList, response)
	}

	ids := make([]string, 0, interceptSize)

	for _, intercept := range interceptList {
		for _, id := range intercept.Intercepts {
			ids = append(ids, id)
		}
	}

	return ids
}

func (app *Leader) Pause(id string, all bool) error {
	if all {
		app.dm.PauseAll()
		_, err := app.cl.Broadcast(&opApi.PauseRequest{
			Id:  "",
			All: true,
		}, http.MethodPost, "pause")
		return err
	} else {
		if n, ok := app.topo.GetNode(id); ok {
			if n.MachineId == app.GetMachineId() {
				return app.dm.Pause(id)
			} else {
				_, err := app.cl.SendMsg(n.MachineId, &opApi.PauseRequest{
					Id:  id,
					All: false,
				}, "pause")
				return err
			}
		} else {
			return errors.New("invalid node id")
		}
	}
}
func (app *Leader) Unpause(id string, all bool) error {
	if all {
		app.dm.UnpauseAll()
		_, err := app.cl.Broadcast(&opApi.UnpauseRequest{
			Id:  "",
			All: true,
		}, http.MethodPost, "unpause")
		return err
	} else {
		if n, ok := app.topo.GetNode(id); ok {
			if n.MachineId == app.GetMachineId() {
				return app.dm.Unpause(id)
			} else {
				_, err := app.cl.SendMsg(n.MachineId, &opApi.UnpauseRequest{
					Id:  id,
					All: false,
				}, "unpause")
				return err
			}
		} else {
			return errors.New("invalid node id")
		}
	}
}

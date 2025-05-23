package application

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/David-Antunes/gone-proxy/xdp"
	"github.com/David-Antunes/gone/api"
	addApi "github.com/David-Antunes/gone/api/Add"
	connectApi "github.com/David-Antunes/gone/api/Connect"
	disconnectApi "github.com/David-Antunes/gone/api/Disconnect"
	opApi "github.com/David-Antunes/gone/api/Operations"
	removeApi "github.com/David-Antunes/gone/api/Remove"
	"github.com/David-Antunes/gone/internal"
	internalApi "github.com/David-Antunes/gone/internal/api"
	"github.com/David-Antunes/gone/internal/cluster"
	"github.com/David-Antunes/gone/internal/docker"
	"github.com/David-Antunes/gone/internal/graphDB"
	"github.com/David-Antunes/gone/internal/network"
	"github.com/David-Antunes/gone/internal/proxy"
	redirecttraffic "github.com/David-Antunes/gone/internal/redirect-traffic"
	"github.com/David-Antunes/gone/internal/topology"
	"net"
	"net/http"
)

type Leader struct {
	cl              *cluster.Cluster
	dm              *docker.DockerManager
	proxy           *proxy.Proxy
	topo            *topology.Topology
	icm             *cluster.InterCommunicationManager
	rm              *LocalRttManager
	redirectManager *redirecttraffic.RedirectManager
}

func NewLeader(cl *cluster.Cluster, dm *docker.DockerManager, proxy *proxy.Proxy, icm *cluster.InterCommunicationManager, rm *LocalRttManager) *Leader {
	return &Leader{
		cl:              cl,
		dm:              dm,
		proxy:           proxy,
		topo:            topology.CreateTopology(dm.GetMachineId(), proxy, rm.GetRtt()),
		icm:             icm,
		rm:              rm,
		redirectManager: redirecttraffic.NewRedirectManager(),
	}
}

func (app *Leader) GetMachineId() string {
	return app.dm.GetMachineId()
}

func (app *Leader) GetNode(id string) (api.Node, bool) {
	if n, ok := app.topo.GetNode(id); ok {
		return convertToAPINode(n), true
	} else {
		return api.Node{}, false
	}
}

func (app *Leader) GetBridge(id string) (api.Bridge, bool) {

	if b, ok := app.topo.GetBridge(id); ok {
		return convertToAPIBridge(b), true
	} else {
		return api.Bridge{}, false
	}
}
func (app *Leader) GetRouter(id string) (api.Router, bool) {
	if r, ok := app.topo.GetRouter(id); ok {
		return convertToAPIRouter(r), true
	} else {
		return api.Router{}, false
	}
}

func (app *Leader) GetRouterWeights(id string) map[string]topology.Weight {
	app.topo.Lock()
	defer app.topo.Unlock()
	r, _ := app.topo.GetRouter(id)
	return r.Weights
}

// Changes sniffShaper into network shaper
func (app *Leader) clearSniffLink(id string) error {
	if sniffer, err := app.redirectManager.GetSniffer(id); err != nil {
		return err
	} else {
		err = app.redirectManager.RemoveSniffer(id)
		sniffer.Socket.Stop()
		if err != nil {
			return err
		} else {
			// Convert to topology Bilink
			bilink := sniffer.Component.(*topology.BiLink).NetworkBILink

			// Convert to individual Link
			bilink.Left.GetShaper().Close()
			l := bilink.Left.GetShaper().(*network.SniffShaper).ConvertToNetworkShaper()
			bilink.Left.SetShaper(l)
			l.Start()

			bilink.Right.GetShaper().Close()
			r := bilink.Right.GetShaper().(*network.SniffShaper).ConvertToNetworkShaper()
			bilink.Right.SetShaper(l)
			r.Start()
		}
	}
	return nil
}

// Changes Intercept into network shaper
func (app *Leader) clearInterceptLink(id string) error {
	if intercept, err := app.redirectManager.GetIntercept(id); err != nil {
		return err
	} else {
		err = app.redirectManager.RemoveIntercept(id)
		if err != nil {
			return err
		} else {

			// Convert to network link
			link := intercept.Component.(*topology.Link).NetworkLink

			link.GetShaper().Close()
			l := link.GetShaper().(*network.InterceptShaper).ConvertToNetworkShaper()
			link.SetShaper(l)
			l.Start()
		}
	}
	return nil
}

// Garbage collects shapers
func (app *Leader) gcLinkShaper(link *topology.BiLink) error {

	if link.ConnectsTo != nil {

		link.ConnectsTo.NetworkLink.Close()
		if s, ok := link.ConnectsTo.NetworkLink.GetShaper().(*network.SniffShaper); ok {
			_ = app.redirectManager.RemoveSniffer(s.GetRtID())
		}

		if s, ok := link.ConnectsTo.NetworkLink.GetShaper().(*network.InterceptShaper); ok {
			_ = app.redirectManager.RemoveIntercept(s.GetRtID())
		}
		if s, ok := link.ConnectsTo.NetworkLink.GetShaper().(*network.RemoteShaper); ok {
			app.icm.RemoveConnection(s.To, s.From)
		}
	}

	if link.ConnectsFrom != nil {
		link.ConnectsFrom.NetworkLink.Close()
		if s, ok := link.ConnectsFrom.NetworkLink.GetShaper().(*network.SniffShaper); ok {
			_ = app.redirectManager.RemoveSniffer(s.GetRtID())
		}

		if s, ok := link.ConnectsFrom.NetworkLink.GetShaper().(*network.InterceptShaper); ok {
			_ = app.redirectManager.RemoveIntercept(s.GetRtID())
		}

		if s, ok := link.ConnectsFrom.NetworkLink.GetShaper().(*network.RemoteShaper); ok {
			app.icm.RemoveConnection(s.To, s.From)
		}
	}

	return nil
}

func (app *Leader) HandleNewMac(frame *xdp.Frame, routerId string) {

	dest := frame.MacDestination

	r, _ := app.topo.GetRouter(routerId)
	path, distance := graphDB.FindPathToRouter(routerId, net.HardwareAddr(dest).String())

	if len(path) > 0 {
		if net.HardwareAddr(dest).String() == path[len(path)-1] {
			if internal.LocalQuery {
				fmt.Println(routerId, ":", net.HardwareAddr(dest), ":", path[:2])
				app.topo.InsertLocalPath(path[:2], frame, distance)
			} else {
				fmt.Println(routerId, ":", net.HardwareAddr(dest), ":", path)
				app.topo.InsertNewPath(path[:len(path)-1], frame, distance)
			}
			r.NetworkRouter.InjectFrame(frame)
		}
	} else {
		//app.topo.InsertNullPath(frame.MacDestination, routerId)
		return
	}
}

func (app *Leader) execInMachine(machineId string, dockerCmd []string) (string, string, string, error) {
	body := &addApi.AddNodeRequest{
		DockerCmd: dockerCmd,
		MachineId: machineId,
	}

	resp, err := app.cl.SendMsg(machineId, body, "addNode")

	if err != nil {
		return "", "", "", err
	}

	d := json.NewDecoder(resp.Body)

	result := &addApi.AddNodeResponse{}
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

	_, err = app.cl.Broadcast(&internalApi.RegisterNodeRequest{
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
			Latency:   linkProps.FLatency,
			Jitter:    linkProps.Jitter * 2.0,
			DropRate:  linkProps.DropRate * 2.0,
			Bandwidth: linkProps.Bandwidth * 8,
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
			Latency:   linkProps.FLatency,
			Jitter:    linkProps.Jitter * 2.0,
			DropRate:  linkProps.DropRate * 2.0,
			Bandwidth: linkProps.Bandwidth * 8,
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

func (app *Leader) ConnectRouterToRouter(router1ID string, router2ID string, linkProps network.LinkProps, propagate bool) error {

	if router1ID == router2ID {
		return errors.New("can't connect a router to itself")
	}

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
			app.TradeRoutes(r1, r2)
			if propagate {
				app.PropagateNewRoutes(r1)
			}
			return nil
		} else {
			return app.connectRouterToRouterRemote(r1, r2, linkProps, propagate)
		}
	} else {
		return app.RedirectConnection(r1, r2, linkProps, propagate)
	}
}

func (app *Leader) connectRouterToRouterRemote(r1 *topology.Router, r2 *topology.Router, linkProps network.LinkProps, propagate bool) error {
	app.topo.Lock()
	if _, ok := r1.ConnectedRouters[r2.ID()]; ok {
		return errors.New(r1.ID() + " is already connected to " + r2.ID())
	}

	if _, ok := r2.ConnectedRouters[r1.ID()]; ok {
		return errors.New(r1.ID() + " is already connected to " + r2.ID())
	}

	router1Channel := make(chan *xdp.Frame, internal.QueueSize)
	conn := app.cl.Endpoints[r2.MachineId]
	d, _ := app.cl.GetNodeDelay(r2.MachineId)
	// Temporary Fix
	app.icm.AddMachine(conn, r2.MachineId)
	app.icm.AddConnection(r2.ID(), d, r2.MachineId, r1.ID(), r1.NetworkRouter)
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
	s.SetDelay(d)
	toLink.SetShaper(s)
	toLink.Start()
	b := &internalApi.ConnectRouterToRouterRequest{
		R1:        r2.ID(),
		R2:        r1.ID(),
		MachineID: r1.MachineId,
		Latency:   linkProps.Latency * 2.0,
		Jitter:    linkProps.Jitter * 2.0,
		DropRate:  linkProps.DropRate * 2.0,
		Bandwidth: linkProps.Bandwidth * 8,
		Weight:    linkProps.Weight,
		Propagate: propagate,
	}

	app.topo.Unlock()

	_, err := app.cl.SendMsg(r2.MachineId, b, "connectRouterToRouterRemote")
	if err != nil {
		return errors.New("couldn't contact machine")
	}

	graphDB.AddPath(r1.ID(), r2.ID(), BiLink.ID(), linkProps.Weight)

	app.TradeRoutesRemote(r1, r2)
	if propagate {
		app.PropagateNewRoutes(r1)

		_, err := app.cl.SendMsg(r2.MachineId, opApi.PropagateRequest{Name: r2.ID()}, "propagate")
		if err != nil {
			return errors.New("couldn't contact machine")
		}
	}
	return nil
}

func (app *Leader) ApplyConnectRouterToRouterRemote(router1ID string, router2ID string, machineId string, linkProps network.LinkProps, propagate bool) error {

	app.topo.Lock()
	r1, ok := app.topo.GetRouter(router1ID)
	app.topo.Unlock()

	if !ok {
		return errors.New("router not found")
	}

	if _, ok = r1.ConnectedRouters[router2ID]; ok {
		return errors.New("router already connected")
	}
	app.topo.Lock()
	r2, ok := app.topo.GetRouter(router2ID)
	app.topo.Unlock()

	if !ok {
		_, err := app.AddRouter(machineId, router2ID)
		if err != nil {
			return err
		}
	}

	r2, _ = app.topo.GetRouter(router2ID)

	router1Channel := make(chan *xdp.Frame, internal.QueueSize)
	conn := app.cl.Endpoints[r2.MachineId]
	d, _ := app.cl.GetNodeDelay(r2.MachineId)
	// Temporary Fix
	app.icm.AddMachine(conn, r2.MachineId)

	app.icm.AddConnection(r2.ID(), d, machineId, r1.ID(), r1.NetworkRouter)

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
	toLink.SetShaper(s)
	s.SetDelay(d)
	toLink.Start()
	//if propagate {
	//	app.PropagateNewRoutes(r1)
	//}
	return nil
}

func (app *Leader) RedirectConnection(r1 *topology.Router, r2 *topology.Router, linkProps network.LinkProps, propagate bool) error {

	b := &internalApi.ConnectRouterToRouterRequest{
		R1:        r1.ID(),
		R2:        r2.ID(),
		MachineID: r2.MachineId,
		Latency:   linkProps.Latency * 2.0,
		Jitter:    linkProps.Jitter * 2.0,
		DropRate:  linkProps.DropRate * 2.0,
		Bandwidth: linkProps.Bandwidth * 8,
		Weight:    linkProps.Weight,
		Propagate: propagate,
	}
	resp, err := app.cl.SendMsg(r1.MachineId, b, "connectRouterToRouter")
	if err != nil {
		return errors.New("couldn't contact machine")
	}

	d := json.NewDecoder(resp.Body)
	req := &internalApi.ConnectRouterToRouterResponse{}
	err = d.Decode(&req)

	if err != nil {
		return errors.New("couldn't decode weights response")
	}
	return nil
}

func (app *Leader) PropagateNewRoutes(r *topology.Router) {
	app.topo.Lock()
	visitedRouters := make(map[string]*topology.Router, app.topo.GetRouterNumber())

	toVisit := make([]*topology.Router, 0)

	toVisit = append(toVisit, r)

	for _, router := range r.ConnectedRouters {
		if router.MachineId == app.GetMachineId() {
			toVisit = append(toVisit, router)
		}
	}

	app.topo.Unlock()

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
	app.topo.Lock()
	defer app.topo.Unlock()
	biLink := r1.RouterLinks[r2.ID()]
	newWeight := biLink.ConnectsTo.NetworkLink.GetProps().Weight

	for mac, weight := range r1.Weights {

		//fmt.Println(r1.ID(), mac)
		if len(net.HardwareAddr(mac)) != 6 {
			continue
		}

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

		//fmt.Println(r2.ID(), mac)
		if len(net.HardwareAddr(mac)) != 6 {
			continue
		}

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
	b := &internalApi.GetRouterWeightsRequest{
		Router: r2.ID(),
	}
	resp, err := app.cl.SendMsg(r2.MachineId, b, "weights")
	if err != nil {
		return errors.New("couldn't contact machine")
	}

	d := json.NewDecoder(resp.Body)
	req := &internalApi.GetRouterWeightsResponse{}
	err = d.Decode(&req)

	if err != nil {
		return errors.New("couldn't decode weights response")
	}

	app.topo.Lock()
	biLink := r1.RouterLinks[r2.ID()]
	newWeight := biLink.ConnectsTo.NetworkLink.GetProps().Weight

	for mac, weight := range req.Weights {

		//fmt.Println("tradeRemote", r2.ID(), mac)

		if len(net.HardwareAddr(mac)) != 6 {
			continue
		}
		//fmt.Println(r1.ID(), net.HardwareAddr(mac), weight)

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

	body := &internalApi.TradeRoutesRequest{
		To:      r2.ID(),
		From:    r1.ID(),
		Weights: r1.Weights,
	}

	resp, err = app.cl.SendMsg(r2.MachineId, body, "trade")
	if err != nil {
		return errors.New("couldn't contact machine")
	}

	app.topo.Unlock()

	d = json.NewDecoder(resp.Body)
	tradeReq := &internalApi.TradeRoutesResponse{}
	err = d.Decode(&tradeReq)

	if err != nil {
		return errors.New("couldn't decode weights response")
	}
	return nil
}

func (app *Leader) ApplyRoutes(to string, from string, weights map[string]topology.Weight) {
	app.topo.Lock()
	r, ok := app.topo.GetRouter(to)

	if !ok {
		return
	}
	biLink := r.RouterLinks[from]
	newWeight := biLink.ConnectsTo.NetworkLink.GetProps().Weight
	for mac, weight := range weights {

		//fmt.Println("ApplyTrade", from, mac)

		if len(net.HardwareAddr(mac)) != 6 {
			continue
		}

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
	app.topo.Unlock()

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
		app.gcLinkShaper(link)
	}

	if n.MachineId == app.GetMachineId() {
		err = app.dm.RemoveNode(nodeId)
		if err != nil {
			return err
		}
	}

	if n.MachineId == app.GetMachineId() {

		body := &internalApi.ClearNodeRequest{
			Id: nodeId,
		}

		resp, err := app.cl.Broadcast(body, http.MethodPost, "clearNode")
		if err != nil {
			return err
		}

		for _, result := range resp {

			d := json.NewDecoder(result.Body)
			req := &internalApi.ClearNodeResponse{}
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
		app.gcLinkShaper(link)
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
		for _, router := range r.ConnectedRouters {
			if router.MachineId != app.GetMachineId() {
				body := &disconnectApi.DisconnectRoutersRequest{
					First:  routerId,
					Second: router.ID(),
				}
				resp, err := app.cl.SendMsg(router.MachineId, body, "localDisconnect")
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
				app.icm.RemoveConnection(router.ID(), r.ID())
			}
		}

	} else {
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

	}

	for _, link := range r.RouterLinks {
		app.gcLinkShaper(link)
	}

	_, err := app.topo.RemoveRouter(routerId)

	if err != nil {
		return err
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
				app.gcLinkShaper(link)
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
			app.gcLinkShaper(link)
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
			app.gcLinkShaper(link)
		}
		if r2.MachineId != app.GetMachineId() {
			app.icm.RemoveConnection(r2.ID(), r1.ID())
			body := &disconnectApi.DisconnectRoutersRequest{
				First:  router2,
				Second: router1,
			}
			resp, err := app.cl.SendMsg(r2.MachineId, body, "localDisconnect")
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
				app.gcLinkShaper(link)
			}
			app.icm.RemoveConnection(r1.ID(), r2.ID())
			body := &disconnectApi.DisconnectRoutersRequest{
				First:  router1,
				Second: router2,
			}
			resp, err := app.cl.SendMsg(r1.MachineId, body, "localDisconnect")
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
			resp, err := app.cl.SendMsg(r1.MachineId, body, "DisconnectRouters")
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
	}
}

func (app *Leader) LocalDisconnect(router1 string, router2 string) error {

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
			app.gcLinkShaper(link)
		}

		if r2.MachineId != app.GetMachineId() {
			app.icm.RemoveConnection(r2.ID(), r1.ID())
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
				app.gcLinkShaper(link)
			}
			app.icm.RemoveConnection(r1.ID(), r2.ID())
			return nil
		} else {
			fmt.Println("tried to disconnect two remote routers in a follower")
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

func (app *Leader) ListSniffers() []api.SniffComponent {
	sniffers := app.redirectManager.ListSniffers()

	list := make([]api.SniffComponent, 0, len(sniffers))

	for id, s := range sniffers {
		link := s.Component.(*topology.BiLink)
		list = append(list, api.SniffComponent{
			Id:        id,
			MachineId: s.MachineId,
			To:        link.To.ID(),
			From:      link.From.ID(),
			Path:      s.Socket.GetSocketPath(),
		})
	}
	if len(app.cl.Nodes) == 0 {
		return list
	} else {

		broadcast, err := app.cl.Broadcast(nil, http.MethodGet, "listSniffers")
		if err != nil {
			return nil
		}
		for _, res := range broadcast {
			response := &opApi.ListSniffersResponse{}

			d := json.NewDecoder(res.Body)
			err = d.Decode(&response)
			for _, i := range response.Sniffers {
				list = append(list, i)
			}
		}
		return list
	}
}

func (app *Leader) StopSniff(id string) error {
	if s, err := app.redirectManager.GetSniffer(id); err != nil {
		return err
	} else {
		if app.GetMachineId() == s.MachineId {
			err = app.clearSniffLink(id)
			if err != nil {
				return err
			}
		} else {
			body := &opApi.StopSniffRequest{
				Id: id,
			}
			resp, err := app.cl.SendMsg(s.MachineId, body, "stopSniff")
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
			app.redirectManager.RemoveSniffer(id)
		}
	}
	return nil
}

func (app *Leader) SniffNode(nodeId string, id string) (string, string, string, error) {

	n, ok := app.topo.GetNode(nodeId)

	if !ok {
		return "", "", "", errors.New(nodeId + " ID doesn't exist")
	}

	if _, err := app.redirectManager.GetSniffer(id); err == nil {
		return "", "", "", errors.New("Already sniffing with " + id)
	}

	if n.MachineId == app.GetMachineId() {

		if n.Bridge == nil {
			return "", "", "", errors.New(id + "is not connected to any bridge")
		}

		if isSpecialLink(n.Link.ConnectsFrom) {
			return "", "", "", errors.New("already performing an operation on this link")
		} else if isSpecialLink(n.Link.ConnectsTo) {
			return "", "", "", errors.New("already performing an operation on this link")
		}

		redirect, err := redirecttraffic.NewRedirectionSocket(id, sniffSocketPath(id))
		if err != nil {
			return "", "", "", err

		}

		sniffComponent := &redirecttraffic.SniffComponent{
			Id:        id,
			MachineId: app.GetMachineId(),
			Socket:    redirect,
			Component: n.Link,
		}

		app.redirectManager.AddSniffer(id, sniffComponent)

		n.Link.NetworkBILink.Left.Pause()
		n.Link.NetworkBILink.Right.Pause()

		newSniff := n.Link.NetworkBILink.Left.GetShaper().(*network.NetworkShaper).ConvertToSniffShaper(sniffComponent)
		n.Link.NetworkBILink.Left.SetShaper(newSniff)
		n.Link.NetworkBILink.Left.Unpause()

		newSniff = n.Link.NetworkBILink.Right.GetShaper().(*network.NetworkShaper).ConvertToSniffShaper(sniffComponent)
		n.Link.NetworkBILink.Right.SetShaper(newSniff)
		n.Link.NetworkBILink.Right.Unpause()

		n.Link.ConnectsFrom.NetworkLink = n.Link.NetworkBILink.Left
		n.Link.ConnectsTo.NetworkLink = n.Link.NetworkBILink.Right

		go redirect.Start()

		return id, redirect.GetSocketPath(), n.MachineId, nil

	} else {
		body := &opApi.SniffNodeRequest{
			Node: nodeId,
			Id:   id,
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

		s := &redirecttraffic.SniffComponent{
			Id:        id,
			MachineId: n.MachineId,
			Socket:    nil,
			Component: nil,
		}

		app.redirectManager.AddSniffer(id, s)

		return req.Id, req.Path, req.MachineId, nil
	}
}

func (app *Leader) SniffBridge(bridgeId string, id string) (string, string, string, error) {

	b, ok := app.topo.GetBridge(bridgeId)

	if !ok {
		return "", "", "", errors.New(bridgeId + " ID doesn't exist")
	}

	if _, err := app.redirectManager.GetSniffer(id); err == nil {
		return "", "", "", errors.New("Already sniffing with " + id)
	}

	if b.MachineId == app.GetMachineId() {

		if b.Router == nil {
			return "", "", "", errors.New(id + "is not connected to any router")
		}

		if isSpecialLink(b.RouterLink.ConnectsFrom) {
			return "", "", "", errors.New("already performing an operation on this link")
		} else if isSpecialLink(b.RouterLink.ConnectsTo) {
			return "", "", "", errors.New("already performing an operation on this link")
		}

		redirect, err := redirecttraffic.NewRedirectionSocket(id, sniffSocketPath(id))

		if err != nil {
			return "", "", "", err

		}

		sniffComponent := &redirecttraffic.SniffComponent{
			Id:        id,
			MachineId: app.GetMachineId(),
			Socket:    redirect,
			Component: b.RouterLink,
		}
		app.redirectManager.AddSniffer(id, sniffComponent)

		b.RouterLink.NetworkBILink.Left.Pause()
		b.RouterLink.NetworkBILink.Right.Pause()

		newSniff := b.RouterLink.NetworkBILink.Left.GetShaper().(*network.NetworkShaper).ConvertToSniffShaper(sniffComponent)
		b.RouterLink.NetworkBILink.Left.SetShaper(newSniff)
		b.RouterLink.NetworkBILink.Left.Unpause()

		newSniff = b.RouterLink.NetworkBILink.Right.GetShaper().(*network.NetworkShaper).ConvertToSniffShaper(sniffComponent)
		b.RouterLink.NetworkBILink.Right.SetShaper(newSniff)
		b.RouterLink.NetworkBILink.Right.Unpause()

		b.RouterLink.ConnectsFrom.NetworkLink = b.RouterLink.NetworkBILink.Left
		b.RouterLink.ConnectsTo.NetworkLink = b.RouterLink.NetworkBILink.Right

		go redirect.Start()

		return id, redirect.GetSocketPath(), b.MachineId, nil

	} else {
		body := &opApi.SniffBridgeRequest{
			Bridge: bridgeId,
			Id:     id,
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

		s := &redirecttraffic.SniffComponent{
			Id:        id,
			MachineId: b.MachineId,
			Socket:    nil,
			Component: nil,
		}
		app.redirectManager.AddSniffer(id, s)

		return req.Id, req.Path, req.MachineId, nil
	}
}
func (app *Leader) SniffRouters(router1 string, router2 string, id string) (string, string, string, error) {

	r1, ok := app.topo.GetRouter(router1)

	if !ok {
		return "", "", "", errors.New(router1 + " ID doesn't exist")
	}

	r2, ok := app.topo.GetRouter(router2)

	if !ok {
		return "", "", "", errors.New(router2 + " ID doesn't exist")
	}

	if _, err := app.redirectManager.GetSniffer(id); err == nil {
		return "", "", "", errors.New("Already sniffing with " + id)
	}

	if r1.MachineId == app.GetMachineId() {
		if r2.MachineId == app.GetMachineId() {
			link, ok := r1.RouterLinks[router2]
			if !ok {
				return "", "", "", errors.New(router1 + " and " + router2 + "are not connected")
			}

			if isSpecialLink(link.ConnectsFrom) {
				return "", "", "", errors.New("already performing an operation on this link")
			} else if isSpecialLink(link.ConnectsTo) {
				return "", "", "", errors.New("already performing an operation on this link")
			}

			redirect, err := redirecttraffic.NewRedirectionSocket(router1+"-"+router2, sniffSocketPath(id))
			if err != nil {
				return "", "", "", err

			}

			sniffComponent := &redirecttraffic.SniffComponent{
				Id:        id,
				MachineId: app.GetMachineId(),
				Socket:    redirect,
				Component: link,
			}

			app.redirectManager.AddSniffer(id, sniffComponent)

			link.NetworkBILink.Left.Pause()
			link.NetworkBILink.Right.Pause()

			newSniff := link.NetworkBILink.Left.GetShaper().(*network.NetworkShaper).ConvertToSniffShaper(sniffComponent)
			link.NetworkBILink.Left.SetShaper(newSniff)
			link.NetworkBILink.Left.Unpause()

			newSniff = link.NetworkBILink.Right.GetShaper().(*network.NetworkShaper).ConvertToSniffShaper(sniffComponent)
			link.NetworkBILink.Right.SetShaper(newSniff)
			link.NetworkBILink.Right.Unpause()

			link.ConnectsFrom.NetworkLink = link.NetworkBILink.Left
			link.ConnectsTo.NetworkLink = link.NetworkBILink.Right

			go redirect.Start()

			return id, redirect.GetSocketPath(), r1.MachineId, nil
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
				Id:      id,
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

			s := &redirecttraffic.SniffComponent{
				Id:        id,
				MachineId: r2.MachineId,
				Socket:    nil,
				Component: nil,
			}

			app.redirectManager.AddSniffer(id, s)

			return req.Id, req.Path, req.MachineId, nil
		}
	}
}

func (app *Leader) InterceptNode(nodeId string, id string, direction bool) (string, string, string, error) {

	n, ok := app.topo.GetNode(nodeId)

	if !ok {
		return "", "", "", errors.New(nodeId + " ID doesn't exist")
	}

	if _, err := app.redirectManager.GetIntercept(id); err == nil {
		return "", "", "", errors.New("Already intercepting with " + id)
	}

	if n.MachineId == app.GetMachineId() {

		if n.Bridge == nil {
			return "", "", "", errors.New(nodeId + "is not connected to any bridge")
		}

		var interceptComponent *redirecttraffic.InterceptComponent

		if direction {

			if isSpecialLink(n.Link.ConnectsFrom) {
				return "", "", "", errors.New("already performing an operation on this link")
			}
			redirect, err := redirecttraffic.NewRedirectionSocket(id, interceptSocketPath(id))
			if err != nil {
				return "", "", "", err
			}
			interceptComponent = &redirecttraffic.InterceptComponent{
				Id:        id,
				MachineId: app.GetMachineId(),
				Socket:    redirect,
				Component: n.Link.ConnectsFrom,
			}

			n.Link.NetworkBILink.Left.Pause()

			intercept := n.Link.NetworkBILink.Left.GetShaper().(*network.NetworkShaper).ConvertToInterceptShaper(interceptComponent)
			n.Link.NetworkBILink.Left.SetShaper(intercept)
			n.Link.NetworkBILink.Left.Unpause()
			n.Link.ConnectsFrom.NetworkLink = n.Link.NetworkBILink.Left

		} else {

			if isSpecialLink(n.Link.ConnectsTo) {
				return "", "", "", errors.New("already performing an operation on this link")
			}

			redirect, err := redirecttraffic.NewRedirectionSocket(id, interceptSocketPath(id))
			if err != nil {
				return "", "", "", err
			}
			interceptComponent = &redirecttraffic.InterceptComponent{
				Id:        id,
				MachineId: app.GetMachineId(),
				Socket:    redirect,
				Component: n.Link.ConnectsTo,
			}

			n.Link.NetworkBILink.Right.Pause()
			intercept := n.Link.NetworkBILink.Right.GetShaper().(*network.NetworkShaper).ConvertToInterceptShaper(interceptComponent)
			n.Link.NetworkBILink.Right.SetShaper(intercept)
			n.Link.NetworkBILink.Right.Unpause()
			n.Link.ConnectsTo.NetworkLink = n.Link.NetworkBILink.Right

		}

		app.redirectManager.AddIntercept(id, interceptComponent)

		go interceptComponent.Socket.Start()

		return id, interceptComponent.Socket.GetSocketPath(), n.MachineId, nil

	} else {
		body := &opApi.InterceptNodeRequest{
			Node:      nodeId,
			Id:        id,
			Direction: direction,
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

		interceptComponent := &redirecttraffic.InterceptComponent{
			Id:        id,
			MachineId: n.MachineId,
			Socket:    nil,
			Component: nil,
		}

		app.redirectManager.AddIntercept(id, interceptComponent)

		return req.Id, req.Path, req.MachineId, nil
	}
}

func (app *Leader) InterceptBridge(bridgeId string, id string, direction bool) (string, string, string, error) {

	b, ok := app.topo.GetBridge(bridgeId)

	if !ok {
		return "", "", "", errors.New(bridgeId + " ID doesn't exist")
	}

	if _, err := app.redirectManager.GetIntercept(id); err == nil {
		return "", "", "", errors.New("Already intercepting with " + id)
	}

	if b.MachineId == app.GetMachineId() {

		if b.Router == nil {
			return "", "", "", errors.New(bridgeId + "is not connected to any router")
		}

		var interceptComponent *redirecttraffic.InterceptComponent

		if direction {

			if isSpecialLink(b.RouterLink.ConnectsFrom) {
				return "", "", "", errors.New("already performing an operation on this link")
			}

			redirect, err := redirecttraffic.NewRedirectionSocket(id, interceptSocketPath(id))
			if err != nil {
				return "", "", "", err
			}

			interceptComponent = &redirecttraffic.InterceptComponent{
				Id:        id,
				MachineId: app.GetMachineId(),
				Socket:    redirect,
				Component: b.RouterLink.ConnectsFrom,
			}
			b.RouterLink.NetworkBILink.Left.Pause()

			intercept := b.RouterLink.NetworkBILink.Left.GetShaper().(*network.NetworkShaper).ConvertToInterceptShaper(interceptComponent)
			b.RouterLink.NetworkBILink.Left.SetShaper(intercept)
			b.RouterLink.NetworkBILink.Left.Unpause()
			b.RouterLink.ConnectsFrom.NetworkLink = b.RouterLink.NetworkBILink.Left

		} else {

			if isSpecialLink(b.RouterLink.ConnectsTo) {
				return "", "", "", errors.New("already performing an operation on this link")
			}
			redirect, err := redirecttraffic.NewRedirectionSocket(id, interceptSocketPath(id))
			if err != nil {
				return "", "", "", err
			}

			interceptComponent = &redirecttraffic.InterceptComponent{
				Id:        id,
				MachineId: app.GetMachineId(),
				Socket:    redirect,
				Component: b.RouterLink.ConnectsTo,
			}

			b.RouterLink.NetworkBILink.Right.Pause()

			intercept := b.RouterLink.NetworkBILink.Right.GetShaper().(*network.NetworkShaper).ConvertToInterceptShaper(interceptComponent)
			b.RouterLink.NetworkBILink.Right.SetShaper(intercept)
			b.RouterLink.NetworkBILink.Right.Unpause()
			b.RouterLink.ConnectsTo.NetworkLink = b.RouterLink.NetworkBILink.Right
		}

		app.redirectManager.AddIntercept(id, interceptComponent)

		go interceptComponent.Socket.Start()

		return id, interceptComponent.Socket.GetSocketPath(), b.MachineId, nil

	} else {
		body := &opApi.InterceptBridgeRequest{
			Bridge:    bridgeId,
			Id:        id,
			Direction: direction,
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

		interceptComponent := redirecttraffic.InterceptComponent{
			Id:        id,
			MachineId: b.MachineId,
			Socket:    nil,
			Component: nil,
		}

		app.redirectManager.AddIntercept(id, &interceptComponent)

		return req.Id, req.Path, req.MachineId, nil
	}
}

func (app *Leader) InterceptRouters(router1 string, router2 string, id string, direction bool) (string, string, string, error) {

	r1, ok := app.topo.GetRouter(router1)

	if !ok {
		return "", "", "", errors.New(router1 + " ID doesn't exist")
	}

	r2, ok := app.topo.GetRouter(router2)

	if !ok {
		return "", "", "", errors.New(router2 + " ID doesn't exist")
	}

	if _, err := app.redirectManager.GetIntercept(id); err == nil {
		return "", "", "", errors.New("Already intercepting with " + id)
	}

	if r1.MachineId == app.GetMachineId() {
		if r2.MachineId == app.GetMachineId() {

			link, ok := r1.RouterLinks[router2]
			if !ok {
				return "", "", "", errors.New(router1 + " and " + router2 + "are not connected")
			}

			var interceptComponent *redirecttraffic.InterceptComponent

			if link.To.ID() == r2.ID() {

				if direction {
					if isSpecialLink(link.ConnectsFrom) {
						return "", "", "", errors.New("already performing an operation on this link")
					}

					redirect, err := redirecttraffic.NewRedirectionSocket(id, interceptSocketPath(id))

					if err != nil {
						return "", "", "", err

					}

					interceptComponent = &redirecttraffic.InterceptComponent{
						Id:        id,
						MachineId: app.GetMachineId(),
						Socket:    redirect,
						Component: link.ConnectsFrom,
					}

					link.ConnectsFrom.NetworkLink.Pause()

					intercept := link.ConnectsFrom.NetworkLink.GetShaper().(*network.NetworkShaper).ConvertToInterceptShaper(interceptComponent)
					link.ConnectsFrom.NetworkLink.SetShaper(intercept)
					link.ConnectsFrom.NetworkLink.Unpause()

				} else {
					if isSpecialLink(link.ConnectsTo) {
						return "", "", "", errors.New("already performing an operation on this link")
					}
					redirect, err := redirecttraffic.NewRedirectionSocket(id, interceptSocketPath(id))

					if err != nil {
						return "", "", "", err

					}

					interceptComponent = &redirecttraffic.InterceptComponent{
						Id:        id,
						MachineId: app.GetMachineId(),
						Socket:    redirect,
						Component: link.ConnectsTo,
					}

					link.ConnectsTo.NetworkLink.Pause()

					intercept := link.ConnectsTo.NetworkLink.GetShaper().(*network.NetworkShaper).ConvertToInterceptShaper(interceptComponent)
					link.ConnectsTo.NetworkLink.SetShaper(intercept)
					link.ConnectsTo.NetworkLink.Unpause()
				}
			} else {
				if direction {
					if isSpecialLink(link.ConnectsTo) {
						return "", "", "", errors.New("already performing an operation on this link")
					}
					redirect, err := redirecttraffic.NewRedirectionSocket(id, interceptSocketPath(id))

					if err != nil {
						return "", "", "", err

					}

					interceptComponent = &redirecttraffic.InterceptComponent{
						Id:        id,
						MachineId: app.GetMachineId(),
						Socket:    redirect,
						Component: link.ConnectsTo,
					}

					link.ConnectsTo.NetworkLink.Pause()

					intercept := link.ConnectsTo.NetworkLink.GetShaper().(*network.NetworkShaper).ConvertToInterceptShaper(interceptComponent)
					link.ConnectsTo.NetworkLink.SetShaper(intercept)
					link.ConnectsTo.NetworkLink.Unpause()

				} else {
					if isSpecialLink(link.ConnectsFrom) {
						return "", "", "", errors.New("already performing an operation on this link")
					}

					redirect, err := redirecttraffic.NewRedirectionSocket(id, interceptSocketPath(id))

					if err != nil {
						return "", "", "", err

					}

					interceptComponent = &redirecttraffic.InterceptComponent{
						Id:        id,
						MachineId: app.GetMachineId(),
						Socket:    redirect,
						Component: link.ConnectsFrom,
					}

					link.ConnectsFrom.NetworkLink.Pause()

					intercept := link.ConnectsFrom.NetworkLink.GetShaper().(*network.NetworkShaper).ConvertToInterceptShaper(interceptComponent)
					link.ConnectsFrom.NetworkLink.SetShaper(intercept)
					link.ConnectsFrom.NetworkLink.Unpause()
				}
			}

			app.redirectManager.AddIntercept(id, interceptComponent)

			go interceptComponent.Socket.Start()

			return id, interceptComponent.Socket.GetSocketPath(), r1.MachineId, nil
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

			interceptComponent := &redirecttraffic.InterceptComponent{
				Id:        id,
				MachineId: r1.MachineId,
				Socket:    nil,
				Component: nil,
			}

			app.redirectManager.AddIntercept(id, interceptComponent)

			return req.Id, req.Path, req.MachineId, nil
		}
	}
}

func (app *Leader) StopIntercept(id string) error {
	if s, err := app.redirectManager.GetIntercept(id); err != nil {
		return err
	} else {
		if app.GetMachineId() == s.MachineId {
			err = app.clearInterceptLink(id)
			if err != nil {
				return err
			}
		} else {
			body := &opApi.StopInterceptRequest{
				Id: id,
			}
			resp, err := app.cl.SendMsg(s.MachineId, body, "stopIntercept")
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

			app.redirectManager.RemoveIntercept(id)
		}
	}
	return nil
}

func (app *Leader) ListIntercepts() []api.InterceptComponent {
	intercepts := app.redirectManager.ListIntercepts()

	list := make([]api.InterceptComponent, 0, len(intercepts))

	for id, s := range intercepts {
		link := s.Component.(*topology.Link)
		list = append(list, api.InterceptComponent{
			Id:        id,
			MachineId: s.MachineId,
			To:        link.To.ID(),
			From:      link.From.ID(),
			Path:      s.Socket.GetSocketPath(),
		})
	}
	if len(app.cl.Nodes) == 0 {
		return list
	} else {

		broadcast, err := app.cl.Broadcast(nil, http.MethodGet, "listIntercepts")
		if err != nil {
			return nil
		}
		for _, res := range broadcast {
			response := &opApi.ListInterceptsResponse{}

			d := json.NewDecoder(res.Body)
			err = d.Decode(&response)
			for _, i := range response.Intercepts {
				list = append(list, i)
			}
		}
		return list
	}
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

func (app *Leader) DisruptNode(id string) error {
	if n, ok := app.topo.GetNode(id); ok {
		if n.MachineId == app.GetMachineId() {
			if n.Link.NetworkBILink.Disrupt() {
				return nil
			} else {
				return errors.New("could not disrupt link")
			}
		} else {

			resp, err := app.cl.SendMsg(n.MachineId, &opApi.DisruptNodeRequest{Node: id}, "disruptNode")
			if err != nil {
				return err
			}

			d := json.NewDecoder(resp.Body)
			req := &opApi.DisruptNodeResponse{}
			err = d.Decode(&req)

			if err != nil {
				return err
			}

			if req.Error.ErrCode != 0 {
				return errors.New(req.Error.ErrMsg)
			}
			return nil
		}
	} else {
		return errors.New("invalid node id")
	}
}

func (app *Leader) DisruptBridge(id string) error {
	if b, ok := app.topo.GetBridge(id); ok {
		if b.MachineId == app.GetMachineId() {
			if b.RouterLink.NetworkBILink.Disrupt() {
				return nil
			} else {
				return errors.New("could not disrupt link")
			}
		} else {

			resp, err := app.cl.SendMsg(b.MachineId, &opApi.DisruptBridgeRequest{Bridge: id}, "disruptBridge")
			if err != nil {
				return err
			}

			d := json.NewDecoder(resp.Body)
			req := &opApi.DisruptBridgeResponse{}
			err = d.Decode(&req)

			if err != nil {
				return err
			}

			if req.Error.ErrCode != 0 {
				return errors.New(req.Error.ErrMsg)
			}
			return nil
		}
	} else {
		return errors.New("invalid bridge id")
	}
}

func (app *Leader) DisruptRouters(router1Id string, router2Id string) error {
	app.topo.Lock()
	defer app.topo.Unlock()
	if r1, ok := app.topo.GetRouter(router1Id); !ok {
		return errors.New("invalid router id: " + router1Id)
	} else if r2, ok := app.topo.GetRouter(router2Id); !ok {
		return errors.New("invalid router id: " + router2Id)
	} else if !(r1.MachineId == r2.MachineId) {
		return errors.New("could not disrupt connnection between remote routers")
	} else if r1.MachineId == app.GetMachineId() {
		if l, ok := r1.RouterLinks[r2.ID()]; ok {
			if l.NetworkBILink.Disrupt() {
				return nil
			} else {
				return errors.New("could not stop link")
			}
		} else {
			return errors.New("could not disrupt link")
		}
	} else {
		resp, err := app.cl.SendMsg(r1.MachineId, &opApi.DisruptRoutersRequest{
			Router1: r1.ID(),
			Router2: r2.ID(),
		}, "disruptRouters")

		if err != nil {
			return err
		}

		d := json.NewDecoder(resp.Body)
		req := &opApi.DisruptRoutersResponse{}
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
func (app *Leader) StopDisruptNode(id string) error {
	if n, ok := app.topo.GetNode(id); ok {
		if n.MachineId == app.GetMachineId() {
			if n.Link.NetworkBILink.StopDisrupt() {
				return nil
			} else {
				return errors.New("could not stop link")
			}
		} else {

			resp, err := app.cl.SendMsg(n.MachineId, &opApi.DisruptNodeRequest{Node: id}, "stopDisruptNode")
			if err != nil {
				return err
			}

			d := json.NewDecoder(resp.Body)
			req := &opApi.DisruptNodeResponse{}
			err = d.Decode(&req)

			if err != nil {
				return err
			}

			if req.Error.ErrCode != 0 {
				return errors.New(req.Error.ErrMsg)
			}
			return nil
		}
	} else {
		return errors.New("invalid node id")
	}
}

func (app *Leader) StopDisruptBridge(id string) error {
	if b, ok := app.topo.GetBridge(id); ok {
		if b.MachineId == app.GetMachineId() {
			if b.RouterLink.NetworkBILink.StopDisrupt() {
				return nil
			} else {
				return errors.New("could not stop link")
			}
		} else {

			resp, err := app.cl.SendMsg(b.MachineId, &opApi.DisruptBridgeRequest{Bridge: id}, "stopDisruptBridge")
			if err != nil {
				return err
			}

			d := json.NewDecoder(resp.Body)
			req := &opApi.DisruptBridgeResponse{}
			err = d.Decode(&req)

			if err != nil {
				return err
			}

			if req.Error.ErrCode != 0 {
				return errors.New(req.Error.ErrMsg)
			}
			return nil
		}
	} else {
		return errors.New("invalid bridge id")
	}
}

func (app *Leader) StopDisruptRouters(router1Id string, router2Id string) error {

	if r1, ok := app.topo.GetRouter(router1Id); !ok {
		return errors.New("invalid router id: " + router1Id)
	} else if r2, ok := app.topo.GetRouter(router2Id); !ok {
		return errors.New("invalid router id: " + router2Id)
	} else if !(r1.MachineId == r2.MachineId) {
		return errors.New("could not stop disrupt connnection between remote routers")
	} else if r1.MachineId == app.GetMachineId() {
		if r1.RouterLinks[r2.ID()].NetworkBILink.StopDisrupt() {
			return nil
		} else {
			return errors.New("could not stop link")
		}
	} else {
		resp, err := app.cl.SendMsg(r1.MachineId, &opApi.DisruptRoutersRequest{
			Router1: r1.ID(),
			Router2: r2.ID(),
		}, "stopDisruptRouters")

		if err != nil {
			return err
		}

		d := json.NewDecoder(resp.Body)
		req := &opApi.DisruptRoutersResponse{}
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

func (app *Leader) StopBridge(id string) error {
	if b, ok := app.topo.GetBridge(id); ok {
		if b.MachineId == app.GetMachineId() {
			if b.NetworkBridge.Disrupt() {
				return nil
			} else {
				return errors.New("could not disrupt bridge")
			}
		} else {

			resp, err := app.cl.SendMsg(b.MachineId, &opApi.StopBridgeRequest{Bridge: id}, "stopBridge")
			if err != nil {
				return err
			}

			d := json.NewDecoder(resp.Body)
			req := &opApi.StopBridgeResponse{}
			err = d.Decode(&req)

			if err != nil {
				return err
			}

			if req.Error.ErrCode != 0 {
				return errors.New(req.Error.ErrMsg)
			}
			return nil
		}
	} else {
		return errors.New("invalid bridge id")
	}
}

func (app *Leader) StopRouter(id string) error {

	if r, ok := app.topo.GetRouter(id); ok {
		if r.MachineId == app.GetMachineId() {
			if r.NetworkRouter.Disrupt() {
				return nil
			} else {
				return errors.New("could not disrupt router")
			}
		} else {

			resp, err := app.cl.SendMsg(r.MachineId, &opApi.StopRouterRequest{Router: id}, "stopRouter")
			if err != nil {
				return err
			}

			d := json.NewDecoder(resp.Body)
			req := &opApi.StopRouterResponse{}
			err = d.Decode(&req)

			if err != nil {
				return err
			}

			if req.Error.ErrCode != 0 {
				return errors.New(req.Error.ErrMsg)
			}
			return nil
		}
	} else {
		return errors.New("invalid router id")
	}
}
func (app *Leader) StartBridge(id string) error {
	if b, ok := app.topo.GetBridge(id); ok {
		if b.MachineId == app.GetMachineId() {
			if b.NetworkBridge.StopDisrupt() {
				return nil
			} else {
				return errors.New("could not disrupt bridge")
			}
		} else {

			resp, err := app.cl.SendMsg(b.MachineId, &opApi.StartBridgeRequest{Bridge: id}, "startBridge")
			if err != nil {
				return err
			}

			d := json.NewDecoder(resp.Body)
			req := &opApi.StartBridgeResponse{}
			err = d.Decode(&req)

			if err != nil {
				return err
			}

			if req.Error.ErrCode != 0 {
				return errors.New(req.Error.ErrMsg)
			}
			return nil
		}
	} else {
		return errors.New("invalid bridge id")
	}
}

func (app *Leader) StartRouter(id string) error {

	if r, ok := app.topo.GetRouter(id); ok {
		if r.MachineId == app.GetMachineId() {
			if r.NetworkRouter.StopDisrupt() {
				return nil
			} else {
				return errors.New("could not disrupt router")
			}
		} else {

			resp, err := app.cl.SendMsg(r.MachineId, &opApi.StartRouterRequest{Router: id}, "startRouter")
			if err != nil {
				return err
			}

			d := json.NewDecoder(resp.Body)
			req := &opApi.StartRouterResponse{}
			err = d.Decode(&req)

			if err != nil {
				return err
			}

			if req.Error.ErrCode != 0 {
				return errors.New(req.Error.ErrMsg)
			}
			return nil
		}
	} else {
		return errors.New("invalid router id")
	}
}

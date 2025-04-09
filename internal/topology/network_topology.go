package topology

import (
	"errors"
	"fmt"
	"github.com/David-Antunes/gone-proxy/xdp"
	"github.com/David-Antunes/gone/internal"
	"github.com/David-Antunes/gone/internal/network"
	"net"
	"strconv"
	"sync"
	"time"
)

type LocalFlowManager interface {
	AddMac([]byte, chan *xdp.Frame, chan *xdp.Frame)
	RemoveMac([]byte)
	GetOutgoingChannel(string) (chan *xdp.Frame, bool)
	GetIncomingChannel(string) (chan *xdp.Frame, bool)
}

type RemoteFlowManager interface {
	AddMac([]byte, chan *xdp.Frame)
	RemoveMac([]byte)
	GetOutboundChannel() chan *xdp.Frame
	GetMacChannel() chan *xdp.Frame
}
type Topology struct {
	sync.Mutex
	machineId string
	nodes     map[string]*Node
	macs      map[string]*Node
	bridges   map[string]*Bridge
	routers   map[string]*Router
	links     map[string]*BiLink
	fl        LocalFlowManager
	delay     *network.DynamicDelay
}

func CreateTopology(machineId string, fl LocalFlowManager, delay *network.DynamicDelay) *Topology {
	return &Topology{
		Mutex:     sync.Mutex{},
		machineId: machineId,
		nodes:     make(map[string]*Node),
		macs:      make(map[string]*Node),
		bridges:   make(map[string]*Bridge),
		routers:   make(map[string]*Router),
		links:     map[string]*BiLink{},
		fl:        fl,
		delay:     delay,
	}
}

func (topo *Topology) GetRouterNumber() int {
	return len(topo.routers)
}
func (topo *Topology) GetNode(id string) (*Node, bool) {
	n, ok := topo.nodes[id]
	return n, ok
}

func (topo *Topology) GetNodeFromMac(mac string) (*Node, bool) {
	n, ok := topo.macs[mac]
	return n, ok
}

func (topo *Topology) GetBridge(id string) (*Bridge, bool) {
	b, ok := topo.bridges[id]
	return b, ok
}

func (topo *Topology) GetRouter(id string) (*Router, bool) {
	r, ok := topo.routers[id]
	return r, ok
}

func (topo *Topology) GetBiLink(id string) (*BiLink, bool) {
	l, ok := topo.links[id]
	return l, ok
}

func (topo *Topology) RegisterNode(id string, mac string, machineId string) (Node, error) {
	topo.Lock()
	defer topo.Unlock()
	if n, ok := topo.nodes[id]; ok {
		return *n, errors.New("node is already registered")
	}
	var n *network.Node = nil
	if topo.machineId == machineId {

		incoming := make(chan *xdp.Frame, internal.QueueSize)
		outgoing := make(chan *xdp.Frame, internal.QueueSize)

		topo.fl.AddMac([]byte(mac), incoming, outgoing)
		n = network.CreateNode(string(internal.ConvertMacStringToBytes(mac)), incoming, outgoing, network.CreateBILink(network.CreateNullLink(incoming), nil))
		n.GetLink().GetLeft().Start()

	} else {
		n = network.CreateNode(string(internal.ConvertMacStringToBytes(mac)), nil, nil, nil)
	}

	component := &Node{
		Id:          id,
		NetworkNode: n,
		Link:        nil,
		Bridge:      nil,
		MachineId:   machineId,
	}
	topo.nodes[id] = component
	topo.macs[string(internal.ConvertMacStringToBytes(mac))] = component
	return *component, nil
}

func (topo *Topology) RegisterBridge(id string, machineId string) (Bridge, error) {
	topo.Lock()
	defer topo.Unlock()
	if b, ok := topo.bridges[id]; ok {
		return *b, errors.New("bridge already registered")
	}

	var b *network.Bridge = nil
	if topo.machineId == machineId {
		b = network.CreateBridge()
		b.SetGateway(internal.GetNullChan())
		b.Start()
	}

	component := &Bridge{
		Id:             id,
		NetworkBridge:  b,
		Router:         nil,
		RouterLink:     nil,
		ConnectedNodes: make(map[string]*Node),
		NodeLinks:      make(map[string]*BiLink),
		MachineId:      machineId,
	}
	topo.bridges[id] = component

	return *component, nil
}

func (topo *Topology) RegisterRouter(id string, machineId string) (Router, error) {
	topo.Lock()
	defer topo.Unlock()
	if r, ok := topo.routers[id]; ok {
		return *r, errors.New("router already registered")
	}
	var r *network.Router = nil
	if topo.machineId == machineId {

		r = network.CreateRouter(id)
		r.Start()
	}

	component := &Router{
		Id:               id,
		NetworkRouter:    r,
		ConnectedRouters: make(map[string]*Router),
		RouterLinks:      make(map[string]*BiLink),
		ConnectedBridges: make(map[string]*Bridge),
		BridgeLinks:      make(map[string]*BiLink),
		Weights:          make(map[string]Weight),
		MachineId:        machineId,
	}
	topo.routers[id] = component

	return *component, nil
}

func (topo *Topology) registerBiLink(toComponent Component, fromComponent Component, networkBiLink *network.BiLink) *BiLink {
	toId := "link" + strconv.Itoa(len(topo.links)) + "-1"
	fromId := "link" + strconv.Itoa(len(topo.links)) + "-2"
	linkId := "BiLink" + strconv.Itoa(len(topo.links))

	toLinkComponent := &Link{
		Id:          toId,
		NetworkLink: networkBiLink.Left,
		From:        toComponent,
		To:          fromComponent,
	}
	fromLinkComponent := &Link{
		Id:          fromId,
		NetworkLink: networkBiLink.Right,
		From:        fromComponent,
		To:          toComponent,
	}
	biLink := &BiLink{
		Id:            linkId,
		NetworkBILink: networkBiLink,
		ConnectsTo:    toLinkComponent,
		ConnectsFrom:  fromLinkComponent,
		To:            toComponent,
		From:          fromComponent,
	}
	topo.links[linkId] = biLink

	return biLink
}

func (topo *Topology) registerRemoteBiLink(toComponent Component, fromComponent Component) *BiLink {
	toId := "link" + strconv.Itoa(len(topo.links)) + "-1"
	fromId := "link" + strconv.Itoa(len(topo.links)) + "-2"
	linkId := "BiLink" + strconv.Itoa(len(topo.links))

	toLinkComponent := &Link{
		Id:          toId,
		NetworkLink: nil,
		From:        toComponent,
		To:          fromComponent,
	}
	fromLinkComponent := &Link{
		Id:          fromId,
		NetworkLink: nil,
		From:        fromComponent,
		To:          toComponent,
	}
	biLink := &BiLink{
		Id:            linkId,
		NetworkBILink: nil,
		ConnectsTo:    toLinkComponent,
		ConnectsFrom:  fromLinkComponent,
		To:            toComponent,
		From:          fromComponent,
	}
	topo.links[linkId] = biLink

	return biLink
}

func (topo *Topology) ConnectNodeToBridge(nodeID string, bridgeID string, linkProps network.LinkProps) (*BiLink, error) {
	topo.Lock()
	defer topo.Unlock()

	var n *Node
	var b *Bridge
	var ok bool

	if n, ok = topo.nodes[nodeID]; !ok {
		return nil, errors.New(nodeID + " ID doesn't exist")
	}

	if b, ok = topo.bridges[bridgeID]; !ok {
		return nil, errors.New(bridgeID + " ID doesn't exist")
	}

	if n.Bridge != nil {
		return nil, errors.New(nodeID + " is already connected to a bridge")
	}

	if _, ok = b.ConnectedNodes[nodeID]; ok {
		return nil, errors.New(bridgeID + " is already connected to " + nodeID)
	}

	if n.MachineId != b.MachineId {
		return nil, errors.New("can't connect a node and bridge in different machines")
	}

	if n.MachineId != topo.machineId {
		link := topo.registerRemoteBiLink(n, b)
		n.SetBridge(b, link)
		b.AddNode(n, link)
		return link, nil
	}
	n.NetworkNode.GetLink().Left.Close()
	biLink := network.ConnectNodeToBridge(n.NetworkNode, b.NetworkBridge, linkProps)
	topoLink := topo.registerBiLink(n, b, biLink)
	n.SetBridge(b, topoLink)
	b.AddNode(n, topoLink)

	biLink.Left.GetShaper().SetDelay(topo.delay.ReceiveDelay)
	biLink.Right.GetShaper().SetDelay(topo.delay.TransmitDelay)
	biLink.Start()

	if b.Router != nil {
		b.Router.AddWeight(n.NetworkNode.GetMac(), b.Router.ID(), 0)
		b.Router.NetworkRouter.AddNode([]byte(n.NetworkNode.GetMac()), b.NetworkBridge.Link().Right.GetOriginChan())
		fmt.Println("Added ", n.ID(), " to router from bridge", b.ID())
	}

	return topoLink, nil
}

func (topo *Topology) ConnectBridgeToRouter(bridgeID string, routerID string, linkProps network.LinkProps) (*BiLink, error) {
	topo.Lock()
	defer topo.Unlock()
	var b *Bridge
	var r *Router
	var ok bool

	if b, ok = topo.bridges[bridgeID]; !ok {
		return nil, errors.New(bridgeID + " ID doesn't exist")
	}

	if r, ok = topo.routers[routerID]; !ok {
		return nil, errors.New(routerID + " ID doesn't exist")
	}

	if b.Router != nil {
		return nil, errors.New(bridgeID + " is already connected to a router")
	}

	if _, ok = r.ConnectedBridges[bridgeID]; ok {
		return nil, errors.New(routerID + " is already connected to " + bridgeID)
	}

	if b.MachineId != r.MachineId {
		return nil, errors.New("can't connect a node and bridge in different machines")
	}

	if topo.machineId != b.MachineId {
		topoLink := topo.registerRemoteBiLink(b, r)

		b.SetRouter(r, topoLink)
		r.AddBridge(b, topoLink)
		return topoLink, nil
	}

	biLink := network.ConnectBridgeToRouter(b.NetworkBridge, r.NetworkRouter, linkProps)
	topoLink := topo.registerBiLink(b, r, biLink)

	b.SetRouter(r, topoLink)
	r.AddBridge(b, topoLink)

	biLink.Start()

	for _, netNode := range b.ConnectedNodes {
		r.AddWeight(netNode.NetworkNode.GetMac(), r.ID(), 0)
		r.NetworkRouter.AddNode([]byte(netNode.NetworkNode.GetMac()), biLink.Right.GetOriginChan())
		fmt.Println("Added ", netNode.ID(), " to router from bridge", routerID)
	}

	return topoLink, nil
}

func (topo *Topology) ConnectRouterToRouterLocal(router1 string, router2 string, linkProps network.LinkProps) (*BiLink, error) {

	topo.Lock()
	defer topo.Unlock()

	var r1 *Router
	var r2 *Router
	var ok bool

	if r1, ok = topo.routers[router1]; !ok {
		return nil, errors.New(router1 + " ID doesn't exist")
	}

	if r2, ok = topo.routers[router2]; !ok {
		return nil, errors.New(router2 + " ID doesn't exist")
	}

	if _, ok = r1.ConnectedRouters[router2]; ok {
		return nil, errors.New(router1 + " is already connected to " + router2)
	}

	if _, ok = r2.ConnectedRouters[router1]; ok {
		return nil, errors.New(router1 + " is already connected to " + router2)
	}

	biLink := network.ConnectRouterToRouter(r1.NetworkRouter, r2.NetworkRouter, linkProps)
	link := topo.registerBiLink(r1, r2, biLink)

	r1.AddRouter(r2, link)
	r2.AddRouter(r1, link)

	biLink.Start()

	return link, nil
}

func (topo *Topology) InsertNewPath(path []string, frame *xdp.Frame, distance int) {
	topo.Lock()
	defer topo.Unlock()
	currDistance := distance
	router1 := topo.routers[path[0]]

	for i := 1; i < len(path); i++ {
		router2 := topo.routers[path[i]]
		if router1.MachineId == topo.machineId {
			currDistance = AddNewMacBetweenRouters(router1, router2, frame.GetMacDestination(), currDistance)
			if currDistance < 0 {
				fmt.Println("weight is negative!!!")
			}
		} else {
			break
		}
		router1 = router2
	}
}

func (topo *Topology) InsertLocalPath(path []string, frame *xdp.Frame, distance int) {
	topo.Lock()
	defer topo.Unlock()
	currDistance := distance
	router1 := topo.routers[path[0]]
	router2 := topo.routers[path[1]]
	if router1.MachineId == topo.machineId {
		currDistance = AddNewMacBetweenRouters(router1, router2, frame.GetMacDestination(), currDistance)
		if currDistance < 0 {
			fmt.Println("weight is negative!!!")
		}
	}
}

func (topo *Topology) InsertNullPath(mac string, routerId string) {
	topo.Lock()

	router := topo.routers[routerId]
	if router == nil {
		return
	}
	router.NetworkRouter.AddNode([]byte(mac), internal.GetNullChan())
	topo.Unlock()

	go func(m string, rId string) {
		fmt.Println("Inserting null path for 10 seconds", net.HardwareAddr(m), rId)
		time.Sleep(1 * time.Second)
		topo.Lock()
		r := topo.routers[rId]
		r.NetworkRouter.RemoveNode([]byte(m))
		fmt.Println("Removed", net.HardwareAddr(m), rId)
		topo.Unlock()
	}(mac, routerId)
}

func (topo *Topology) RemoveNode(nodeId string) (*Node, error) {
	topo.Lock()
	defer topo.Unlock()
	n, ok := topo.nodes[nodeId]

	if !ok {
		return nil, errors.New(nodeId + " ID doesn't exist")
	}

	if n.MachineId != topo.machineId {
		delete(topo.macs, n.NetworkNode.GetMac())
		delete(topo.nodes, nodeId)
		return n, nil
	}

	if n.Bridge != nil {
		b := n.Bridge
		n.RemoveBridge()
		b.RemoveNode(nodeId)
		b.NetworkBridge.RemoveNode([]byte(n.NetworkNode.GetMac()))
		delete(topo.links, n.Link.ID())
		topo.removeNodeFromRouters([]byte(n.NetworkNode.GetMac()))
	} else {
		n.Link.NetworkBILink.Left.Close()
	}

	topo.fl.RemoveMac([]byte(n.NetworkNode.GetMac()))
	delete(topo.macs, n.NetworkNode.GetMac())
	delete(topo.nodes, nodeId)
	return n, nil
}

func (topo *Topology) removeNodeFromRouters(mac []byte) {

	for _, router := range topo.routers {
		if router.MachineId == topo.machineId {
			router.NetworkRouter.RemoveNode(mac)
		}
	}
}

// TODO Add condition to verify if node has bridge associated, if not then do nothing
func (topo *Topology) RemoveBridge(bridgeId string) (*Bridge, error) {

	topo.Lock()
	defer topo.Unlock()
	b, ok := topo.bridges[bridgeId]

	if !ok {
		return nil, errors.New(bridgeId + " ID doesn't exist")
	}

	if b.MachineId != topo.machineId {
		delete(topo.bridges, bridgeId)
		return b, nil
	}

	b.NetworkBridge.Close()
	if b.NetworkBridge.Link() != nil {
		b.Router.RemoveBridge(b.ID())
		delete(topo.links, b.RouterLink.ID())
		b.RemoveRouter()
	}

	for _, node := range b.ConnectedNodes {
		node.RemoveBridge()
		delete(topo.links, node.Link.ID())
		topo.removeNodeFromRouters([]byte(node.NetworkNode.GetMac()))
		nullLink := network.CreateBILink(network.CreateNullLink(node.NetworkNode.GetOutgoing()), nil)
		node.NetworkNode.SetLink(nullLink)
		nullLink.GetLeft().Start()
	}

	delete(topo.bridges, bridgeId)

	return b, nil

}

// TODO configure cleanup channel for
func (topo *Topology) RemoveRouter(routerId string) (*Router, error) {

	topo.Lock()
	r, ok := topo.routers[routerId]

	if !ok {
		return nil, errors.New(routerId + " ID doesn't exist")
	}

	routers := make([]string, 0, len(r.ConnectedRouters))
	for router, _ := range r.ConnectedRouters {
		routers = append(routers, router)
	}
	topo.Unlock()
	for _, router := range routers {
		topo.DisconnectRouters(routerId, router)
	}

	topo.Lock()
	topo.Unlock()
	delete(topo.routers, routerId)
	if r.MachineId != topo.machineId {
		return r, nil
	}

	r.NetworkRouter.Close()
	for _, bridge := range r.ConnectedBridges {
		macs := bridge.NetworkBridge.GetMacs()
		for _, mac := range macs {

			for _, router := range topo.routers {
				router.NetworkRouter.RemoveNode(mac)
				delete(router.Weights, string(mac))
			}
		}
		bridge.RemoveRouter()
	}

	return r, nil
}

func (topo *Topology) DisconnectNode(id string) error {
	topo.Lock()
	defer topo.Unlock()
	n, ok := topo.GetNode(id)

	if !ok {
		return errors.New(id + " ID doesn't exist")
	}

	if n.Bridge == nil {
		return errors.New(id + "is not connected")
	}

	b := n.Bridge

	delete(topo.links, n.Link.ID())
	n.RemoveBridge()
	b.RemoveNode(n.ID())
	l := network.CreateBILink(network.CreateNullLink(n.NetworkNode.GetOutgoing()), nil)
	n.NetworkNode.SetLink(l)
	l.GetLeft().Start()

	if b.Router != nil {
		b.Router.NetworkRouter.RemoveNode([]byte(n.NetworkNode.GetMac()))
		//delete(b.Router.Weights, n.NetworkNode.GetMac())
		b.Router.RemoveWeight(n.NetworkNode.GetMac())
	}
	return nil
}

func (topo *Topology) DisconnectBridge(id string) error {

	topo.Lock()
	defer topo.Unlock()
	b, ok := topo.GetBridge(id)

	if !ok {
		return errors.New(id + " ID doesn't exist")
	}

	if b.Router == nil {
		return errors.New(id + "is not connected")
	}

	r := b.Router

	delete(topo.links, b.RouterLink.ID())

	b.RemoveRouter()
	r.RemoveBridge(b.ID())

	for _, node := range b.ConnectedNodes {
		r.NetworkRouter.RemoveNode([]byte(node.NetworkNode.GetMac()))
		r.RemoveWeight(node.NetworkNode.GetMac())
	}

	return nil
}

func (topo *Topology) DisconnectRouters(router1 string, router2 string) error {

	topo.Lock()
	defer topo.Unlock()
	r1, ok := topo.GetRouter(router1)

	if !ok {
		return errors.New(router1 + " ID doesn't exist")
	}
	r2, ok := topo.GetRouter(router2)

	if !ok {
		return errors.New(router2 + " ID doesn't exist")
	}

	if r1.MachineId == topo.machineId && r2.MachineId == topo.machineId {

		link, ok := r1.RouterLinks[router2]

		if !ok {
			return errors.New(router1 + " and " + router2 + " are not connected")
		}
		link.NetworkBILink.Close()
		delete(topo.links, link.ID())

		r1.RemoveRouter(router2)
		r2.RemoveRouter(router1)

		weights := make([]string, 0, len(r1.Weights))

		for mac, weight := range r1.Weights {
			if weight.Router == router2 {
				weights = append(weights, mac)
			}
		}
		r1.NetworkRouter.Pause()
		for _, mac := range weights {
			r1.RemoveWeight(mac)
			r1.NetworkRouter.RemoveNode([]byte(mac))
		}
		r1.NetworkRouter.Unpause()

		weights = make([]string, 0, len(r1.Weights))

		for mac, weight := range r2.Weights {
			if weight.Router == router1 {
				weights = append(weights, mac)
			}
		}
		r2.NetworkRouter.Pause()
		for _, mac := range weights {
			r2.RemoveWeight(mac)
			r2.NetworkRouter.RemoveNode([]byte(mac))
		}
		r2.NetworkRouter.Unpause()

	} else if r1.MachineId == topo.machineId && r2.MachineId != topo.machineId {
		link, ok := r1.RouterLinks[router2]

		if !ok {
			return errors.New(router1 + " and " + router2 + " are not connected")
		}

		delete(r1.RouterLinks, router2)
		delete(r1.ConnectedRouters, router2)

		delete(r2.RouterLinks, router1)
		delete(r2.ConnectedRouters, router1)

		link.ConnectsTo.NetworkLink.Close()

		weights := make([]string, 0, len(r1.Weights))

		for mac, weight := range r1.Weights {
			if weight.Router == router2 {
				weights = append(weights, mac)
			}
		}
		r1.NetworkRouter.Pause()
		for _, mac := range weights {
			r1.RemoveWeight(mac)
			r1.NetworkRouter.RemoveNode([]byte(mac))
		}
		r1.NetworkRouter.Unpause()

		weights = make([]string, 0, len(r2.Weights))

		for mac, weight := range r2.Weights {
			if weight.Router == router1 {
				weights = append(weights, mac)
			}
		}
		for _, mac := range weights {
			r2.RemoveWeight(mac)
			r2.NetworkRouter.RemoveNode([]byte(mac))
		}

	} else if r2.MachineId == topo.machineId && r1.MachineId != topo.machineId {
		link, ok := r2.RouterLinks[router1]

		if !ok {
			return errors.New(router1 + " and " + router2 + " are not connected")
		}

		delete(r1.RouterLinks, router2)
		delete(r1.ConnectedRouters, router2)

		delete(r2.RouterLinks, router1)
		delete(r2.ConnectedRouters, router1)

		link.ConnectsTo.NetworkLink.Close()

		weights := make([]string, 0, len(r1.Weights))

		for mac, weight := range r2.Weights {
			if weight.Router == router1 {
				weights = append(weights, mac)
			}
		}
		r2.NetworkRouter.Pause()
		for _, mac := range weights {
			r2.RemoveWeight(mac)
			r2.NetworkRouter.RemoveNode([]byte(mac))
		}
		r2.NetworkRouter.Unpause()

		weights = make([]string, 0, len(r1.Weights))

		for mac, weight := range r1.Weights {
			if weight.Router == router2 {
				weights = append(weights, mac)
			}
		}
		for _, mac := range weights {
			r1.RemoveWeight(mac)
			r1.NetworkRouter.RemoveNode([]byte(mac))
		}
	}
	return nil
}

func (topo *Topology) ForgetRoutes(routerId string) error {

	topo.Lock()
	defer topo.Unlock()
	r, ok := topo.GetRouter(routerId)

	if !ok {
		return errors.New(routerId + " ID doesn't exist")
	}

	r.NetworkRouter.Pause()
	r.NetworkRouter.ClearRoutes()

	r.Weights = make(map[string]Weight)

	for _, bridge := range r.ConnectedBridges {
		for _, node := range bridge.ConnectedNodes {
			r.NetworkRouter.AddNode([]byte(node.NetworkNode.GetMac()), GetOriginChanFromLink(r.ID(), bridge.RouterLink))
			r.AddWeight(node.NetworkNode.GetMac(), r.ID(), 0)
		}
	}
	r.NetworkRouter.Unpause()
	return nil
}

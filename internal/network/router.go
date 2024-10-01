package network

import (
	"fmt"
	"github.com/David-Antunes/gone-proxy/xdp"
	"github.com/David-Antunes/gone/internal/network/routing"
	"sync"
)

type Router struct {
	sync.Mutex
	id              string
	channels        map[string]chan *xdp.Frame
	incomingChannel chan *xdp.Frame
	running         bool
	queue           chan *xdp.Frame
	ctx             chan struct{}
}

func CreateRouter(id string) *Router {

	return &Router{
		Mutex:           sync.Mutex{},
		id:              id,
		channels:        make(map[string]chan *xdp.Frame),
		incomingChannel: make(chan *xdp.Frame, queueSize),
		running:         false,
		queue:           make(chan *xdp.Frame, queueSize),
		ctx:             make(chan struct{}),
	}
	//return &Router{id, make(map[string]chan *xdp.Frame), make(chan *xdp.Frame, queueSize), false, make(chan *xdp.Frame, queueSize), make(chan struct{})}
}

func (router *Router) Incoming() chan *xdp.Frame {
	return router.incomingChannel
}

func (router *Router) GetMacs() [][]byte {
	macs := make([][]byte, 0, len(router.channels))

	for key := range router.channels {
		macs = append(macs, []byte(key))
	}

	return macs
}

func (router *Router) AddNode(mac []byte, channel chan *xdp.Frame) {
	router.Lock()
	router.channels[string(mac)] = channel
	router.Unlock()
}

func (router *Router) RemoveNode(mac []byte) {
	router.Lock()
	if _, ok := router.channels[string(mac)]; ok {
		delete(router.channels, string(mac))
	}
	router.Unlock()
}

func (router *Router) HasMac(mac []byte) bool {
	_, ok := router.channels[string(mac)]
	return ok
}

func (router *Router) ClearRoutes() {
	router.Lock()
	router.channels = make(map[string]chan *xdp.Frame)
	router.Unlock()
}

func (router *Router) Start() {
	if !router.running {
		router.running = true
		go router.receive()
		go router.send()
	}
}

func (router *Router) Stop() {
	if router.running {
		router.running = false
		router.ctx <- struct{}{}
		router.ctx <- struct{}{}
	}
}

func (router *Router) receive() {
	for {

		select {
		case <-router.ctx:
			return

		case frame := <-router.incomingChannel:
			if len(router.queue) < queueSize {
				router.queue <- frame
			} else {
				fmt.Println(router.id, "Queue Full!")
			}
		}
	}
}

func (router *Router) send() {
	for {
		select {
		case <-router.ctx:
			return
		case frame := <-router.queue:
			if channel, ok := router.channels[frame.GetMacDestination()]; ok {
				channel <- frame
			} else {
				routing.HandleNewMac(frame, router.id)
			}
		}
	}
}

func (router *Router) InjectFrame(frame *xdp.Frame) {
	if channel, ok := router.channels[frame.GetMacDestination()]; ok {
		channel <- frame
	} else {
		routing.HandleNewMac(frame, router.id)
	}
}

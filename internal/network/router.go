package network

import (
	"fmt"
	"github.com/David-Antunes/gone-proxy/xdp"
	"github.com/David-Antunes/gone/internal/network/routing"
)

type Router struct {
	id              string
	channels        map[string]chan *xdp.Frame
	incomingChannel chan *xdp.Frame
	running         bool
	queue           chan *xdp.Frame
	ctx             chan struct{}
}

func CreateRouter(id string) *Router {

	return &Router{id, make(map[string]chan *xdp.Frame), make(chan *xdp.Frame, queueSize), false, make(chan *xdp.Frame, queueSize), make(chan struct{})}
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
	router.channels[string(mac)] = channel
}

func (router *Router) RemoveNode(mac []byte) {
	if _, ok := router.channels[string(mac)]; ok {
		delete(router.channels, string(mac))
	}
}

func (router *Router) HasMac(mac []byte) bool {
	_, ok := router.channels[string(mac)]
	return ok
}

func (router *Router) ClearRoutes() {
	router.channels = make(map[string]chan *xdp.Frame)
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
				fmt.Println("Queue Full!")
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
			channel, ok := router.channels[frame.GetMacDestination()]
			if !ok {
				routing.HandleNewMac(frame, router.id)
			} else {
				channel <- frame
			}
		}
	}
}

func (router *Router) InjectFrame(frame *xdp.Frame) {
	channel, ok := router.channels[frame.GetMacDestination()]
	if !ok {
		routing.HandleNewMac(frame, router.id)
	} else {
		channel <- frame
	}
}

package network

import (
	"fmt"
	"github.com/David-Antunes/gone-proxy/xdp"
	"github.com/David-Antunes/gone/internal"
	"github.com/David-Antunes/gone/internal/network/routing"
	"net"
	"sync"
	"time"
)

type Router struct {
	sync.RWMutex
	id              string
	channels        map[string]chan *xdp.Frame
	incomingChannel chan *xdp.Frame
	running         bool
	queue           chan *xdp.Frame
	ctx             chan struct{}
	disrupted       disruptLogic
}

func CreateRouter(id string) *Router {

	return &Router{
		RWMutex:         sync.RWMutex{},
		id:              id,
		channels:        make(map[string]chan *xdp.Frame),
		incomingChannel: make(chan *xdp.Frame, internal.ComponentQueueSize),
		running:         false,
		queue:           make(chan *xdp.Frame, internal.ComponentQueueSize),
		ctx:             make(chan struct{}, 2),
		disrupted: disruptLogic{
			disrupted: false,
			ctx:       make(chan struct{}, 1),
		},
	}
}

func (router *Router) Incoming() chan *xdp.Frame {
	return router.incomingChannel
}

func (router *Router) GetMacs() [][]byte {
	router.RLock()
	macs := make([][]byte, 0, len(router.channels))

	for key := range router.channels {
		macs = append(macs, []byte(key))
	}

	router.RUnlock()
	return macs
}

func (router *Router) AddNode(mac []byte, channel chan *xdp.Frame) {
	router.Lock()
	if len(mac) != 6 {
		fmt.Println(router.id, "AddNode: invalid mac")
	}
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
	router.RLock()
	_, ok := router.channels[string(mac)]
	router.RUnlock()
	return ok
}

func (router *Router) ClearRoutes() {
	router.Lock()
	router.channels = make(map[string]chan *xdp.Frame)
	fmt.Println("New Size:", len(router.channels))
	router.Unlock()
}

func (router *Router) null() {
	for {
		select {
		case <-router.disrupted.ctx:
			return
		case <-router.incomingChannel:
			continue
		}
	}
}

func (router *Router) receive() {
	for {

		select {
		case <-router.ctx:
			return

		case frame := <-router.incomingChannel:
			//if len(router.queue) < internal.ComponentQueueSize {
			router.queue <- frame
			//} else {
			//	fmt.Println(router.id, "Queue Full!")
			//}
		}
	}
}

func (router *Router) send() {
	for {
		select {
		case <-router.ctx:
			return
		case frame := <-router.queue:
			router.RLock()
			if channel, ok := router.channels[frame.GetMacDestination()]; ok {
				if len(channel) < internal.QueueSize {
					channel <- frame
				}
				router.RUnlock()
			} else {
				router.RUnlock()
				routing.HandleNewMac(frame, router.id)
			}
		}
	}
}

func (router *Router) InjectFrame(frame *xdp.Frame) {
	router.RLock()
	defer router.RUnlock()
	if channel, ok := router.channels[frame.GetMacDestination()]; ok && len(channel) < internal.QueueSize {
		channel <- frame
	} else {
		fmt.Println("Failed to inject Frame", router.id, net.HardwareAddr(frame.MacDestination))
	}
}

func (router *Router) RemoteInjectFrame(frame *xdp.Frame) {
	router.RLock()
	if channel, ok := router.channels[frame.GetMacDestination()]; ok {
		if len(channel) < internal.QueueSize {
			channel <- frame
		}
		router.RUnlock()
	} else {
		router.RUnlock()
		routing.HandleNewMac(frame, router.id)
	}
}

func (router *Router) Close() {
	router.Stop()
	router.StopDisrupt()
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

func (router *Router) Pause() {
	if router.running {
		router.ctx <- struct{}{}
		router.ctx <- struct{}{}
	} else if router.disrupted.disrupted {
		router.disrupted.ctx <- struct{}{}
	}
}

func (router *Router) Unpause() {
	if router.running {
		go router.receive()
		go router.send()
	} else if router.disrupted.disrupted {
		go router.null()
	}
}

func (router *Router) Disrupt() bool {
	if !router.disrupted.disrupted {
		router.disrupted.disrupted = true
		router.Stop()
		go router.null()

		// Clear queue for requests
		go func() {
			go router.send()
			time.Sleep(time.Second)
			router.ctx <- struct{}{}
		}()
		return true
	} else {
		return false
	}

}

func (router *Router) StopDisrupt() bool {
	if router.disrupted.disrupted {
		router.disrupted.disrupted = false
		router.disrupted.ctx <- struct{}{}
		router.Start()
		return true
	} else {
		return false
	}
}

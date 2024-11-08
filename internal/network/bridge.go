package network

import (
	"bytes"
	"fmt"
	"github.com/David-Antunes/gone-proxy/xdp"
	"sync"
	"time"
)

type Bridge struct {
	sync.RWMutex
	channels        map[string]chan *xdp.Frame
	incomingChannel chan *xdp.Frame
	gateway         chan *xdp.Frame
	link            *BiLink
	running         bool
	queue           chan *xdp.Frame
	ctx             chan struct{}
	disrupted       disruptLogic
}

func CreateBridge() *Bridge {
	return &Bridge{
		RWMutex:         sync.RWMutex{},
		channels:        make(map[string]chan *xdp.Frame),
		incomingChannel: make(chan *xdp.Frame, queueSize),
		gateway:         nil,
		link:            nil,
		running:         false,
		queue:           make(chan *xdp.Frame, queueSize),
		ctx:             make(chan struct{}, 2),
		disrupted: disruptLogic{
			disrupted: false,
			ctx:       make(chan struct{}, 1),
		},
	}
}

func (bridge *Bridge) Gateway() chan *xdp.Frame {
	return bridge.gateway
}

func (bridge *Bridge) Incoming() chan *xdp.Frame {
	return bridge.incomingChannel
}

func (bridge *Bridge) SetGateway(gateway chan *xdp.Frame) {
	bridge.gateway = gateway
}

func (bridge *Bridge) SetLink(biLink *BiLink) {
	bridge.link = biLink
}

func (bridge *Bridge) Link() *BiLink {
	return bridge.link
}

func (bridge *Bridge) GetMacs() [][]byte {
	bridge.RLock()
	macs := make([][]byte, 0, len(bridge.channels))

	for key := range bridge.channels {
		macs = append(macs, []byte(key))
	}

	bridge.RUnlock()
	return macs
}

func (bridge *Bridge) AddNode(mac []byte, channel chan *xdp.Frame) {
	bridge.Lock()
	bridge.channels[string(mac)] = channel
	bridge.Unlock()
}

func (bridge *Bridge) RemoveNode(mac []byte) {
	bridge.Lock()
	delete(bridge.channels, string(mac))
	bridge.Unlock()
}

func (bridge *Bridge) Start() {
	if !bridge.running {
		bridge.running = true
		go bridge.receive()
		go bridge.send()
	}
}

func (bridge *Bridge) Stop() {
	if bridge.running {
		bridge.running = false
		bridge.ctx <- struct{}{}
		bridge.ctx <- struct{}{}
	}
}

func (bridge *Bridge) receive() {
	for {

		select {
		case <-bridge.ctx:
			return

		case frame := <-bridge.incomingChannel:
			if len(bridge.queue) < queueSize {
				bridge.queue <- frame
			} else {
				fmt.Println("Queue Full!")
			}
		}
	}
}

func (bridge *Bridge) Disrupt() bool {
	if !bridge.disrupted.disrupted {
		bridge.disrupted.disrupted = true
		bridge.Stop()
		go bridge.null()

		// Clear queue for requests
		go func() {
			go bridge.send()
			time.Sleep(time.Second)
			bridge.ctx <- struct{}{}
		}()
		return true
	} else {
		return false
	}
}

func (bridge *Bridge) null() {
	for {
		select {
		case <-bridge.disrupted.ctx:
			return
		case <-bridge.incomingChannel:
			continue
		}
	}
}

func (bridge *Bridge) StopDisrupt() bool {
	if bridge.disrupted.disrupted {
		bridge.disrupted.disrupted = false
		bridge.disrupted.ctx <- struct{}{}
		bridge.Start()
		return true
	}
	return false
}

func (bridge *Bridge) send() {

	for {

		select {
		case <-bridge.ctx:
			return

		case frame := <-bridge.queue:
			if bytes.Equal([]byte(frame.MacDestination), broadcastAddr) {
				bridge.RLock()
				for _, channel := range bridge.channels {
					channel <- frame
				}
				bridge.RUnlock()
				continue
			}
			bridge.RLock()
			if channel, ok := bridge.channels[frame.GetMacDestination()]; ok {
				channel <- frame
			} else {
				bridge.gateway <- frame
			}
			bridge.RUnlock()
		}
	}
}

func (bridge *Bridge) Close() {
	bridge.Stop()
	bridge.StopDisrupt()
}

func (bridge *Bridge) Pause() {
	if bridge.running {
		bridge.ctx <- struct{}{}
		bridge.ctx <- struct{}{}
	} else if bridge.disrupted.disrupted {
		bridge.disrupted.ctx <- struct{}{}
	}
}

func (bridge *Bridge) Unpause() {
	if bridge.running {
		go bridge.receive()
		go bridge.send()
	} else if bridge.disrupted.disrupted {
		go bridge.null()
	}
}

package cluster

import (
	"encoding/gob"
	"fmt"
	"github.com/David-Antunes/gone/internal"
	"github.com/David-Antunes/gone/internal/network"
	"log"
	"net"
	"os"
	"sync"
)

var icmLog = log.New(os.Stdout, "REMOTE INFO: ", log.Ltime)

type InterCommunicationManager struct {
	sync.RWMutex
	conn          net.Listener
	connections   map[string]chan *network.RouterFrame
	remoteRouters map[string]chan *network.RouterFrame
	delays        map[string]*network.Delay
	inQueue       chan *network.RouterFrame
	outQueue      chan *network.RouterFrame
	routers       map[string]*network.Router
	ctx           chan struct{}
	running       bool
}

func (icm *InterCommunicationManager) GetoutQueue() chan *network.RouterFrame {
	return icm.outQueue
}

func CreateInterCommunicationManager() *InterCommunicationManager {
	return &InterCommunicationManager{
		RWMutex:       sync.RWMutex{},
		conn:          nil,
		connections:   make(map[string]chan *network.RouterFrame),
		remoteRouters: make(map[string]chan *network.RouterFrame),
		delays:        make(map[string]*network.Delay),
		inQueue:       make(chan *network.RouterFrame, internal.RemoteQueueSize),
		outQueue:      make(chan *network.RouterFrame, internal.RemoteQueueSize),
		routers:       make(map[string]*network.Router),
		ctx:           make(chan struct{}),
		running:       false,
	}
}
func (icm *InterCommunicationManager) SetConnection(conn net.Listener) {
	icm.conn = conn
}

func (icm *InterCommunicationManager) Start() {
	if !icm.running {
		if icm.conn == nil {
			panic("Router socket is not configured")
		}
		icm.running = true
		go icm.Accept()
		go icm.receiveFrames()
		go icm.send()
	}
}

func (icm *InterCommunicationManager) Accept() {

	for {
		conn, err := icm.conn.Accept()
		if err != nil {
			fmt.Println("Error:", err)
			continue
		}
		go icm.receive(conn)
	}
}

func (icm *InterCommunicationManager) Stop() {
	icm.running = false
	icm.ctx <- struct{}{}
	icm.ctx <- struct{}{}
}

func (icm *InterCommunicationManager) AddMachine(conn net.Conn, machineId string) {
	icm.Lock()
	defer icm.Unlock()
	if _, ok := icm.connections[machineId]; !ok {
		icm.connections[machineId] = make(chan *network.RouterFrame, internal.QueueSize)
		go sendMachine(conn, icm.connections[machineId])
	}
}

func (icm *InterCommunicationManager) AddConnection(remoteRouter string, delay *network.Delay, machineId string, localRouter string, router *network.Router) {
	icm.Lock()
	icmLog.Println("Adding connection to remote router", remoteRouter, "with delay", delay.Value, "from local router", localRouter, "in machine", machineId)
	icm.remoteRouters[remoteRouter] = icm.connections[machineId]
	icm.routers[localRouter] = router
	icm.delays[remoteRouter] = delay
	icm.Unlock()
}

func (icm *InterCommunicationManager) RemoveConnection(remoteRouter string, localRouter string) {
	icm.Lock()
	icmLog.Println("Removing connection to remote router", remoteRouter, "from local router", localRouter)
	delete(icm.remoteRouters, remoteRouter)
	delete(icm.routers, localRouter)
	delete(icm.delays, remoteRouter)
	icm.Unlock()
}

func (icm *InterCommunicationManager) receiveFrames() {
	for {
		select {
		case <-icm.ctx:
			return
		case frame := <-icm.inQueue:
			//icm.Lock()
			icm.RLock()
			if router, ok := icm.routers[frame.To]; ok {
				//frame.Frame.Time = time.Now()
				//frame.Frame.Time = time.Now().Add(-icm.delays[frame.From].Value)
				//frame.Frame.Time = frame.Frame.Time.Add(-icm.delays[frame.From].Value)
				router.InjectFrame(frame.Frame)
			} else {
				fmt.Println("No router for ", frame.To)
			}
			icm.RUnlock()
		}
	}
}

func sendMachine(conn net.Conn, channel chan *network.RouterFrame) {
	enc := gob.NewEncoder(conn)
	for {
		select {

		case frame := <-channel:
			if err := enc.Encode(frame); err != nil {
				fmt.Println("Error:", err)
			}
		}
	}
}

func (icm *InterCommunicationManager) receive(conn net.Conn) {
	defer conn.Close()
	dec := gob.NewDecoder(conn)
	for {
		var frame *network.RouterFrame
		err := dec.Decode(&frame)
		if err != nil {
			panic(err)
		}
		if len(icm.inQueue) < internal.RemoteQueueSize {
			icm.inQueue <- frame
		}
	}
}

func (icm *InterCommunicationManager) send() {
	for {
		select {
		case <-icm.ctx:
			return
		case frame := <-icm.outQueue:
			icm.RLock()
			if conn, ok := icm.remoteRouters[frame.To]; ok {
				conn <- frame
			} else {
				fmt.Println("No router for ", frame.To)
			}
			icm.RUnlock()
		}
	}
}

package cluster

import (
	"encoding/gob"
	"fmt"
	"github.com/David-Antunes/gone/internal/network"
	"log"
	"net"
	"os"
	"sync"
)

var icmLog = log.New(os.Stdout, "REMOTE INFO: ", log.Ltime)

type InterCommunicationManager struct {
	sync.Mutex
	conn        net.Listener
	connections map[string]*gob.Encoder
	delays      map[string]*network.Delay
	inQueue     chan *network.RouterFrame
	outQueue    chan *network.RouterFrame
	routers     map[string]*network.Router
	ctx         chan struct{}
	running     bool
}

func (icm *InterCommunicationManager) GetoutQueue() chan *network.RouterFrame {
	return icm.outQueue
}

func CreateInterCommunicationManager() *InterCommunicationManager {
	return &InterCommunicationManager{
		Mutex:       sync.Mutex{},
		conn:        nil,
		connections: make(map[string]*gob.Encoder),
		delays:      make(map[string]*network.Delay),
		inQueue:     make(chan *network.RouterFrame, queueSize),
		outQueue:    make(chan *network.RouterFrame, queueSize),
		routers:     make(map[string]*network.Router),
		ctx:         make(chan struct{}),
		running:     false,
	}
}
func (icm *InterCommunicationManager) SetConnection(conn net.Listener) {
	icm.Lock()
	icm.conn = conn
	icm.Unlock()
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

func (icm *InterCommunicationManager) AddConnection(remoteRouter string, delay *network.Delay, connection net.Conn, localRouter string, router *network.Router) {
	icm.Lock()
	icmLog.Println("Adding connection to remote router", remoteRouter, "with delay", delay.Value, "from local router", localRouter)
	icm.connections[remoteRouter] = gob.NewEncoder(connection)
	icm.routers[localRouter] = router
	icm.delays[remoteRouter] = delay
	icm.Unlock()
}

func (icm *InterCommunicationManager) RemoveConnection(remoteRouter string, localRouter string) {
	icm.Lock()
	icmLog.Println("Removing connection to remote router", remoteRouter, "from local router", localRouter)
	delete(icm.connections, remoteRouter)
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
			if router, ok := icm.routers[frame.To]; ok {
				//frame.Frame.Time = time.Now()
				//frame.Frame.Time = time.Now().Add(-icm.delays[frame.From].Value)
				router.InjectFrame(frame.Frame)
			} else {
			}

			//icm.Unlock()
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

		if len(icm.inQueue) < queueSize {
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
			icm.Lock()
			conn, ok := icm.connections[frame.To]
			if ok {
				err := conn.Encode(&frame)
				if err != nil {
					panic(err)
				}
			} else {
			}
			icm.Unlock()
		}
	}
}

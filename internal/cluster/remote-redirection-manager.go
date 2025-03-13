package cluster

import (
	"bytes"
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
	conn        net.PacketConn
	connections map[string]*net.UDPConn
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
		connections: make(map[string]*net.UDPConn),
		delays:      make(map[string]*network.Delay),
		inQueue:     make(chan *network.RouterFrame, queueSize),
		outQueue:    make(chan *network.RouterFrame, queueSize),
		routers:     make(map[string]*network.Router),
		ctx:         make(chan struct{}),
		running:     false,
	}
}
func (icm *InterCommunicationManager) SetConnection(conn net.PacketConn) {
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
		//go icm.Accept()
		go icm.receive()
		go icm.receiveFrames()
		go icm.send()
	}
}

func (icm *InterCommunicationManager) Stop() {
	icm.running = false
	icm.ctx <- struct{}{}
	icm.ctx <- struct{}{}
}

func (icm *InterCommunicationManager) AddConnection(remoteRouter string, delay *network.Delay, connection *net.UDPConn, localRouter string, router *network.Router) {
	icm.Lock()
	icmLog.Println("Adding connection to remote router", remoteRouter, "with delay", delay.Value, "from local router", localRouter)
	icm.connections[remoteRouter] = connection
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
				fmt.Println("Failed to inject frame to router", frame.To)
			}

			//icm.Unlock()
		}
	}
}

func (icm *InterCommunicationManager) receive() {

	defer icm.conn.Close()

	for {
		buffer := make([]byte, 2048)
		n, _, err := icm.conn.ReadFrom(buffer)
		if err != nil {
			panic(err)
		}
		dec := gob.NewDecoder(bytes.NewReader(buffer[:n]))
		var frame *network.RouterFrame
		err = dec.Decode(&frame)
		if err != nil {
			panic(err)
		}
		//fmt.Println("Received from", addr, "Bytes:", n, frame.To)

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
			var response bytes.Buffer
			enc := gob.NewEncoder(&response)
			if enc.Encode(&frame) != nil {
				log.Fatal("encode frame failed", frame)
			} else {
				icm.Lock()
				if conn, ok := icm.connections[frame.To]; ok {
					_, err := conn.Write(response.Bytes())
					//fmt.Println("Sent from iphopn", conn.RemoteAddr(), len(response.Bytes()))
					if err != nil {
						panic(err)
					}
				}
				icm.Unlock()
			}

		}
	}
}

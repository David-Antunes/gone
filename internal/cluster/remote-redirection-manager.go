package cluster

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/David-Antunes/gone-proxy/xdp"
	"github.com/David-Antunes/gone/internal/network"
	"log"
	"net"
	"os"
	"sync"
	"time"
)

var icmLog = log.New(os.Stdout, "REMOTE INFO: ", log.Ltime)

type InterCommunicationManager struct {
	sync.Mutex
	conn        *net.UDPConn
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
func (icm *InterCommunicationManager) SetConnection(conn *net.UDPConn) {
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
		if _, _, err := icm.conn.ReadFrom(buffer); err != nil {
			panic(err)
		}
		if frame, err := decodeMessage(buffer); err != nil {
			panic(err)
		} else {
			if len(icm.inQueue) < queueSize {
				icm.inQueue <- frame
			}
		}

	}
}

func (icm *InterCommunicationManager) send() {

	for {
		select {
		case <-icm.ctx:
			return
		case frame := <-icm.outQueue:

			if buf, err := encodeMessage(frame); err != nil {
				log.Fatal("encode frame failed", frame)
			} else {
				icm.Lock()
				if conn, ok := icm.connections[frame.To]; ok {
					if _, err := conn.Write(buf); err != nil {
						panic(err)
					}
				}
				icm.Unlock()
			}

		}
	}
}

func encodeMessage(frame *network.RouterFrame) ([]byte, error) {

	buf := bytes.NewBuffer(make([]byte, 0, 2048))

	if err := binary.Write(buf, binary.LittleEndian, uint32(len(frame.To))); err != nil {
		return nil, err
	}
	if _, err := buf.Write([]byte(frame.To)); err != nil {
		return nil, err
	}

	if err := binary.Write(buf, binary.LittleEndian, uint32(len(frame.From))); err != nil {
		return nil, err
	}

	if _, err := buf.Write([]byte(frame.From)); err != nil {
		return nil, err
	}

	if _, err := buf.Write([]byte(frame.Frame.MacDestination)); err != nil {
		return nil, err
	}

	if _, err := buf.Write([]byte(frame.Frame.MacOrigin)); err != nil {
		return nil, err
	}

	if err := binary.Write(buf, binary.LittleEndian, frame.Frame.Time.Unix()); err != nil {
		return nil, err
	}

	if err := binary.Write(buf, binary.LittleEndian, int32(frame.Frame.Time.Nanosecond())); err != nil {
		return nil, err
	}
	if err := binary.Write(buf, binary.LittleEndian, uint32(frame.Frame.FrameSize)); err != nil {
		return nil, err
	}
	if n, err := buf.Write(frame.Frame.FramePointer); err != nil || n != len(frame.Frame.FramePointer) {
		return nil, err
	}
	return buf.Bytes(), nil
}

func decodeMessage(data []byte) (*network.RouterFrame, error) {

	buf := bytes.NewBuffer(data)

	var toSize uint32
	var fromSize uint32
	macOrigin := make([]byte, 6)
	macDestination := make([]byte, 6)
	var frameSize uint32
	var t int64
	var tnano int32

	if err := binary.Read(buf, binary.LittleEndian, &toSize); err != nil {
		return nil, err
	}
	to := make([]byte, toSize)
	if _, err := buf.Read(to); err != nil {
		return nil, err
	}

	if err := binary.Read(buf, binary.LittleEndian, &fromSize); err != nil {
		return nil, err
	}

	from := make([]byte, fromSize)
	if err := binary.Read(buf, binary.LittleEndian, &from); err != nil {
		return nil, err
	}

	if _, err := buf.Read(macDestination); err != nil {
		return nil, err
	}

	if _, err := buf.Read(macOrigin); err != nil {
		return nil, err
	}

	if err := binary.Read(buf, binary.LittleEndian, &t); err != nil {
		return nil, err
	}

	if err := binary.Read(buf, binary.LittleEndian, &tnano); err != nil {
		return nil, err
	}

	if err := binary.Read(buf, binary.LittleEndian, &frameSize); err != nil {
		return nil, err
	}

	framePointer := make([]byte, frameSize)

	if n, err := buf.Read(framePointer); err != nil || n != len(framePointer) {
		return nil, err
	}
	return &network.RouterFrame{
		To:   string(to),
		From: string(from),
		Frame: &xdp.Frame{
			FramePointer:   framePointer,
			FrameSize:      int(frameSize),
			Time:           time.Unix(t, int64(tnano)),
			MacOrigin:      string(macOrigin),
			MacDestination: string(macDestination),
		},
	}, nil
}

package proxy

import (
	"encoding/gob"
	"fmt"
	"github.com/David-Antunes/gone-proxy/xdp"
	"log"
	"net"
	"os"
	"sync"
)

var proxyLog = log.New(os.Stdout, "PROXY INFO: ", log.Ltime)

type Proxy struct {
	sync.Mutex
	incoming    map[string]chan *xdp.Frame
	outgoing    map[string]chan *xdp.Frame
	connections map[string]net.Conn
	running     bool
}

func CreateProxy() *Proxy {
	return &Proxy{
		Mutex:       sync.Mutex{},
		incoming:    make(map[string]chan *xdp.Frame),
		outgoing:    make(map[string]chan *xdp.Frame),
		connections: make(map[string]net.Conn),
		running:     false,
	}
}

func (p *Proxy) GetIncomingChannel(id string) (chan *xdp.Frame, bool) {
	channel, ok := p.incoming[id]
	return channel, ok
}
func (p *Proxy) GetOutgoingChannel(id string) (chan *xdp.Frame, bool) {
	channel, ok := p.outgoing[id]
	return channel, ok
}

func (p *Proxy) AddMac(mac []byte, incoming chan *xdp.Frame, outgoing chan *xdp.Frame) {
	p.Lock()

	conn, err := net.Dial("unix", "/tmp/"+string(mac)+".sock")
	if err != nil {
		panic(err)
	}
	enc := gob.NewEncoder(conn)
	dec := gob.NewDecoder(conn)
	p.incoming[string(mac)] = incoming
	p.outgoing[string(mac)] = outgoing
	p.Unlock()

	go receive(string(mac), dec, incoming)
	go send(string(mac), enc, outgoing)
}

func (p *Proxy) RemoveMac(mac []byte) {
	p.Lock()
	delete(p.incoming, string(mac))
	delete(p.outgoing, string(mac))
	delete(p.connections, string(mac))
	p.Unlock()
}

func receive(mac string, dec *gob.Decoder, incoming chan *xdp.Frame) {

	for {
		var frame *xdp.Frame

		err := dec.Decode(&frame)
		if err != nil {
			proxyLog.Println(mac+":", err)
			return
		}
		if len(incoming) < queueSize {
			//frame.Time = frame.Time.Add(-time.Now().Sub(frame.Time))
			incoming <- frame
		} else {
			fmt.Println("Proxy queue full")
		}
	}
}

func send(mac string, enc *gob.Encoder, outgoing chan *xdp.Frame) {
	for {
		select {
		case frame := <-outgoing:
			err := enc.Encode(&frame)
			if err != nil {
				proxyLog.Println(mac+":", err)
				return
			}
		}
	}
}

func (p *Proxy) Close() {

	for _, conn := range p.connections {
		conn.Close()
	}
}

package redirect_traffic

import (
	"encoding/gob"
	"fmt"
	"github.com/David-Antunes/gone-proxy/xdp"
	"net"
	"os"
)

const (
	queueSize = 1000
)

type RedirectionSocket struct {
	id         string
	socketPath string
	sock       net.Listener
	incoming   chan *xdp.Frame
	outgoing   chan *xdp.Frame
}

func (rs *RedirectionSocket) Id() string {
	return rs.id
}

func (rs *RedirectionSocket) GetIncoming() chan *xdp.Frame {
	return rs.incoming
}

func (rs *RedirectionSocket) GetOutgoing() chan *xdp.Frame {
	return rs.outgoing
}
func (rs *RedirectionSocket) GetSocketPath() string {
	return rs.socketPath
}

func NewRedirectionSocket(id string, socketPath string) (*RedirectionSocket, error) {
	os.Remove(socketPath)
	socket, err := net.Listen("unix", socketPath)
	if err != nil {
		return nil, err
	}
	return &RedirectionSocket{
		id:         id,
		socketPath: socketPath,
		sock:       socket,
		incoming:   make(chan *xdp.Frame, queueSize),
		outgoing:   make(chan *xdp.Frame, queueSize),
	}, nil
}

func (rs *RedirectionSocket) Start() {

	for {
		conn, err := rs.sock.Accept()
		if err != nil {
			fmt.Println(err)
			return
		}
		go rs.redirect(conn)
		dec := gob.NewDecoder(conn)
		for {
			var frame *xdp.Frame

			err = dec.Decode(&frame)
			if err != nil {
				fmt.Println(err)
				break
			}
			rs.incoming <- frame
		}
	}

}

func (rs *RedirectionSocket) Stop() {
	err := rs.sock.Close()
	if err != nil {
		return
	}
	err = os.Remove(rs.socketPath)
	if err != nil {
		return
	}
}

func (rs *RedirectionSocket) redirect(conn net.Conn) {
	enc := gob.NewEncoder(conn)
	for {
		select {
		case frame := <-rs.outgoing:
			err := enc.Encode(frame)
			if err != nil {
				fmt.Println(err)
				return
			}
		}
	}
}

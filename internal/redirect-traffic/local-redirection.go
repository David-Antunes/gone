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
	incoming   chan *xdp.Frame // Incoming packets from outside
	outgoing   chan *xdp.Frame // Outgoing packets to outside
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
	os.Chmod(socketPath, 0777)
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

		// Ignore packets until someone makes a connection
		c := make(chan struct{})
		go func() {
			for {
				select {
				case <-c:
					return
				case <-rs.outgoing:
					continue
				}
			}
		}()

		// Await connection
		conn, err := rs.sock.Accept()
		if err != nil {
			fmt.Println(err)
			return
		}
		c <- struct{}{}

		go rs.redirect(conn)
		dec := gob.NewDecoder(conn)

		// Handle packet receiving from connection
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

	// Manage sending out packets
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

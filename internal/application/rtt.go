package application

import (
	"encoding/gob"
	"github.com/David-Antunes/gone-proxy/api"
	"github.com/David-Antunes/gone/internal/network"
	"net"
	"time"
)

type RttManager struct {
	conn    net.Conn
	rtt     *network.DynamicDelay
	timeout time.Duration
}

func NewRttManager(conn net.Conn, timeout time.Duration) *RttManager {
	return &RttManager{
		conn: conn,
		rtt: &network.DynamicDelay{
			ReceiveDelay:  &network.Delay{Value: time.Duration(0)},
			TransmitDelay: &network.Delay{Value: time.Duration(0)},
		},
		timeout: timeout,
	}
}

func (rm *RttManager) Start() {
	dec := gob.NewDecoder(rm.conn)
	for {
		var rtt *api.UpdateRTTRequest

		err := dec.Decode(&rtt)
		if err != nil {
			panic(err)
		}
		rm.rtt.ReceiveDelay.Value = rtt.ReceiveLatency
		rm.rtt.TransmitDelay.Value = rtt.TransmitLatency
		time.Sleep(rm.timeout)
	}
}
func (rm *RttManager) Stop() {
	rm.conn.Close()
}

func (rm *RttManager) GetRtt() *network.DynamicDelay {
	return rm.rtt
}

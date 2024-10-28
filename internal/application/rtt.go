package application

import (
	"encoding/gob"
	"fmt"
	"github.com/David-Antunes/gone-proxy/api"
	"github.com/David-Antunes/gone/internal/network"
	"log"
	"math"
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
		rm.rtt.ReceiveDelay.Value = time.Duration(math.Min(float64(rm.rtt.ReceiveDelay.Value), float64(rm.rtt.TransmitDelay.Value)))
		rm.rtt.TransmitDelay.Value = time.Duration(math.Min(float64(rm.rtt.ReceiveDelay.Value), float64(rm.rtt.TransmitDelay.Value)))
		fmt.Println("RTT INFO PROXY:", log.Ltime, "Receive:", rm.rtt.ReceiveDelay.Value, "Transmit:", rm.rtt.TransmitDelay.Value)
		if rm.timeout > 0 {
			time.Sleep(rm.timeout)
		} else {
			return
		}
	}
}
func (rm *RttManager) Stop() {
	rm.conn.Close()
}

func (rm *RttManager) GetRtt() *network.DynamicDelay {
	return rm.rtt
}

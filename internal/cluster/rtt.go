package cluster

import (
	"errors"
	"fmt"
	"github.com/David-Antunes/gone/internal/network"
	"log"
	"net/http"
	"time"
)

type ClusterRTTManager struct {
	endpoints map[string]*nodeRTT
	numObs    int
	timeout   time.Duration
}

type nodeRTT struct {
	id     string
	ipAddr string
	delay  *network.Delay
}

func NewClusterRTTManager(numObs int, timeout time.Duration) *ClusterRTTManager {
	return &ClusterRTTManager{
		endpoints: make(map[string]*nodeRTT),
		numObs:    numObs,
		timeout:   timeout,
	}
}

func (rtt *ClusterRTTManager) GetDelay(id string) (*network.Delay, error) {
	if node, ok := rtt.endpoints[id]; !ok {
		return nil, errors.New("node not found")
	} else {
		return node.delay, nil
	}
}

func (rtt *ClusterRTTManager) AddNode(id string, ip string) {
	delay := &network.Delay{
		Value: 0,
	}
	if rtt.endpoints[id] == nil {
		rtt.endpoints[id] = &nodeRTT{
			id:     id,
			ipAddr: ip,
			delay:  delay,
		}
	} else {
		return
	}

	go func() {
		time.Sleep(1 * time.Second)
		for {
			var obs time.Duration

			for i := 0; i < rtt.numObs; i++ {
				start := time.Now()
				_, err := http.Get("http://" + ip + "/ping")
				if err != nil {
					return
				}
				end := time.Now()
				obs += end.Sub(start)
			}

			delay.Value = (obs / time.Duration(rtt.numObs)) / 2
			fmt.Println("RTT INFO REMOTE:", log.Ltime, "Delay of", delay.Value, ip)

			if rtt.timeout >= 0 {
				time.Sleep(rtt.timeout)
			} else {
				return
			}
		}
	}()
}

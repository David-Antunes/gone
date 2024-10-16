package cluster

import (
	"errors"
	"fmt"
	"github.com/David-Antunes/gone/internal/network"
	"net/http"
	"time"
)

type ClusterRTTManager struct {
	endpoints map[string]*nodeRTT
}

type nodeRTT struct {
	id     string
	ipAddr string
	delay  *network.Delay
}

func NewClusterRTTManager() *ClusterRTTManager {
	return &ClusterRTTManager{
		endpoints: make(map[string]*nodeRTT),
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
			fmt.Println("Started rtt logic for", ip)
			var obs time.Duration

			for i := 0; i < 25; i++ {
				start := time.Now()
				_, err := http.Get("http://" + ip + "/ping")
				if err != nil {
					return
				}
				end := time.Now()
				obs += end.Sub(start)
			}

			delay.Value = obs / 25
			fmt.Println("Registered delay of", delay.Value, ip)
			time.Sleep(time.Minute)
		}
	}()
}

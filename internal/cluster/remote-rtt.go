package cluster

import (
	"errors"
	"fmt"
	"github.com/David-Antunes/gone/internal/network"
	"math"
	"net/http"
	"time"
)

type RemoteRTTManager struct {
	endpoints map[string]*nodeRTT
	numObs    int
	timeout   time.Duration
}

type nodeRTT struct {
	id     string
	ipAddr string
	delay  *network.Delay
}

func NewClusterRTTManager(numObs int, timeout time.Duration) *RemoteRTTManager {
	return &RemoteRTTManager{
		endpoints: make(map[string]*nodeRTT),
		numObs:    numObs,
		timeout:   timeout,
	}
}

func (rtt *RemoteRTTManager) GetDelay(id string) (*network.Delay, error) {
	if node, ok := rtt.endpoints[id]; !ok {
		return nil, errors.New("node not found")
	} else {
		return node.delay, nil
	}
}

func (rtt *RemoteRTTManager) AddNode(id string, ip string) {
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
			d := time.Duration(math.MaxInt64)
			for i := 0; i < rtt.numObs; i++ {
				start := time.Now()
				_, err := http.Get("http://" + ip + "/ping")
				if err != nil {
					return
				}
				end := time.Now()
				obs := end.Sub(start)
				d = time.Duration(math.Min(float64(obs), float64(d)))
			}

			delay.Value = d / 2.0
			fmt.Println("RTT INFO REMOTE: Delay of", delay.Value, ip)

			if rtt.timeout > 0 {
				time.Sleep(rtt.timeout)
			} else {
				return
			}
		}
	}()
}

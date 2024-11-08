package cluster

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/David-Antunes/gone/internal/network"
	"net"
	"net/http"
	"time"
)

type Cluster struct {
	Primary   string
	Nodes     map[string]ClusterNode
	Endpoints map[string]net.Conn
	Rtt       *RemoteRTTManager
}

type ClusterNode struct {
	Hostname string `json:"hostname"`
	IpAddr   string `json:"ipAddr"`
	UdpAddr  string `json:"udpAddr"`
}

type ClusterNodeRequest struct {
	Hostname string `json:"hostname"`
	IpAddr   string `json:"ipAddr"`
	UdpAddr  string `json:"udpAddr"`
}

type ClusterNodeResponse struct {
	Hostname string                 `json:"hostname"`
	IpAddr   string                 `json:"ipAddr"`
	UdpAddr  string                 `json:"udpAddr"`
	Nodes    map[string]ClusterNode `json:"nodes"`
}

func CreateCluster(primary string, numObs int, timeout time.Duration) *Cluster {
	return &Cluster{
		Primary:   primary,
		Nodes:     make(map[string]ClusterNode),
		Endpoints: make(map[string]net.Conn),
		Rtt:       NewClusterRTTManager(numObs, timeout),
	}
}

func (cl *Cluster) Contains(machineId string) bool {
	if cl.Primary == machineId {
		return true
	}
	if _, ok := cl.Nodes[machineId]; ok {
		return ok
	}
	return false
}

func (cl *Cluster) SendMsg(machineId string, body any, endpoint string) (*http.Response, error) {
	if !cl.Contains(machineId) {
		return nil, errors.New("machine id not found")
	}

	msgBody, err := json.Marshal(body)

	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPost, "http://"+cl.Nodes[machineId].IpAddr+"/"+endpoint, bytes.NewReader(msgBody))

	if err != nil {
		return nil, err
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	return res, nil
}

// TODO return node replies
func (cl *Cluster) Broadcast(body any, httpMethod string, endpoint string) ([]*http.Response, error) {
	responses := make([]*http.Response, 0, len(cl.Nodes))
	for _, node := range cl.Nodes {
		var msgBody []byte
		var err error
		if body != nil {
			msgBody, err = json.Marshal(body)
		}

		if err != nil {
			return nil, err
		}
		var req *http.Request
		if httpMethod == http.MethodPost {
			req, err = http.NewRequest(http.MethodPost, "http://"+node.IpAddr+"/"+endpoint, bytes.NewReader(msgBody))
		} else if httpMethod == http.MethodGet {
			req, err = http.NewRequest(http.MethodGet, "http://"+node.IpAddr+"/"+endpoint, nil)
		}

		if err != nil {
			continue
		}
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			continue
		}
		responses = append(responses, res)

	}
	return responses, nil
}

func (cl *Cluster) RegisterNode(node string, ipAddr string, udpAddr string) error {
	if _, ok := cl.Nodes[node]; !ok {
		cl.Nodes[node] = ClusterNode{
			Hostname: node,
			IpAddr:   ipAddr,
			UdpAddr:  udpAddr,
		}
		clusterLog.Println("Dialing", udpAddr)
		conn, err := net.Dial("tcp", udpAddr)
		if err != nil {
			return errors.New("failed to connect to node " + node)
		}
		cl.Endpoints[node] = conn
		cl.Rtt.AddNode(node, ipAddr)
	} else {
		return errors.New("Node already registered")
	}
	return nil
}

func (cl *Cluster) GetNodeDelay(id string) (*network.Delay, error) {
	if node, ok := cl.Nodes[id]; !ok {
		return nil, errors.New("node not found")
	} else {
		return cl.Rtt.GetDelay(node.Hostname)
	}
}

func (cl *Cluster) JoinMembership(serverIp string, hostname string, ipAddr string, udpAddr string) {

	clusterLog.Println("Joining membership", serverIp, hostname, ipAddr, udpAddr)
	for {
		time.Sleep(1 * time.Second)

		body, err := json.Marshal(&ClusterNodeRequest{
			Hostname: hostname,
			IpAddr:   ipAddr,
			UdpAddr:  udpAddr,
		})

		if err != nil {
			panic(err)
		}

		req, err := http.NewRequest(http.MethodPost, "http://"+serverIp+"/registerMachine", bytes.NewReader(body))

		if err != nil {
			clusterLog.Println(err)
			continue
		}
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			clusterLog.Println(err)
			continue
		}
		clusterLog.Println(res.Status)
		if res.Status != "200 OK" {
			panic("failed to join cluster")
		}
		if err != nil {
			clusterLog.Println(err)
			continue
		}

		d := json.NewDecoder(res.Body)
		response := &ClusterNodeResponse{}
		err = d.Decode(&response)

		if err == nil {
			err = cl.RegisterNode(response.Hostname, response.IpAddr, response.UdpAddr)
			if err != nil {
				return
			}
			clusterLog.Println("Joined", response.Hostname, response.IpAddr, response.UdpAddr)
			for _, node := range response.Nodes {
				if node.Hostname == cl.Primary {
					continue
				}
				err = cl.RegisterNode(node.Hostname, node.IpAddr, node.UdpAddr)
				if err != nil {
					clusterLog.Println(err)
				}
				clusterLog.Println("Added", node.Hostname, node.IpAddr, node.UdpAddr)
			}
			return
		} else {
			clusterLog.Println(err)
			continue
		}
	}
}

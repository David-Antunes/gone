package cluster

import (
	"encoding/json"
	"github.com/David-Antunes/gone/internal/daemon"
	"log"
	"net/http"
	"os"
)

var clusterLog = log.New(os.Stdout, "Cluster INFO: ", log.Ltime)

type ClusterDaemon struct {
	Cl      *Cluster
	ipAddr  string
	udpAddr string
}

func (cd *ClusterDaemon) GetIpAddr() string {
	return cd.ipAddr
}
func (cd *ClusterDaemon) GetUdpAddr() string {
	return cd.udpAddr
}
func CreateClusterDaemon(cl *Cluster, ipAddr string, udpAddr string) *ClusterDaemon {
	return &ClusterDaemon{
		Cl:      cl,
		ipAddr:  ipAddr,
		udpAddr: udpAddr,
	}
}

func (cd *ClusterDaemon) RegisterMachine(w http.ResponseWriter, r *http.Request) {

	d := json.NewDecoder(r.Body)
	req := &ClusterNodeRequest{}
	err := d.Decode(&req)

	if err != nil {
		http.Error(w, "failed to register machine", http.StatusBadRequest)
		clusterLog.Println("failed to register machine")
		return
	}
	err = cd.Cl.RegisterNode(req.Hostname, req.IpAddr, req.UdpAddr)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		clusterLog.Println(err.Error())
		return
	}
	for _, node := range cd.Cl.Nodes {
		if node.Hostname == req.Hostname {
			continue
		}
		_, err := cd.Cl.SendMsg(node.Hostname, req, "registerClusterNode")
		if err != nil {
			panic(err.Error())
		}
	}

	resp := &ClusterNodeResponse{
		Hostname: cd.Cl.Primary,
		IpAddr:   cd.ipAddr,
		UdpAddr:  cd.udpAddr,
		Nodes:    cd.Cl.Nodes,
	}
	daemon.SendResponse(w, resp)

	clusterLog.Println("Added", req.Hostname, req.IpAddr, req.UdpAddr)
}
func (cd *ClusterDaemon) RegisterClusterNode(w http.ResponseWriter, r *http.Request) {

	d := json.NewDecoder(r.Body)
	req := &ClusterNodeRequest{}
	err := d.Decode(&req)

	if err != nil {
		http.Error(w, "failed to register machine", http.StatusBadRequest)
		clusterLog.Println("failed to register machine")
		return
	}
	err = cd.Cl.RegisterNode(req.Hostname, req.IpAddr, req.UdpAddr)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		clusterLog.Println(err.Error())
		return
	}
	resp := &ClusterNodeResponse{
		Hostname: cd.Cl.Primary,
		IpAddr:   cd.ipAddr,
		UdpAddr:  cd.udpAddr,
	}
	daemon.SendResponse(w, resp)

	clusterLog.Println("Added", req.Hostname, req.IpAddr, req.UdpAddr)
}

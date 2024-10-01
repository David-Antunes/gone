package main

import (
	"context"
	"fmt"
	"github.com/David-Antunes/gone/internal/application"
	"github.com/David-Antunes/gone/internal/cluster"
	"github.com/David-Antunes/gone/internal/docker"
	"github.com/David-Antunes/gone/internal/follower"
	"github.com/David-Antunes/gone/internal/graphDB"
	"github.com/David-Antunes/gone/internal/leader"
	"github.com/David-Antunes/gone/internal/network/routing"
	"github.com/David-Antunes/gone/internal/proxy"
	"github.com/spf13/viper"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"runtime/pprof"
	"sort"
	"syscall"
	"time"
)

var service = "localhost"

var unixUrl = "http+unix://" + service
var emulationLog = log.New(os.Stdout, "EMULATION INFO: ", log.Ltime)

func getClient() *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", "/tmp/proxy-server.sock")
			},
		},
	}
}

func cleanup() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		os.Exit(0)
	}()
}
func setEnvVariables() {
	cleanup()
	viper.SetConfigFile(".env")
	if err := viper.ReadInConfig(); err != nil {
		emulationLog.Println(err)
	}
	viper.SetDefault("GRAPHDB", "localhost")
	viper.SetDefault("GRAPHDB_USER", "")
	viper.SetDefault("GRAPHDB_PASSWORD", "")
	viper.SetDefault("ID", "primary")
	viper.SetDefault("PRIMARY", 0)
	viper.SetDefault("PRIMARY_SERVER_IP", "192.168.1.1")
	viper.SetDefault("PRIMARY_SERVER_PORT", "3000")
	viper.SetDefault("PRIMARY_ROUTE_PORT", "3001")
	viper.SetDefault("SERVER_IP", "192.168.1.1")
	viper.SetDefault("SERVER_PORT", "3000")
	viper.SetDefault("SERVER_ROUTE_PORT", "3001")
	viper.SetDefault("PROXY_RTT_SOCKET", "/tmp/proxy-rtt.sock")
	viper.SetDefault("PROXY_SERVER", "/tmp/proxy-server.sock")
	viper.SetDefault("PROXY_RTT_UPDATE_MS", 60000)
	viper.SetDefault("NETWORK_NAMESPACE", "gone_net")
	viper.SetConfigType("env")
	err := viper.WriteConfigAs(".env")
	if err != nil {
		emulationLog.Println(err)
	}
}

func printVariables() {
	settings := viper.AllSettings()
	sortedList := make([]string, 0, len(settings))
	for id, _ := range settings {
		sortedList = append(sortedList, id)
	}

	sort.Strings(sortedList)

	for _, id := range sortedList {
		emulationLog.Println(id, settings[id])
	}
}

func main() {

	// Read environment variables from .env
	setEnvVariables()
	viper.AutomaticEnv()
	printVariables()

	rttSocket, err := net.Dial("unix", viper.GetString("PROXY_RTT_SOCKET"))
	if err != nil {
		fmt.Println(err)
		panic("Proxy not running!")
	}

	proxyClient := getClient()

	proxyServer := proxy.NewProxyServer(proxyClient, unixUrl)

	graphDB.StartConnection("neo4j://"+viper.GetString("GRAPHDB"), viper.GetString("GRAPHDB_USER"), viper.GetString("GRAPHDB_PASSWORD"))

	// Configure emulation according if it is a local deployment or not
	p := viper.GetInt("PRIMARY")
	id := viper.GetString("ID")

	primaryAddr := viper.GetString("PRIMARY_SERVER_IP")
	primaryPort := viper.GetString("PRIMARY_SERVER_PORT")
	serverIP := viper.GetString("SERVER_IP")
	serverPort := viper.GetString("SERVER_PORT")
	framePort := viper.GetString("SERVER_ROUTE_PORT")
	var cl *cluster.Cluster
	prox := proxy.CreateProxy()
	rtt := application.NewRttManager(rttSocket, time.Duration(viper.GetInt("PROXY_RTT_UPDATE_MS"))*time.Millisecond)
	go rtt.Start()
	// Setup channel to send to proxy
	cl = cluster.CreateCluster(id)
	cd := cluster.CreateClusterDaemon(cl, serverIP+":"+serverPort, serverIP+":"+framePort)

	r, err := net.Listen("tcp", ":"+framePort)
	if err != nil {
		fmt.Println(err)
		panic("Proxy not running!")
	}
	icm := cluster.CreateInterCommunicationManager()
	icm.SetConnection(r)
	icm.Start()

	d := docker.CreateDockerManager(id, proxyServer, viper.GetString("NETWORK_NAMESPACE"))

	// Setup channel to send to proxy
	if p != 0 {
		app := application.NewLeader(cl, d, prox, icm, rtt)
		routing.SetRouting(app)

		leader.StartDaemon(app, cd, "0.0.0.0:"+serverPort)
		leader.Serve()
	} else {
		cl.JoinMembership(primaryAddr+":"+primaryPort, id, serverIP+":"+serverPort, serverIP+":"+framePort)
		fmt.Println("Contacted Main Node")

		app := application.NewFollower(cl, d, prox, icm, rtt)
		routing.SetRouting(app)

		follower.StartDaemon(app, cd, "0.0.0.0:"+serverPort)
		follower.Serve()
	}
}

package follower

import (
	"fmt"
	"github.com/David-Antunes/gone/internal/application"
	"github.com/David-Antunes/gone/internal/cluster"
	"log"
	"net"
	"net/http"
)

type server struct {
	httpServer *http.Server
	socket     net.Listener
	app        *application.Follower
}

var engine *server

func StartDaemon(app *application.Follower, cd *cluster.ClusterDaemon, ipAddr string) {
	if engine == nil {
		engine = createDaemon(app, cd, ipAddr)
	}
}

func createDaemon(app *application.Follower, cd *cluster.ClusterDaemon, ipAddr string) *server {
	socket, err := net.Listen("tcp", ipAddr)

	if err != nil {
		panic(err)
	}

	m := http.NewServeMux()
	m.HandleFunc("/ping", ping)

	m.HandleFunc("/registerNode", registerNode)
	m.HandleFunc("/clearNode", clearNode)

	m.HandleFunc("/addNode", addNode)
	m.HandleFunc("/addBridge", addBridge)
	m.HandleFunc("/addRouter", addRouter)
	m.HandleFunc("/connectNodeToBridge", connectNodeToBridge)
	m.HandleFunc("/connectBridgeToRouter", connectBridgeToRouter)
	m.HandleFunc("/connectRouterToRouter", connectRouterToRouter)
	m.HandleFunc("/inspectNode", inspectNode)
	m.HandleFunc("/inspectBridge", inspectBridge)
	m.HandleFunc("/inspectRouter", inspectRouter)
	m.HandleFunc("/removeNode", removeNode)
	m.HandleFunc("/removeBridge", removeBridge)
	m.HandleFunc("/removeRouter", removeRouter)
	m.HandleFunc("/disconnectNode", disconnectNode)
	m.HandleFunc("/disconnectBridge", disconnectBridge)
	m.HandleFunc("/disconnectRouters", disconnectRouters)

	m.HandleFunc("/weights", routerWeights)
	m.HandleFunc("/trade", trade)
	m.HandleFunc("/connectRouterToRouterRemote", connectRouterToRouterRemote)

	m.HandleFunc("/sniffNode", sniffNode)
	m.HandleFunc("/sniffBridge", sniffBridge)
	m.HandleFunc("/sniffRouter", sniffRouters)
	m.HandleFunc("/stopSniffNode", stopSniffNode)
	m.HandleFunc("/stopSniffBridge", stopSniffBridge)
	m.HandleFunc("/stopSniffRouters", stopSniffRouters)
	m.HandleFunc("/listSniffers", listSniffers)

	m.HandleFunc("/interceptNode", interceptNode)
	m.HandleFunc("/interceptBridge", interceptBridge)
	m.HandleFunc("/interceptRouter", interceptRouters)
	m.HandleFunc("/stopInterceptNode", stopInterceptNode)
	m.HandleFunc("/stopInterceptBridge", stopInterceptBridge)
	m.HandleFunc("/stopInterceptRouters", stopInterceptRouters)
	m.HandleFunc("/listIntercepts", listIntercepts)

	m.HandleFunc("/pause", pause)
	m.HandleFunc("/unpause", unpause)

	m.HandleFunc("/registerClusterNode", cd.RegisterClusterNode)

	httpServer := http.Server{
		Handler: m,
	}
	return &server{&httpServer, socket, app}
}

func Serve() {
	fmt.Println("Serving...")
	if err := engine.httpServer.Serve(engine.socket); err != nil {
		log.Fatal(err)
	}
}

func ping(w http.ResponseWriter, r *http.Request) {
	return
}

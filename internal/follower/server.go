package follower

import (
	"fmt"
	"github.com/David-Antunes/gone/internal"
	"github.com/David-Antunes/gone/internal/application"
	"github.com/David-Antunes/gone/internal/cluster"
	"log"
	"net"
	"net/http"
	"os"
	"runtime/pprof"
)

type server struct {
	httpServer *http.Server
	socket     net.Listener
	app        *application.Follower
	cd         *cluster.ClusterDaemon
	profiling  bool
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

	httpServer := http.Server{
		Handler: m,
	}

	s := &server{&httpServer, socket, app, cd, false}
	m.HandleFunc("/ping", ping)

	m.HandleFunc("/registerNode", registerNode)
	m.HandleFunc("/clearNode", clearNode)

	m.HandleFunc("/addNode", addNode)
	m.HandleFunc("/addBridge", addBridge)
	m.HandleFunc("/addRouter", addRouter)

	m.HandleFunc("/connectNodeToBridge", connectNodeToBridge)
	m.HandleFunc("/connectBridgeToRouter", connectBridgeToRouter)
	m.HandleFunc("/connectRouterToRouter", connectRouterToRouter)
	m.HandleFunc("/connectRouterToRouterRemote", connectRouterToRouterRemote)

	m.HandleFunc("/inspectNode", inspectNode)
	m.HandleFunc("/inspectBridge", inspectBridge)
	m.HandleFunc("/inspectRouter", inspectRouter)

	m.HandleFunc("/removeNode", removeNode)
	m.HandleFunc("/removeBridge", removeBridge)
	m.HandleFunc("/removeRouter", removeRouter)

	m.HandleFunc("/disconnectNode", disconnectNode)
	m.HandleFunc("/disconnectBridge", disconnectBridge)
	m.HandleFunc("/disconnectRouters", disconnectRouters)
	m.HandleFunc("/localDisconnect", localDisconnect)

	m.HandleFunc("/weights", routerWeights)
	m.HandleFunc("/trade", trade)

	m.HandleFunc("/forget", forget)
	m.HandleFunc("/propagate", propagate)

	m.HandleFunc("/sniffNode", sniffNode)
	m.HandleFunc("/sniffBridge", sniffBridge)
	m.HandleFunc("/sniffRouters", sniffRouters)
	m.HandleFunc("/stopSniff", stopSniff)
	m.HandleFunc("/listSniffers", listSniffers)

	m.HandleFunc("/interceptNode", interceptNode)
	m.HandleFunc("/interceptBridge", interceptBridge)
	m.HandleFunc("/interceptRouter", interceptRouters)
	m.HandleFunc("/stopIntercept", stopIntercept)
	m.HandleFunc("/listIntercepts", listIntercepts)

	m.HandleFunc("/disruptNode", disruptNode)
	m.HandleFunc("/disruptBridge", disruptBridge)
	m.HandleFunc("/disruptRouters", disruptRouters)
	m.HandleFunc("/stopDisruptNode", stopDisruptNode)
	m.HandleFunc("/stopDisruptBridge", stopDisruptBridge)
	m.HandleFunc("/stopDisruptRouters", stopDisruptRouters)

	m.HandleFunc("/stopBridge", stopBridge)
	m.HandleFunc("/stopRouter", stopRouter)
	m.HandleFunc("/startBridge", startBridge)
	m.HandleFunc("/startRouter", startRouter)

	m.HandleFunc("/pause", pause)
	m.HandleFunc("/unpause", unpause)

	m.HandleFunc("/registerClusterNode", cd.RegisterClusterNode)

	m.HandleFunc("/profile", s.profile)
	m.HandleFunc("/stopProfile", s.stopProfile)
	m.HandleFunc("/localQuery", localQuery)

	return s
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

func (server *server) profile(w http.ResponseWriter, r *http.Request) {

	if !server.profiling {
		server.profiling = true
		f, err := os.Create("/tmp/profiler.prof")
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
	}
	return
}

func (server *server) stopProfile(w http.ResponseWriter, r *http.Request) {
	if server.profiling {
		server.profiling = false
		pprof.StopCPUProfile()
		return
	}
}

func localQuery(w http.ResponseWriter, r *http.Request) {

	internal.LocalQuery = !internal.LocalQuery

	if internal.LocalQuery {
		fmt.Println("Turned on Local Query")
	} else {
		fmt.Println("Turned off Local Query")
	}
}

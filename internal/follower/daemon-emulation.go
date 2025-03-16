package follower

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/David-Antunes/gone/api"
	addApi "github.com/David-Antunes/gone/api/Add"
	connectApi "github.com/David-Antunes/gone/api/Connect"
	disconnectApi "github.com/David-Antunes/gone/api/Disconnect"
	apiErrors "github.com/David-Antunes/gone/api/Errors"
	inspectApi "github.com/David-Antunes/gone/api/Inspect"
	removeApi "github.com/David-Antunes/gone/api/Remove"
	internal "github.com/David-Antunes/gone/internal/api"
	"github.com/David-Antunes/gone/internal/daemon"
	"log"
	"net/http"
	"os"
)

var daemonLog = log.New(os.Stdout, "follower INFO: ", log.Ltime)

func addNode(w http.ResponseWriter, r *http.Request) {

	req := &addApi.AddNodeRequest{}

	if err := daemon.ParseRequest(r, req); err != nil {
		daemonLog.Println("addNode", err)

		daemon.SendError(w, &addApi.AddNodeResponse{
			Id:        "",
			Mac:       "",
			Ip:        "",
			MachineId: "",
			Error: apiErrors.Error{
				ErrCode: 1,
				ErrMsg:  err.Error(),
			},
		})
		return
	}
	machineId := req.MachineId
	if machineId == "" {
		machineId = engine.app.GetMachineId()
	}
	id, mac, ip, err := engine.app.AddNode(machineId, req.DockerCmd)

	if err != nil {
		daemonLog.Println("addNode", err)
		daemon.SendError(w, &addApi.AddNodeResponse{
			Id:        "",
			Mac:       "",
			Ip:        "",
			MachineId: "",
			Error: apiErrors.Error{
				ErrCode: 1,
				ErrMsg:  err.Error(),
			},
		})
		return
	}
	daemon.SendResponse(w, &addApi.AddNodeResponse{
		Id:        id,
		Mac:       mac,
		Ip:        ip,
		MachineId: engine.app.GetMachineId(),
		Error:     apiErrors.Error{},
	})

	daemonLog.Println("addNode:", "Added node", id, mac, ip, machineId)
}

func addBridge(w http.ResponseWriter, r *http.Request) {
	req := &addApi.AddBridgeRequest{}

	if err := daemon.ParseRequest(r, req); err != nil {
		daemonLog.Println("addBridge:", err)
		daemon.SendError(w, &addApi.AddBridgeResponse{
			Name:      "",
			MachineId: "",
			Error: apiErrors.Error{
				ErrCode: 1,
				ErrMsg:  err.Error(),
			},
		})
		return
	}

	machineId := req.MachineId
	if machineId == "" {
		machineId = engine.app.GetMachineId()
	}
	_, err := engine.app.AddBridge(machineId, req.Name)

	if err != nil {
		daemonLog.Println("addBridge:", err)
		daemon.SendError(w, &addApi.AddBridgeResponse{
			Name:      "",
			MachineId: "",
			Error: apiErrors.Error{
				ErrCode: 1,
				ErrMsg:  err.Error(),
			},
		})
		return
	}
	daemon.SendResponse(w, &addApi.AddBridgeResponse{
		Name:      req.Name,
		MachineId: req.MachineId,
		Error:     apiErrors.Error{},
	})
	daemonLog.Println("addBridge:", "Added bridge", req.Name, "to", req.MachineId)
}

func addRouter(w http.ResponseWriter, r *http.Request) {
	req := &addApi.AddRouterRequest{}

	if err := daemon.ParseRequest(r, req); err != nil {
		daemonLog.Println("addRouter:", err)
		daemon.SendError(w, &addApi.AddRouterResponse{
			Name:      "",
			MachineId: "",
			Error: apiErrors.Error{
				ErrCode: 1,
				ErrMsg:  err.Error(),
			},
		})
		return
	}

	machineId := req.MachineId
	if machineId == "" {
		machineId = engine.app.GetMachineId()
	}
	_, err := engine.app.AddRouter(machineId, req.Name)

	if err != nil {
		daemonLog.Println("addRouter:", err)
		daemon.SendError(w, &addApi.AddRouterResponse{
			Name:      "",
			MachineId: "",
			Error: apiErrors.Error{
				ErrCode: 1,
				ErrMsg:  err.Error(),
			},
		})
		return
	}
	daemon.SendResponse(w, &addApi.AddRouterResponse{
		Name:      req.Name,
		MachineId: req.MachineId,
		Error:     apiErrors.Error{},
	})
	daemonLog.Println("addRouter:", "Added router", req.Name, "to", req.MachineId)
}

func connectNodeToBridge(w http.ResponseWriter, r *http.Request) {

	req := &connectApi.ConnectNodeToBridgeRequest{}

	if err := daemon.ParseRequest(r, req); err != nil {
		daemonLog.Println("connectNodeToBridge:", err)
		daemon.SendError(w, &connectApi.ConnectNodeToBridgeResponse{
			Node:   "",
			Bridge: "",
			Error: apiErrors.Error{
				ErrCode: 1,
				ErrMsg:  err.Error(),
			},
		})
		return
	}
	linkProps, err := daemon.ParseLinkProps(req.Latency, req.Bandwidth, req.Jitter, req.DropRate, req.Weight)

	if err != nil {
		daemonLog.Println("connectNodeToBridge:", err)
		daemon.SendError(w, &connectApi.ConnectNodeToBridgeResponse{
			Node:   "",
			Bridge: "",
			Error: apiErrors.Error{
				ErrCode: 1,
				ErrMsg:  err.Error(),
			},
		})
		return
	}

	err = engine.app.ConnectNodeToBridge(req.Node, req.Bridge, linkProps)

	if err != nil {

		daemonLog.Println("connectNodeToBridge:", err)
		daemon.SendError(w, &connectApi.ConnectNodeToBridgeResponse{
			Node:   "",
			Bridge: "",
			Error: apiErrors.Error{
				ErrCode: 1,
				ErrMsg:  err.Error(),
			},
		})
		return
	}
	daemon.SendResponse(w, &connectApi.ConnectNodeToBridgeResponse{
		Node:   req.Node,
		Bridge: req.Bridge,
		Error:  apiErrors.Error{},
	})

	daemonLog.Println("connectNodeToBridge:", "Connected", req.Node, "to", req.Bridge, "Properties:", linkProps)
}

func connectBridgeToRouter(w http.ResponseWriter, r *http.Request) {

	req := &connectApi.ConnectBridgeToRouterRequest{}

	if err := daemon.ParseRequest(r, req); err != nil {
		daemonLog.Println("connectBridgeToRouter:", err)
		daemon.SendError(w, &connectApi.ConnectBridgeToRouterResponse{
			Bridge: req.Bridge,
			Router: req.Router,
			Error: apiErrors.Error{
				ErrCode: 1,
				ErrMsg:  err.Error(),
			},
		})
		return
	}

	linkProps, err := daemon.ParseLinkProps(req.Latency, req.Bandwidth, req.Jitter, req.DropRate, req.Weight)

	if err != nil {
		daemonLog.Println("connectBridgeToRouter:", err)
		daemon.SendError(w, &connectApi.ConnectBridgeToRouterResponse{
			Bridge: req.Bridge,
			Router: req.Router,
			Error: apiErrors.Error{
				ErrCode: 1,
				ErrMsg:  err.Error(),
			},
		})
		return
	}

	err = engine.app.ConnectBridgeToRouter(req.Bridge, req.Router, linkProps)

	if err != nil {

		daemonLog.Println("connectBridgeToRouter:", err)
		daemon.SendError(w, &connectApi.ConnectBridgeToRouterResponse{
			Bridge: req.Bridge,
			Router: req.Router,
			Error: apiErrors.Error{
				ErrCode: 1,
				ErrMsg:  err.Error(),
			},
		})
		return
	}

	daemon.SendResponse(w, &connectApi.ConnectBridgeToRouterResponse{
		Bridge: req.Bridge,
		Router: req.Router,
		Error:  apiErrors.Error{},
	})
	daemonLog.Println("connectBridgeToRouter:", "Connected", req.Bridge, "to", req.Router, "Properties:", linkProps)
}

func connectRouterToRouter(w http.ResponseWriter, r *http.Request) {

	req := &internal.ConnectRouterToRouterRequest{}

	if err := daemon.ParseRequest(r, req); err != nil {
		daemonLog.Println("connectRouterToRouter:", err)
		daemon.SendError(w, &connectApi.ConnectRouterToRouterResponse{
			From: req.R1,
			To:   req.R2,
			Error: apiErrors.Error{
				ErrCode: 1,
				ErrMsg:  err.Error(),
			},
		})
		return
	}

	linkProps, err := daemon.ParseLinkPropsInternal(req.Latency, req.Bandwidth, req.Jitter, req.DropRate, req.Weight)
	if err != nil {
		daemonLog.Println("connectRouterToRouter:", err)
		daemon.SendError(w, &connectApi.ConnectRouterToRouterResponse{
			From: req.R1,
			To:   req.R2,
			Error: apiErrors.Error{
				ErrCode: 1,
				ErrMsg:  err.Error(),
			},
		})
		return
	}

	err = engine.app.ConnectRouterToRouter(req.R1, req.R2, req.MachineID, linkProps, req.Propagate)

	if err != nil {

		daemonLog.Println("connectRouterToRouter:", err)
		daemon.SendError(w, &connectApi.ConnectRouterToRouterResponse{
			From: req.R1,
			To:   req.R2,
			Error: apiErrors.Error{
				ErrCode: 1,
				ErrMsg:  err.Error(),
			},
		})
		return
	}

	daemon.SendResponse(w, &connectApi.ConnectRouterToRouterResponse{
		From:  req.R1,
		To:    req.R2,
		Error: apiErrors.Error{},
	})

	daemonLog.Println("connectRouterToRouter:", "Connected", req.R1, "to", req.R2, "Properties:", linkProps)
}

func connectRouterToRouterRemote(w http.ResponseWriter, r *http.Request) {

	req := &internal.ConnectRouterToRouterRequest{}

	if err := daemon.ParseRequest(r, req); err != nil {
		daemonLog.Println("connectRouterToRouterRemote:", err)
		daemon.SendError(w, &internal.ConnectRouterToRouterResponse{
			R1:        "",
			R2:        "",
			MachineID: "",
			Error: apiErrors.Error{
				ErrCode: 1,
				ErrMsg:  err.Error(),
			},
		})
		return
	}

	linkProps, err := daemon.ParseLinkPropsInternal(req.Latency, req.Bandwidth, req.Jitter, req.DropRate, req.Weight)
	if err != nil {
		daemonLog.Println("connectRouterToRouterRemote:", err)
		daemon.SendError(w, &internal.ConnectRouterToRouterResponse{
			R1:        "",
			R2:        "",
			MachineID: "",
			Error: apiErrors.Error{
				ErrCode: 1,
				ErrMsg:  err.Error(),
			},
		})
		return
	}

	if req.MachineID != engine.app.GetMachineId() {
		err := engine.app.ApplyConnectRouterToRouterRemote(req.R1, req.R2, req.MachineID, linkProps, req.Propagate)
		if err != nil {

			daemonLog.Println("connectRouterToRouterRemote:", err)
			daemon.SendError(w, internal.ConnectRouterToRouterResponse{
				R1:        "",
				R2:        "",
				MachineID: "",
				Error: apiErrors.Error{
					ErrCode: 1,
					ErrMsg:  err.Error(),
				},
			})
			return
		}
	} else {

		daemonLog.Println("connectRouterToRouterRemote:", "invalid apply operation")
		daemon.SendError(w, internal.ConnectRouterToRouterResponse{
			R1:        "",
			R2:        "",
			MachineID: "",
			Error: apiErrors.Error{
				ErrCode: 1,
				ErrMsg:  errors.New("invalid apply operation").Error(),
			},
		})
		return
	}

	daemon.SendResponse(w, &internal.ConnectRouterToRouterResponse{
		R1:        req.R1,
		R2:        req.R2,
		MachineID: req.MachineID,
		Error:     apiErrors.Error{},
	})
	daemonLog.Println("connectRouterToRouterRemote:", "Connected", req.R1, "to", req.R2)

}

func removeNode(w http.ResponseWriter, r *http.Request) {
	req := &removeApi.RemoveNodeRequest{}

	if err := daemon.ParseRequest(r, req); err != nil {
		daemonLog.Println("removeNode:", err)
		daemon.SendError(w, &removeApi.RemoveNodeResponse{
			Name: req.Name,
			Error: apiErrors.Error{
				ErrCode: 1,
				ErrMsg:  err.Error(),
			},
		})
		return
	}

	err := engine.app.RemoveNode(req.Name)

	if err != nil {
		daemonLog.Println("removeNode:", err)
		daemon.SendError(w, &removeApi.RemoveNodeResponse{
			Name: req.Name,
			Error: apiErrors.Error{
				ErrCode: 1,
				ErrMsg:  err.Error(),
			},
		})
		return
	}

	daemon.SendResponse(w, &removeApi.RemoveNodeResponse{Name: req.Name, Error: apiErrors.Error{}})

	daemonLog.Println("removeNode:", "Removed", req.Name)
}

func removeBridge(w http.ResponseWriter, r *http.Request) {
	req := &removeApi.RemoveBridgeRequest{}

	if err := daemon.ParseRequest(r, req); err != nil {
		daemonLog.Println("removeBridge:", err)
		daemon.SendError(w, &removeApi.RemoveBridgeResponse{
			Name: req.Name,
			Error: apiErrors.Error{
				ErrCode: 1,
				ErrMsg:  err.Error(),
			},
		})
		return
	}

	err := engine.app.RemoveBridge(req.Name)

	if err != nil {
		daemonLog.Println("removeBridge:", err)
		daemon.SendError(w, &removeApi.RemoveBridgeResponse{
			Name: req.Name,
			Error: apiErrors.Error{
				ErrCode: 1,
				ErrMsg:  err.Error(),
			},
		})
		return
	}

	daemon.SendResponse(w, &removeApi.RemoveBridgeResponse{Name: req.Name, Error: apiErrors.Error{}})
	daemonLog.Println("removeBridge:", "Removed", req.Name)
}

func removeRouter(w http.ResponseWriter, r *http.Request) {
	req := &removeApi.RemoveRouterRequest{}

	if err := daemon.ParseRequest(r, req); err != nil {
		daemonLog.Println("removeRouter:", err)
		daemon.SendError(w, &removeApi.RemoveRouterResponse{
			Name: req.Name,
			Error: apiErrors.Error{
				ErrCode: 1,
				ErrMsg:  err.Error(),
			},
		})
		return
	}

	err := engine.app.RemoveRouter(req.Name)

	if err != nil {
		daemonLog.Println("removeRouter:", err)
		daemon.SendError(w, &removeApi.RemoveRouterResponse{
			Name: req.Name,
			Error: apiErrors.Error{
				ErrCode: 1,
				ErrMsg:  err.Error(),
			},
		})
		return
	}

	daemon.SendResponse(w, &removeApi.RemoveRouterResponse{Name: req.Name, Error: apiErrors.Error{}})
	daemonLog.Println("removeRouter:", "Removed", req.Name)
}

func disconnectNode(w http.ResponseWriter, r *http.Request) {
	req := &disconnectApi.DisconnectNodeRequest{}

	if err := daemon.ParseRequest(r, req); err != nil {
		daemonLog.Println("DisconnectNode:", err)
		daemon.SendError(w, &disconnectApi.DisconnectNodeResponse{
			Name: req.Name,
			Error: apiErrors.Error{
				ErrCode: 1,
				ErrMsg:  err.Error(),
			},
		})
		return
	}

	err := engine.app.DisconnectNode(req.Name)

	if err != nil {
		daemonLog.Println("disconnectNode:", err)
		daemon.SendError(w, &disconnectApi.DisconnectNodeResponse{
			Name: req.Name,
			Error: apiErrors.Error{
				ErrCode: 1,
				ErrMsg:  err.Error(),
			},
		})
		return
	}

	daemon.SendResponse(w, &disconnectApi.DisconnectNodeResponse{Name: req.Name, Error: apiErrors.Error{}})
	daemonLog.Println("disconnectNode:", "Disconnected", req.Name)
}

func disconnectBridge(w http.ResponseWriter, r *http.Request) {
	req := &disconnectApi.DisconnectBridgeRequest{}

	if err := daemon.ParseRequest(r, req); err != nil {
		daemonLog.Println("DisconnectBridge:", err)
		daemon.SendError(w, &disconnectApi.DisconnectBridgeResponse{
			Name: req.Name,
			Error: apiErrors.Error{
				ErrCode: 1,
				ErrMsg:  err.Error(),
			},
		})
		return
	}

	err := engine.app.DisconnectBridge(req.Name)

	if err != nil {
		daemonLog.Println("disconnectBridge:", err)
		daemon.SendError(w, &disconnectApi.DisconnectBridgeResponse{
			Name: req.Name,
			Error: apiErrors.Error{
				ErrCode: 1,
				ErrMsg:  err.Error(),
			},
		})
		return
	}

	daemon.SendResponse(w, &disconnectApi.DisconnectBridgeResponse{Name: req.Name, Error: apiErrors.Error{}})
	daemonLog.Println("disconnectBridge:", "Disconnected", req.Name)
}
func disconnectRouters(w http.ResponseWriter, r *http.Request) {
	req := &disconnectApi.DisconnectRoutersRequest{}

	if err := daemon.ParseRequest(r, req); err != nil {
		daemonLog.Println("DisconnectRouters:", err)
		daemon.SendError(w, &disconnectApi.DisconnectRoutersResponse{
			First:  req.First,
			Second: req.Second,
			Error: apiErrors.Error{
				ErrCode: 1,
				ErrMsg:  err.Error(),
			},
		})
		return
	}

	err := engine.app.DisconnectRouters(req.First, req.Second)

	if err != nil {
		daemonLog.Println("disconnectRouters:", err)
		daemon.SendError(w, &disconnectApi.DisconnectRoutersResponse{
			First:  req.First,
			Second: req.Second,
			Error: apiErrors.Error{
				ErrCode: 1,
				ErrMsg:  err.Error(),
			},
		})
		return
	}

	daemon.SendResponse(w, &disconnectApi.DisconnectRoutersResponse{
		First:  req.First,
		Second: req.Second,
		Error:  apiErrors.Error{},
	})
	daemonLog.Println("disconnectRouters:", "Disconnected", req.First, "from", req.Second)
}

func localDisconnect(w http.ResponseWriter, r *http.Request) {
	req := &disconnectApi.DisconnectRoutersRequest{}

	if err := daemon.ParseRequest(r, req); err != nil {
		daemonLog.Println("localDisconnect:", err)
		daemon.SendError(w, &disconnectApi.DisconnectRoutersResponse{
			First:  req.First,
			Second: req.Second,
			Error: apiErrors.Error{
				ErrCode: 1,
				ErrMsg:  err.Error(),
			},
		})
		return
	}

	err := engine.app.LocalDisconnect(req.First, req.Second)

	if err != nil {
		daemonLog.Println("localDisconnect:", err)
		daemon.SendError(w, &disconnectApi.DisconnectRoutersResponse{
			First:  req.First,
			Second: req.Second,
			Error: apiErrors.Error{
				ErrCode: 1,
				ErrMsg:  err.Error(),
			},
		})
		return
	}

	daemon.SendResponse(w, &disconnectApi.DisconnectRoutersResponse{
		First:  req.First,
		Second: req.Second,
		Error:  apiErrors.Error{},
	})
	daemonLog.Println("localDisconnect:", "Disconnected", req.First, "from", req.Second)
}

func inspectNode(w http.ResponseWriter, r *http.Request) {

	req := &inspectApi.InspectNodeRequest{}

	if err := daemon.ParseRequest(r, req); err != nil {
		daemonLog.Println("inspectNode:", err)
		daemon.SendError(w, &inspectApi.InspectNodeResponse{
			Node: api.Node{},
			Error: apiErrors.Error{
				ErrCode: 1,
				ErrMsg:  err.Error(),
			},
		})
		return
	}

	if node, ok := engine.app.GetNode(req.Name); ok {
		if node.MachineId == engine.app.GetMachineId() {
			daemon.SendResponse(w, inspectApi.InspectNodeResponse{
				Node:  node,
				Error: apiErrors.Error{},
			})
			return
		} else {
			msg, err := engine.cd.Cl.SendMsg(node.MachineId, req, "inspectNode")
			if err != nil {
				daemonLog.Println("inspectNode:", err)
				daemon.SendError(w, &inspectApi.InspectNodeResponse{
					Node: api.Node{},
					Error: apiErrors.Error{
						ErrCode: 1,
						ErrMsg:  err.Error(),
					},
				})
				return
			}
			d := json.NewDecoder(msg.Body)
			resp := &inspectApi.InspectNodeResponse{}
			err = d.Decode(&resp)
			if err != nil {
				daemonLog.Println("inspectNode:", err)
				daemon.SendError(w, &inspectApi.InspectNodeResponse{
					Node: api.Node{},
					Error: apiErrors.Error{
						ErrCode: 1,
						ErrMsg:  err.Error(),
					},
				})
				return
			}
			daemon.SendResponse(w, resp)
		}

	} else {
		daemonLog.Println("inspectNode: Invalid node id")
		daemon.SendError(w, &inspectApi.InspectNodeResponse{
			Node: api.Node{},
			Error: apiErrors.Error{
				ErrCode: 1,
				ErrMsg:  "Node doesn't exist.",
			},
		})
		return
	}
}

func inspectBridge(w http.ResponseWriter, r *http.Request) {

	req := &inspectApi.InspectBridgeRequest{}

	if err := daemon.ParseRequest(r, req); err != nil {
		daemonLog.Println("inspectBridge:", err)
		daemon.SendError(w, &inspectApi.InspectBridgeResponse{
			Bridge: api.Bridge{},
			Error: apiErrors.Error{
				ErrCode: 1,
				ErrMsg:  err.Error(),
			},
		})
		return
	}

	if bridge, ok := engine.app.GetBridge(req.Name); ok {
		if bridge.MachineId == engine.app.GetMachineId() {
			daemon.SendResponse(w, inspectApi.InspectBridgeResponse{
				Bridge: bridge,
				Error:  apiErrors.Error{},
			})
			return
		} else {
			msg, err := engine.cd.Cl.SendMsg(bridge.MachineId, req, "inspectBridge")

			if err != nil {
				daemonLog.Println("inspectBridge:", err)
				daemon.SendError(w, &inspectApi.InspectBridgeResponse{
					Bridge: api.Bridge{},
					Error: apiErrors.Error{
						ErrCode: 1,
						ErrMsg:  err.Error(),
					},
				})
				return
			}
			d := json.NewDecoder(msg.Body)
			resp := &inspectApi.InspectBridgeResponse{}
			err = d.Decode(&resp)
			if err != nil {
				daemonLog.Println("inspectBridge:", err)
				daemon.SendError(w, &inspectApi.InspectBridgeResponse{
					Bridge: bridge,
					Error: apiErrors.Error{
						ErrCode: 1,
						ErrMsg:  err.Error(),
					},
				})
				return
			}
			daemon.SendResponse(w, resp)
		}
	} else {
		fmt.Println("Invalid bridge id.")
		daemon.SendError(w, &inspectApi.InspectBridgeResponse{
			Bridge: api.Bridge{},
			Error: apiErrors.Error{
				ErrCode: 1,
				ErrMsg:  "Invalid bridge id.",
			},
		})
		return
	}
}

func inspectRouter(w http.ResponseWriter, r *http.Request) {

	req := &inspectApi.InspectRouterRequest{}

	if err := daemon.ParseRequest(r, req); err != nil {
		daemonLog.Println("inspectRouter:", err)
		daemon.SendError(w, &inspectApi.InspectRouterResponse{
			Router: api.Router{},
			Error: apiErrors.Error{
				ErrCode: 1,
				ErrMsg:  err.Error(),
			},
		})
		return
	}

	if router, ok := engine.app.GetRouter(req.Name); ok {
		if router.MachineId == engine.app.GetMachineId() {

			daemon.SendResponse(w, inspectApi.InspectRouterResponse{
				Router: router,
				Error:  apiErrors.Error{},
			})
			return
		} else {
			msg, err := engine.cd.Cl.SendMsg(router.MachineId, req, "inspectRouter")

			if err != nil {
				daemonLog.Println("inspectRouter:", err)
				daemon.SendError(w, &inspectApi.InspectRouterResponse{
					Router: api.Router{},
					Error: apiErrors.Error{
						ErrCode: 1,
						ErrMsg:  err.Error(),
					},
				})
				return
			}
			d := json.NewDecoder(msg.Body)
			resp := &inspectApi.InspectRouterResponse{}
			err = d.Decode(&resp)
			if err != nil {
				daemonLog.Println("inspectRouter:", err)
				daemon.SendError(w, &inspectApi.InspectRouterResponse{
					Router: api.Router{},
					Error: apiErrors.Error{
						ErrCode: 1,
						ErrMsg:  err.Error(),
					},
				})
				return
			}
			daemon.SendResponse(w, resp)
		}
	} else {
		daemonLog.Println("inspectRouter: Invalid router id")
		daemon.SendError(w, &inspectApi.InspectRouterResponse{
			Router: api.Router{},
			Error: apiErrors.Error{
				ErrCode: 1,
				ErrMsg:  "Invalid router id.",
			},
		})
		return
	}
}

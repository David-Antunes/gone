package follower

import (
	"fmt"
	apiErrors "github.com/David-Antunes/gone/api/Errors"
	opApi "github.com/David-Antunes/gone/api/Operations"
	"github.com/David-Antunes/gone/internal/api"
	"github.com/David-Antunes/gone/internal/daemon"
	"net/http"
)

func registerNode(w http.ResponseWriter, r *http.Request) {

	req := &api.RegisterNodeRequest{}

	if err := daemon.ParseRequest(r, req); err != nil {
		daemonLog.Println("registerNode", err)
		daemon.SendError(w, &api.RegisterNodeResponse{
			Id:        "",
			Ip:        "",
			Mac:       "",
			MachineId: "",
			Error: apiErrors.Error{
				ErrCode: 1,
				ErrMsg:  err.Error(),
			},
		})
		return
	}

	err := engine.app.RegisterNode(req.Id, req.Mac, req.Ip, req.MachineId)

	if err != nil {
		daemonLog.Println("registerNode", err)
		daemon.SendError(w, &api.RegisterNodeResponse{
			Id:        "",
			Ip:        "",
			Mac:       "",
			MachineId: "",
			Error: apiErrors.Error{
				ErrCode: 1,
				ErrMsg:  err.Error(),
			},
		})
		return
	}

	daemonLog.Println("registerNode", "registered", req.Id, req.Mac, req.Ip, req.MachineId)
}

func clearNode(w http.ResponseWriter, r *http.Request) {

	req := &api.ClearNodeRequest{}

	if err := daemon.ParseRequest(r, req); err != nil {
		daemonLog.Println("clearNode:", err)
		daemon.SendError(w, &api.ClearNodeResponse{
			Id: req.Id,
			Error: apiErrors.Error{
				ErrCode: 1,
				ErrMsg:  err.Error(),
			},
		})
		return
	}

	err := engine.app.ClearNode(req.Id)

	if err != nil {
		daemonLog.Println("clearNode:", err)
		daemon.SendError(w, &api.ClearNodeResponse{
			Id: req.Id,
			Error: apiErrors.Error{
				ErrCode: 1,
				ErrMsg:  err.Error(),
			},
		})
		return
	}
	daemon.SendResponse(w, &api.ClearNodeResponse{
		Id:    req.Id,
		Error: apiErrors.Error{},
	})

}

func propagate(w http.ResponseWriter, r *http.Request) {

	req := &opApi.PropagateRequest{}

	if err := daemon.ParseRequest(r, req); err != nil {
		daemonLog.Println("propagate:", err)
		daemon.SendError(w, &opApi.PropagateResponse{
			Name: req.Name,
			Error: apiErrors.Error{
				ErrCode: 1,
				ErrMsg:  err.Error(),
			},
		})
		return
	}

	err := engine.app.Propagate(req.Name)

	if err != nil {
		daemonLog.Println("propagate:", err)
		daemon.SendError(w, &opApi.PropagateResponse{
			Name: req.Name,
			Error: apiErrors.Error{
				ErrCode: 1,
				ErrMsg:  err.Error(),
			},
		})
		return

	}
	daemon.SendResponse(w, &opApi.PropagateResponse{
		Name:  req.Name,
		Error: apiErrors.Error{},
	})
}

func forget(w http.ResponseWriter, r *http.Request) {

	req := &opApi.ForgetRequest{}

	if err := daemon.ParseRequest(r, req); err != nil {
		daemonLog.Println("forget:", err)
		daemon.SendError(w, &opApi.ForgetResponse{
			Name: req.Name,
			Error: apiErrors.Error{
				ErrCode: 1,
				ErrMsg:  err.Error(),
			},
		})
		return
	}

	err := engine.app.Forget(req.Name)

	if err != nil {
		daemonLog.Println("forget:", err)
		daemon.SendError(w, &opApi.ForgetResponse{
			Name: req.Name,
			Error: apiErrors.Error{
				ErrCode: 1,
				ErrMsg:  err.Error(),
			},
		})
		return
	}
	daemon.SendResponse(w, &opApi.ForgetResponse{
		Name:  req.Name,
		Error: apiErrors.Error{},
	})

}

func trade(w http.ResponseWriter, r *http.Request) {

	req := &api.TradeRoutesRequest{}

	if err := daemon.ParseRequest(r, req); err != nil {
		fmt.Println("Invalid fields in trade request.")
		daemon.SendError(w, &api.TradeRoutesResponse{
			To:   "",
			From: "",
			Error: apiErrors.Error{
				ErrCode: 1,
				ErrMsg:  err.Error(),
			},
		})
		return
	}

	engine.app.ApplyRoutes(req.To, req.From, req.Weights)
	daemon.SendResponse(w, &api.TradeRoutesResponse{
		To:    req.To,
		From:  req.From,
		Error: apiErrors.Error{},
	})
	daemonLog.Println("trade:", req.To, req.From)
}

func routerWeights(w http.ResponseWriter, r *http.Request) {

	req := &api.GetRouterWeightsRequest{}

	if err := daemon.ParseRequest(r, req); err != nil {
		fmt.Println("Invalid fields in getWeights request.")
		daemon.SendError(w, &api.GetRouterWeightsResponse{
			Weights: nil,
			Error: apiErrors.Error{
				ErrCode: 1,
				ErrMsg:  err.Error(),
			},
		})
		return
	}

	weights := engine.app.GetRouterWeights(req.Router)
	daemon.SendResponse(w, &api.GetRouterWeightsResponse{
		Weights: weights,
		Error:   apiErrors.Error{},
	})
	daemonLog.Println("weights:", req.Router)
}
func sniffNode(w http.ResponseWriter, r *http.Request) {

	req := &opApi.SniffNodeRequest{}

	if err := daemon.ParseRequest(r, req); err != nil {
		daemonLog.Println("sniffNode:", err)
		daemon.SendError(w, &opApi.SniffNodeResponse{
			Node: req.Node,
			Id:   req.Id,
			Error: apiErrors.Error{
				ErrCode: 1,
				ErrMsg:  err.Error(),
			},
		})
		return
	}

	id, path, machineId, err := engine.app.SniffNode(req.Node, req.Id)

	if err != nil {
		daemonLog.Println("sniffNode:", err)
		daemon.SendError(w, &opApi.SniffNodeResponse{
			Node: req.Node,
			Id:   req.Id,
			Error: apiErrors.Error{
				ErrCode: 1,
				ErrMsg:  err.Error(),
			},
		})
		return
	}
	daemon.SendResponse(w, &opApi.SniffNodeResponse{
		Node:      req.Node,
		Id:        id,
		Path:      path,
		MachineId: machineId,
		Error:     apiErrors.Error{},
	})

}
func sniffBridge(w http.ResponseWriter, r *http.Request) {

	req := &opApi.SniffBridgeRequest{}

	if err := daemon.ParseRequest(r, req); err != nil {
		daemonLog.Println("sniffBridge:", err)
		daemon.SendError(w, &opApi.SniffBridgeResponse{
			Bridge: req.Bridge,
			Id:     req.Id,
			Error: apiErrors.Error{
				ErrCode: 1,
				ErrMsg:  err.Error(),
			},
		})
		return
	}

	id, path, machineId, err := engine.app.SniffBridge(req.Bridge, req.Id)

	if err != nil {
		daemonLog.Println("sniffBridge:", err)
		daemon.SendError(w, &opApi.SniffBridgeResponse{
			Bridge: req.Bridge,
			Id:     req.Id,
			Error: apiErrors.Error{
				ErrCode: 1,
				ErrMsg:  err.Error(),
			},
		})
		return
	}
	daemon.SendResponse(w, &opApi.SniffBridgeResponse{
		Id:        id,
		Bridge:    req.Bridge,
		Path:      path,
		MachineId: machineId,
		Error:     apiErrors.Error{},
	})

}

func sniffRouters(w http.ResponseWriter, r *http.Request) {

	req := &opApi.SniffRoutersRequest{}

	if err := daemon.ParseRequest(r, req); err != nil {
		daemonLog.Println("sniffRouters:", err)
		daemon.SendError(w, &opApi.SniffRoutersResponse{
			Router1: req.Router1,
			Router2: req.Router2,
			Id:      req.Id,
			Error: apiErrors.Error{
				ErrCode: 1,
				ErrMsg:  err.Error(),
			},
		})
		return
	}

	id, path, machineId, err := engine.app.SniffRouters(req.Router1, req.Router2, req.Id)

	if err != nil {
		daemonLog.Println("sniffRouters:", err)
		daemon.SendError(w, &opApi.SniffRoutersResponse{
			Router1: req.Router1,
			Router2: req.Router2,
			Id:      req.Id,
			Error: apiErrors.Error{
				ErrCode: 1,
				ErrMsg:  err.Error(),
			},
		})
		return
	}
	daemon.SendResponse(w, &opApi.SniffRoutersResponse{
		Router1:   req.Router1,
		Router2:   req.Router2,
		Id:        id,
		Path:      path,
		MachineId: machineId,
		Error:     apiErrors.Error{},
	})

}

func stopSniff(w http.ResponseWriter, r *http.Request) {

	req := &opApi.StopSniffRequest{}

	if err := daemon.ParseRequest(r, req); err != nil {
		daemonLog.Println("stopSniffNode:", err)
		daemon.SendError(w, &opApi.StopSniffResponse{
			Id: req.Id,
			Error: apiErrors.Error{
				ErrCode: 1,
				ErrMsg:  err.Error(),
			},
		})
		return
	}

	err := engine.app.StopSniff(req.Id)

	if err != nil {
		daemonLog.Println("stopSniffNode:", err)
		daemon.SendError(w, &opApi.StopSniffResponse{
			Id: req.Id,
			Error: apiErrors.Error{
				ErrCode: 1,
				ErrMsg:  err.Error(),
			},
		})
		return
	}
	daemon.SendResponse(w, &opApi.StopSniffResponse{
		Id:    req.Id,
		Error: apiErrors.Error{},
	})

}

func listSniffers(w http.ResponseWriter, r *http.Request) {

	ids := engine.app.ListSniffers()

	daemon.SendResponse(w, &opApi.ListSniffersResponse{
		Sniffers: ids,
	})

}
func interceptNode(w http.ResponseWriter, r *http.Request) {

	req := &opApi.InterceptNodeRequest{}

	if err := daemon.ParseRequest(r, req); err != nil {
		daemonLog.Println("interceptNode:", err)
		daemon.SendError(w, &opApi.InterceptNodeResponse{
			Node:      req.Node,
			Id:        req.Id,
			Path:      "",
			MachineId: "",
			Error: apiErrors.Error{
				ErrCode: 1,
				ErrMsg:  err.Error(),
			},
		})
		return
	}

	id, path, machineId, err := engine.app.InterceptNode(req.Node, req.Id, req.Direction)

	if err != nil {
		daemonLog.Println("interceptNode:", err)
		daemon.SendError(w, &opApi.InterceptNodeResponse{
			Node:      req.Node,
			Id:        req.Id,
			Path:      "",
			MachineId: "",
			Error: apiErrors.Error{
				ErrCode: 1,
				ErrMsg:  err.Error(),
			},
		})
		return
	}
	daemon.SendResponse(w, &opApi.InterceptNodeResponse{
		Id:        id,
		Path:      path,
		MachineId: machineId,
		Error:     apiErrors.Error{},
	})
}

func interceptBridge(w http.ResponseWriter, r *http.Request) {

	req := &opApi.InterceptBridgeRequest{}

	if err := daemon.ParseRequest(r, req); err != nil {
		daemonLog.Println("interceptBridge:", err)
		daemon.SendError(w, &opApi.InterceptBridgeResponse{
			Id:        req.Id,
			Bridge:    req.Bridge,
			Path:      "",
			MachineId: "",
			Error: apiErrors.Error{
				ErrCode: 1,
				ErrMsg:  err.Error(),
			},
		})
		return
	}

	id, path, machineId, err := engine.app.InterceptBridge(req.Bridge, req.Id, req.Direction)

	if err != nil {
		daemonLog.Println("interceptBridge:", err)
		daemon.SendError(w, &opApi.InterceptBridgeResponse{
			Id:        req.Id,
			Bridge:    req.Bridge,
			Path:      "",
			MachineId: "",
			Error: apiErrors.Error{
				ErrCode: 1,
				ErrMsg:  err.Error(),
			},
		})
		return
	}
	daemon.SendResponse(w, &opApi.InterceptBridgeResponse{
		Id:        id,
		Path:      path,
		MachineId: machineId,
		Error:     apiErrors.Error{},
	})
}

func interceptRouters(w http.ResponseWriter, r *http.Request) {

	req := &opApi.InterceptRoutersRequest{}

	if err := daemon.ParseRequest(r, req); err != nil {
		daemonLog.Println("interceptRouters:", err)
		daemon.SendError(w, &opApi.InterceptRoutersResponse{
			Router1:   req.Router1,
			Router2:   req.Router2,
			Id:        req.Id,
			Path:      "",
			MachineId: "",
			Error: apiErrors.Error{
				ErrCode: 1,
				ErrMsg:  err.Error(),
			},
		})
		return
	}

	id, path, machineId, err := engine.app.InterceptRouters(req.Router1, req.Router2, req.Id, req.Direction)

	if err != nil {
		daemonLog.Println("interceptRouters:", err)
		daemon.SendError(w, &opApi.InterceptRoutersResponse{
			Router1: req.Router1,
			Router2: req.Router2,
			Id:      req.Id,

			Error: apiErrors.Error{
				ErrCode: 1,
				ErrMsg:  err.Error(),
			},
		})
		return
	}
	daemon.SendResponse(w, &opApi.InterceptRoutersResponse{
		Id:        id,
		Path:      path,
		MachineId: machineId,
		Error:     apiErrors.Error{},
	})
}

func stopIntercept(w http.ResponseWriter, r *http.Request) {

	req := &opApi.StopInterceptRequest{}

	if err := daemon.ParseRequest(r, req); err != nil {
		daemonLog.Println("stopInterceptNode:", err)
		daemon.SendError(w, &opApi.StopInterceptResponse{
			Id: req.Id,
			Error: apiErrors.Error{
				ErrCode: 1,
				ErrMsg:  err.Error(),
			},
		})
		return
	}

	err := engine.app.StopIntercept(req.Id)

	if err != nil {
		daemonLog.Println("stopInterceptNode:", err)
		daemon.SendError(w, &opApi.StopInterceptResponse{
			Id: req.Id,
			Error: apiErrors.Error{
				ErrCode: 1,
				ErrMsg:  err.Error(),
			},
		})
		return
	}
	daemon.SendResponse(w, &opApi.StopInterceptResponse{
		Id:    req.Id,
		Error: apiErrors.Error{},
	})
}

func listIntercepts(w http.ResponseWriter, r *http.Request) {

	ids := engine.app.ListIntercepts()

	daemon.SendResponse(w, &opApi.ListInterceptsResponse{
		Intercepts: ids,
	})
}
func pause(w http.ResponseWriter, r *http.Request) {

	req := &opApi.PauseRequest{}

	if err := daemon.ParseRequest(r, req); err != nil {
		daemonLog.Println("pause:", err)
		daemon.SendError(w, &opApi.PauseResponse{
			Id:  req.Id,
			All: req.All,
			Error: apiErrors.Error{
				ErrCode: 1,
				ErrMsg:  err.Error(),
			},
		})
		return
	}

	err := engine.app.Pause(req.Id, req.All)

	if err != nil {
		daemonLog.Println("pause:", err)
		daemon.SendError(w, &opApi.PauseResponse{
			Id:  req.Id,
			All: req.All,
			Error: apiErrors.Error{
				ErrCode: 1,
				ErrMsg:  err.Error(),
			},
		})
		return
	}
	daemon.SendResponse(w, &opApi.PauseResponse{
		Id:    req.Id,
		All:   req.All,
		Error: apiErrors.Error{},
	})

}
func unpause(w http.ResponseWriter, r *http.Request) {

	req := &opApi.UnpauseRequest{}

	if err := daemon.ParseRequest(r, req); err != nil {
		daemonLog.Println("unpause:", err)
		daemon.SendError(w, &opApi.UnpauseResponse{
			Id:  req.Id,
			All: req.All,
			Error: apiErrors.Error{
				ErrCode: 1,
				ErrMsg:  err.Error(),
			},
		})
		return
	}

	err := engine.app.Unpause(req.Id, req.All)

	if err != nil {
		daemonLog.Println("unpause:", err)
		daemon.SendError(w, &opApi.UnpauseResponse{
			Id:  req.Id,
			All: req.All,
			Error: apiErrors.Error{
				ErrCode: 1,
				ErrMsg:  err.Error(),
			},
		})
		return
	}
	daemon.SendResponse(w, &opApi.UnpauseResponse{
		Id:    req.Id,
		All:   req.All,
		Error: apiErrors.Error{},
	})
}
func disruptNode(w http.ResponseWriter, r *http.Request) {

	req := &opApi.DisruptNodeRequest{}
	if err := daemon.ParseRequest(r, req); err != nil {
		daemonLog.Println("disruptNode:", err)
		daemon.SendError(w, &opApi.DisruptNodeResponse{
			Node: req.Node,
			Error: apiErrors.Error{
				ErrCode: 1, ErrMsg: err.Error(),
			},
		})
		return
	}

	err := engine.app.DisruptNode(req.Node)

	if err != nil {
		daemonLog.Println("disruptNode:", err)
		daemon.SendError(w, &opApi.DisruptNodeResponse{
			Node: req.Node,
			Error: apiErrors.Error{
				ErrCode: 1,
				ErrMsg:  err.Error(),
			},
		})
		return
	}
	daemon.SendResponse(w, &opApi.DisruptNodeResponse{
		Node:  req.Node,
		Error: apiErrors.Error{},
	})
}
func disruptBridge(w http.ResponseWriter, r *http.Request) {

	req := &opApi.DisruptBridgeRequest{}

	if err := daemon.ParseRequest(r, req); err != nil {
		daemonLog.Println("disruptBridge:", err)
		daemon.SendError(w, &opApi.DisruptBridgeResponse{
			Bridge: req.Bridge,
			Error: apiErrors.Error{
				ErrCode: 1,
				ErrMsg:  err.Error(),
			},
		})
		return
	}

	err := engine.app.DisruptBridge(req.Bridge)

	if err != nil {
		daemonLog.Println("disruptBridge:", err)
		daemon.SendError(w, &opApi.DisruptBridgeResponse{
			Bridge: req.Bridge,
			Error: apiErrors.Error{
				ErrCode: 1,
				ErrMsg:  err.Error(),
			},
		})
		return
	}
	daemon.SendResponse(w, &opApi.DisruptBridgeResponse{
		Bridge: req.Bridge,
		Error:  apiErrors.Error{},
	})
}
func disruptRouters(w http.ResponseWriter, r *http.Request) {

	req := &opApi.DisruptRoutersRequest{}

	if err := daemon.ParseRequest(r, req); err != nil {
		daemonLog.Println("disruptRouters:", err)
		daemon.SendError(w, &opApi.DisruptRoutersResponse{
			Router1: req.Router1,
			Router2: req.Router2,
			Error: apiErrors.Error{
				ErrCode: 1,
				ErrMsg:  err.Error(),
			},
		})
		return
	}

	err := engine.app.DisruptRouters(req.Router1, req.Router2)

	if err != nil {
		daemonLog.Println("disruptRouters:", err)
		daemon.SendError(w, &opApi.DisruptRoutersResponse{
			Router1: req.Router1,
			Router2: req.Router2,
			Error: apiErrors.Error{
				ErrCode: 1,
				ErrMsg:  err.Error(),
			},
		})
		return
	}
	daemon.SendResponse(w, &opApi.DisruptRoutersResponse{
		Router1: req.Router1,
		Router2: req.Router2,
		Error:   apiErrors.Error{},
	})
}
func stopDisruptNode(w http.ResponseWriter, r *http.Request) {

	req := &opApi.DisruptNodeRequest{}
	if err := daemon.ParseRequest(r, req); err != nil {
		daemonLog.Println("stopDisruptNode:", err)
		daemon.SendError(w, &opApi.DisruptNodeResponse{
			Node: req.Node,
			Error: apiErrors.Error{
				ErrCode: 1, ErrMsg: err.Error(),
			},
		})
		return
	}

	err := engine.app.StopDisruptNode(req.Node)

	if err != nil {
		daemonLog.Println("stopDisruptNode:", err)
		daemon.SendError(w, &opApi.DisruptNodeResponse{
			Node: req.Node,
			Error: apiErrors.Error{
				ErrCode: 1,
				ErrMsg:  err.Error(),
			},
		})
		return
	}
	daemon.SendResponse(w, &opApi.DisruptNodeResponse{
		Node:  req.Node,
		Error: apiErrors.Error{},
	})
}
func stopDisruptBridge(w http.ResponseWriter, r *http.Request) {

	req := &opApi.DisruptBridgeRequest{}

	if err := daemon.ParseRequest(r, req); err != nil {
		daemonLog.Println("stopDisruptBridge:", err)
		daemon.SendError(w, &opApi.DisruptBridgeResponse{
			Bridge: req.Bridge,
			Error: apiErrors.Error{
				ErrCode: 1,
				ErrMsg:  err.Error(),
			},
		})
		return
	}

	err := engine.app.StopDisruptBridge(req.Bridge)

	if err != nil {
		daemonLog.Println("stopDisruptBridge:", err)
		daemon.SendError(w, &opApi.DisruptBridgeResponse{
			Bridge: req.Bridge,
			Error: apiErrors.Error{
				ErrCode: 1,
				ErrMsg:  err.Error(),
			},
		})
		return
	}
	daemon.SendResponse(w, &opApi.DisruptBridgeResponse{
		Bridge: req.Bridge,
		Error:  apiErrors.Error{},
	})
}
func stopDisruptRouters(w http.ResponseWriter, r *http.Request) {

	req := &opApi.DisruptRoutersRequest{}

	if err := daemon.ParseRequest(r, req); err != nil {
		daemonLog.Println("stopDisruptRouters:", err)
		daemon.SendError(w, &opApi.DisruptRoutersResponse{
			Router1: req.Router1,
			Router2: req.Router2,
			Error: apiErrors.Error{
				ErrCode: 1,
				ErrMsg:  err.Error(),
			},
		})
		return
	}

	err := engine.app.StopDisruptRouters(req.Router1, req.Router2)

	if err != nil {
		daemonLog.Println("stopDisruptRouters:", err)
		daemon.SendError(w, &opApi.DisruptRoutersResponse{
			Router1: req.Router1,
			Router2: req.Router2,
			Error: apiErrors.Error{
				ErrCode: 1,
				ErrMsg:  err.Error(),
			},
		})
		return
	}
	daemon.SendResponse(w, &opApi.DisruptRoutersResponse{
		Router1: req.Router1,
		Router2: req.Router2,
		Error:   apiErrors.Error{},
	})
}
func startBridge(w http.ResponseWriter, r *http.Request) {

	req := &opApi.StartBridgeRequest{}

	if err := daemon.ParseRequest(r, req); err != nil {
		daemonLog.Println("startBridge:", err)
		daemon.SendError(w, &opApi.StartBridgeResponse{
			Bridge: req.Bridge,
			Error: apiErrors.Error{
				ErrCode: 1,
				ErrMsg:  err.Error(),
			},
		})
		return
	}

	err := engine.app.StartBridge(req.Bridge)

	if err != nil {
		daemonLog.Println("startBridge:", err)
		daemon.SendError(w, &opApi.StartBridgeResponse{
			Bridge: req.Bridge,
			Error: apiErrors.Error{
				ErrCode: 1,
				ErrMsg:  err.Error(),
			},
		})
		return
	}
	daemon.SendResponse(w, &opApi.StopBridgeResponse{
		Bridge: req.Bridge,
		Error:  apiErrors.Error{},
	})
}
func startRouter(w http.ResponseWriter, r *http.Request) {

	req := &opApi.StartRouterRequest{}

	if err := daemon.ParseRequest(r, req); err != nil {
		daemonLog.Println("startRouter:", err)
		daemon.SendError(w, &opApi.StartRouterResponse{
			Router: req.Router,
			Error: apiErrors.Error{
				ErrCode: 1,
				ErrMsg:  err.Error(),
			},
		})
		return
	}

	err := engine.app.StartRouter(req.Router)

	if err != nil {
		daemonLog.Println("startRouter:", err)
		daemon.SendError(w, &opApi.StartRouterResponse{
			Router: req.Router,
			Error: apiErrors.Error{
				ErrCode: 1,
				ErrMsg:  err.Error(),
			},
		})
		return
	}
	daemon.SendResponse(w, &opApi.StartRouterResponse{
		Router: req.Router,
		Error:  apiErrors.Error{},
	})
}
func stopBridge(w http.ResponseWriter, r *http.Request) {

	req := &opApi.StopBridgeRequest{}

	if err := daemon.ParseRequest(r, req); err != nil {
		daemonLog.Println("stopBridge:", err)
		daemon.SendError(w, &opApi.StopBridgeResponse{
			Bridge: req.Bridge,
			Error: apiErrors.Error{
				ErrCode: 1,
				ErrMsg:  err.Error(),
			},
		})
		return
	}

	err := engine.app.StopBridge(req.Bridge)

	if err != nil {
		daemonLog.Println("stopBridge:", err)
		daemon.SendError(w, &opApi.StopBridgeResponse{
			Bridge: req.Bridge,
			Error: apiErrors.Error{
				ErrCode: 1,
				ErrMsg:  err.Error(),
			},
		})
		return
	}
	daemon.SendResponse(w, &opApi.DisruptBridgeResponse{
		Bridge: req.Bridge,
		Error:  apiErrors.Error{},
	})
}
func stopRouter(w http.ResponseWriter, r *http.Request) {

	req := &opApi.StopRouterRequest{}

	if err := daemon.ParseRequest(r, req); err != nil {
		daemonLog.Println("stopRouter:", err)
		daemon.SendError(w, &opApi.StopRouterResponse{
			Router: req.Router,
			Error: apiErrors.Error{
				ErrCode: 1,
				ErrMsg:  err.Error(),
			},
		})
		return
	}

	err := engine.app.StopRouter(req.Router)

	if err != nil {
		daemonLog.Println("stopRouter:", err)
		daemon.SendError(w, &opApi.StopRouterResponse{
			Router: req.Router,
			Error: apiErrors.Error{
				ErrCode: 1,
				ErrMsg:  err.Error(),
			},
		})
		return
	}
	daemon.SendResponse(w, &opApi.StopRouterResponse{
		Router: req.Router,
		Error:  apiErrors.Error{},
	})
}

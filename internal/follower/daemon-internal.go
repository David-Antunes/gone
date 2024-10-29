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

func trade(w http.ResponseWriter, r *http.Request) {

	req := &api.TradeRoutesRequest{}

	if err := daemon.ParseRequest(r, req); err != nil {
		fmt.Println("Invalid fields in connectRouterToRouter request.")
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
}

func routerWeights(w http.ResponseWriter, r *http.Request) {

	req := &api.GetRouterWeightsRequest{}

	if err := daemon.ParseRequest(r, req); err != nil {
		fmt.Println("Invalid fields in connectRouterToRouter request.")
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
		Id:        id,
		Node:      req.Node,
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

func stopSniffNode(w http.ResponseWriter, r *http.Request) {

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

	err := engine.app.StopSniffNode(req.Id)

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
func stopSniffBridge(w http.ResponseWriter, r *http.Request) {

	req := &opApi.StopSniffRequest{}

	if err := daemon.ParseRequest(r, req); err != nil {
		daemonLog.Println("stopSniffBridge:", err)
		daemon.SendError(w, &opApi.StopSniffResponse{
			Id: req.Id,
			Error: apiErrors.Error{
				ErrCode: 1,
				ErrMsg:  err.Error(),
			},
		})
		return
	}

	err := engine.app.StopSniffBridge(req.Id)

	if err != nil {
		daemonLog.Println("stopSniffBridge:", err)
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
func stopSniffRouters(w http.ResponseWriter, r *http.Request) {

	req := &opApi.StopSniffRequest{}

	if err := daemon.ParseRequest(r, req); err != nil {
		daemonLog.Println("stopSniffRouters:", err)
		daemon.SendError(w, &opApi.StopSniffResponse{
			Id: req.Id,
			Error: apiErrors.Error{
				ErrCode: 1,
				ErrMsg:  err.Error(),
			},
		})
		return
	}

	err := engine.app.StopSniffRouters(req.Id)

	if err != nil {
		daemonLog.Println("stopSniffRouters:", err)
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
			Id: req.Name,
			Error: apiErrors.Error{
				ErrCode: 1,
				ErrMsg:  err.Error(),
			},
		})
		return
	}

	id, path, machineId, err := engine.app.InterceptNode(req.Name, req.Direction)

	if err != nil {
		daemonLog.Println("interceptNode:", err)
		daemon.SendError(w, &opApi.InterceptNodeResponse{
			Id: req.Name,
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
			Id: req.Name,
			Error: apiErrors.Error{
				ErrCode: 1,
				ErrMsg:  err.Error(),
			},
		})
		return
	}

	id, path, machineId, err := engine.app.InterceptBridge(req.Name, req.Direction)

	if err != nil {
		daemonLog.Println("interceptBridge:", err)
		daemon.SendError(w, &opApi.InterceptBridgeResponse{
			Id: req.Name,
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
			Id: "",
			Error: apiErrors.Error{
				ErrCode: 1,
				ErrMsg:  err.Error(),
			},
		})
		return
	}

	id, path, machineId, err := engine.app.InterceptRouters(req.Router1, req.Router2, req.Direction)

	if err != nil {
		daemonLog.Println("interceptRouters:", err)
		daemon.SendError(w, &opApi.InterceptRoutersResponse{
			Id: "",
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

func stopInterceptNode(w http.ResponseWriter, r *http.Request) {

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

	err := engine.app.StopInterceptNode(req.Id, req.Direction)

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

func stopInterceptBridge(w http.ResponseWriter, r *http.Request) {

	req := &opApi.StopInterceptRequest{}

	if err := daemon.ParseRequest(r, req); err != nil {
		daemonLog.Println("stopInterceptBridge:", err)
		daemon.SendError(w, &opApi.StopInterceptResponse{
			Id: req.Id,
			Error: apiErrors.Error{
				ErrCode: 1,
				ErrMsg:  err.Error(),
			},
		})
		return
	}

	err := engine.app.StopInterceptBridge(req.Id, req.Direction)

	if err != nil {
		daemonLog.Println("stopInterceptBridge:", err)
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

func stopInterceptRouters(w http.ResponseWriter, r *http.Request) {

	req := &opApi.StopInterceptRequest{}

	if err := daemon.ParseRequest(r, req); err != nil {
		daemonLog.Println("stopInterceptRouters:", err)
		daemon.SendError(w, &opApi.StopInterceptResponse{
			Id: req.Id,
			Error: apiErrors.Error{
				ErrCode: 1,
				ErrMsg:  err.Error(),
			},
		})
		return
	}

	err := engine.app.StopInterceptRouters(req.Id, req.Direction)

	if err != nil {
		daemonLog.Println("stopInterceptRouters:", err)
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

	daemon.SendResponse(w, &opApi.ListSniffersResponse{
		Sniffers: ids,
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

package api

import api "github.com/David-Antunes/gone/api/Errors"

type InspectBridgeResponse struct {
	Name      string       `json:"name"`
	RouterId  string       `json:"router"`
	Nodes     []BridgeNode `json:"nodes"`
	MachineId string       `json:"machineId"`
	Error     api.Error    `json:"err"`
}

type BridgeNode struct {
	Id  string `json:"id"`
	Mac string `json:"mac"`
}

package Operations

import api "github.com/David-Antunes/gone/api/Errors"

type InterceptBridgeResponse struct {
	Id        string    `json:"id"`
	Bridge    string    `json:"bridge"`
	Path      string    `json:"path"`
	MachineId string    `json:"machineId"`
	Error     api.Error `json:"errors"`
}

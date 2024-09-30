package api

import api "github.com/David-Antunes/gone/api/Errors"

type AddRouterResponse struct {
	Name      string    `json:"name"`
	MachineId string    `json:"machineId"`
	Error     api.Error `json:"err"`
}

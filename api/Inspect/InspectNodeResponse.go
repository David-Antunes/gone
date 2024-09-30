package api

import api "github.com/David-Antunes/gone/api/Errors"

type InspectNodeResponse struct {
	Name      string    `json:"name"`
	Bridge    string    `json:"bridge"`
	Router    string    `json:"router"`
	MachineId string    `json:"machineId"`
	Error     api.Error `json:"err"`
}

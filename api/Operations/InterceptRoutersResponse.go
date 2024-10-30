package Operations

import api "github.com/David-Antunes/gone/api/Errors"

type InterceptRoutersResponse struct {
	Router1   string    `json:"router1"`
	Router2   string    `json:"router2"`
	Id        string    `json:"id"`
	Path      string    `json:"path"`
	MachineId string    `json:"machineId"`
	Error     api.Error `json:"error"`
}

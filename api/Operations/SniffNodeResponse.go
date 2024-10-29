package Operations

import api "github.com/David-Antunes/gone/api/Errors"

type SniffNodeResponse struct {
	Node      string    `json:"node"`
	Id        string    `json:"id"`
	Path      string    `json:"path"`
	MachineId string    `json:"machineId"`
	Error     api.Error `json:"error"`
}

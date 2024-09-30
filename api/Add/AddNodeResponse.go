package api

import api "github.com/David-Antunes/gone/api/Errors"

type AddNodeResponse struct {
	Id        string    `json:"name"`
	Mac       string    `json:"mac"`
	Ip        string    `json:"ip"`
	MachineId string    `json:"machineId"`
	Error     api.Error `json:"err"`
}

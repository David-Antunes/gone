package api

import api "github.com/David-Antunes/gone/api/Errors"

type RegisterNodeResponse struct {
	Id        string    `json:"id"`
	Ip        string    `json:"ip"`
	Mac       string    `json:"mac"`
	MachineId string    `json:"machineId"`
	Error     api.Error `json:"error"`
}

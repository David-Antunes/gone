package api

type RegisterNodeRequest struct {
	Id        string `json:"id"`
	Ip        string `json:"ip"`
	Mac       string `json:"mac"`
	MachineId string `json:"machineId"`
}

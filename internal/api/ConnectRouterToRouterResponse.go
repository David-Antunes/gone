package api

import (
	api "github.com/David-Antunes/gone/api/Errors"
)

type ConnectRouterToRouterResponse struct {
	R1        string    `json:"r1"`
	R2        string    `json:"r2"`
	MachineID string    `json:"machineID"`
	Error     api.Error `json:"err"`
}

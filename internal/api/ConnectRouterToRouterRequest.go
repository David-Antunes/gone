package api

import "time"

type ConnectRouterToRouterRequest struct {
	R1        string        `json:"r1"`
	R2        string        `json:"r2"`
	MachineID string        `json:"machineID"`
	Latency   time.Duration `json:"latency"`
	Jitter    float64       `json:"jitter"`
	DropRate  float64       `json:"dropRate"`
	Bandwidth int           `json:"bandwidth"`
	Weight    int           `json:"weight"`
}

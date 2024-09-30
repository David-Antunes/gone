package api

import api "github.com/David-Antunes/gone/api/Errors"

type InspectRouterResponse struct {
	Name      string              `json:"name"`
	MachineId string              `json:"machineId"`
	Bridges   []string            `json:"bridges"`
	Nodes     map[string][]string `json:"nodes"`
	Routers   []string            `json:"routers"`
	Weights   map[string]string   `json:"weights"`
	Error     api.Error           `json:"err"`
}

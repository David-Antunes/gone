package api

type AddNodeRequest struct {
	DockerCmd []string `json:"dockerCmd"`
	MachineId string   `json:"machineId"`
}

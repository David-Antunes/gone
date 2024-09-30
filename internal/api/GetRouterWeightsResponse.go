package api

import (
	api "github.com/David-Antunes/gone/api/Errors"
	"github.com/David-Antunes/gone/internal/topology"
)

type GetRouterWeightsResponse struct {
	Weights map[string]topology.Weight `json:"weights"`
	Error   api.Error                  `json:"err"`
}

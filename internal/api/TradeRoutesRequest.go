package api

import (
	api "github.com/David-Antunes/gone/api/Errors"
	"github.com/David-Antunes/gone/internal/topology"
)

type TradeRoutesRequest struct {
	To      string                     `json:"to"`
	From    string                     `json:"from"`
	Weights map[string]topology.Weight `json:"weights"`
	Error   api.Error                  `json:"error"`
}

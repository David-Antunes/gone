package api

import (
	api "github.com/David-Antunes/gone/api/Errors"
)

type TradeRoutesResponse struct {
	To    string    `json:"to"`
	From  string    `json:"from"`
	Error api.Error `json:"error"`
}

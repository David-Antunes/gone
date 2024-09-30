package api

import api "github.com/David-Antunes/gone/api/Errors"

type ConnectRouterToRouterResponse struct {
	From  string    `json:"from"`
	To    string    `json:"to"`
	Error api.Error `json:"err"`
}

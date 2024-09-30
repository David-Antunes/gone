package api

import api "github.com/David-Antunes/gone/api/Errors"

type ConnectNodeToBridgeResponse struct {
	Node   string    `json:"node"`
	Bridge string    `json:"bridge"`
	Error  api.Error `json:"err"`
}

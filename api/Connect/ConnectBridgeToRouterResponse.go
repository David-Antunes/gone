package api

import api "github.com/David-Antunes/gone/api/Errors"

type ConnectBridgeToRouterResponse struct {
	Bridge string    `json:"bridge"`
	Router string    `json:"router"`
	Error  api.Error `json:"err"`
}

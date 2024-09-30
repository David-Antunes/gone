package Disconnect

import api "github.com/David-Antunes/gone/api/Errors"

type DisconnectBridgeResponse struct {
	Name  string    `json:"name"`
	Error api.Error `json:"err"`
}

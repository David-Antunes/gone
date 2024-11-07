package Operations

import api "github.com/David-Antunes/gone/api/Errors"

type StopBridgeResponse struct {
	Bridge string    `json:"bridge"`
	Error  api.Error `json:"error"`
}

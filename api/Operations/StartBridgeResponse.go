package Operations

import apiErrors "github.com/David-Antunes/gone/api/Errors"

type StartBridgeResponse struct {
	Bridge string          `json:"bridge"`
	Error  apiErrors.Error `json:"error"`
}

package Operations

import apiErrors "github.com/David-Antunes/gone/api/Errors"

type DisruptBridgeResponse struct {
	Bridge string          `json:"bridge"`
	Error  apiErrors.Error `json:"error"`
}

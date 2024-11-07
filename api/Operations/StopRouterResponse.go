package Operations

import apiErrors "github.com/David-Antunes/gone/api/Errors"

type StopRouterResponse struct {
	Router string          `json:"router"`
	Error  apiErrors.Error `json:"error"`
}

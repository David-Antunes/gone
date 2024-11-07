package Operations

import apiErrors "github.com/David-Antunes/gone/api/Errors"

type DisruptRoutersResponse struct {
	Router1 string          `json:"router1"`
	Router2 string          `json:"router2"`
	Error   apiErrors.Error `json:"error"`
}

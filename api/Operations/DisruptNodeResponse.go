package Operations

import apiErrors "github.com/David-Antunes/gone/api/Errors"

type DisruptNodeResponse struct {
	Node  string          `json:"node"`
	Error apiErrors.Error `json:"error"`
}

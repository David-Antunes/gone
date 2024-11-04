package api

import (
	"github.com/David-Antunes/gone/api"
	apiErrors "github.com/David-Antunes/gone/api/Errors"
)

type InspectNodeResponse struct {
	Node  api.Node        `json:"node"`
	Error apiErrors.Error `json:"err"`
}

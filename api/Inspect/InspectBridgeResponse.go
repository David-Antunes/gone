package api

import (
	"github.com/David-Antunes/gone/api"
	apiErrors "github.com/David-Antunes/gone/api/Errors"
)

type InspectBridgeResponse struct {
	Bridge api.Bridge      `json:"bridge"`
	Error  apiErrors.Error `json:"err"`
}

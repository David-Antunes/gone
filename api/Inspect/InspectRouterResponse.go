package api

import (
	"github.com/David-Antunes/gone/api"
	apiErrors "github.com/David-Antunes/gone/api/Errors"
)

type InspectRouterResponse struct {
	Router api.Router      `json:"router"`
	Error  apiErrors.Error `json:"err"`
}

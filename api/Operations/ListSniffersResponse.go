package Operations

import (
	"github.com/David-Antunes/gone/api"
	apiErrors "github.com/David-Antunes/gone/api/Errors"
)

type ListSniffersResponse struct {
	Sniffers []api.SniffComponent `json:"sniffers"`
	Error    apiErrors.Error      `json:"error"`
}

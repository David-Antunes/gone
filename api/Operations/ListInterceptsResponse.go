package Operations

import (
	"github.com/David-Antunes/gone/api"
	apiErrors "github.com/David-Antunes/gone/api/Errors"
)

type ListInterceptsResponse struct {
	Intercepts []api.InterceptComponent `json:"intercepts"`
	Error      apiErrors.Error          `json:"error"`
}

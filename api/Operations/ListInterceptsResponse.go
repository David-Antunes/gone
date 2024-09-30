package Operations

import api "github.com/David-Antunes/gone/api/Errors"

type ListInterceptsResponse struct {
	Intercepts []string  `json:"intercepts"`
	Error      api.Error `json:"error"`
}

package Operations

import api "github.com/David-Antunes/gone/api/Errors"

type PropagateResponse struct {
	Name  string    `json:"name"`
	Error api.Error `json:"errors"`
}

package Operations

import api "github.com/David-Antunes/gone/api/Errors"

type UnpauseResponse struct {
	Id    string    `json:"id"`
	All   bool      `json:"all"`
	Error api.Error `json:"error"`
}

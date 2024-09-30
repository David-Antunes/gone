package Operations

import api "github.com/David-Antunes/gone/api/Errors"

type StopSniffResponse struct {
	Id    string    `json:"id"`
	Error api.Error `json:"error"`
}

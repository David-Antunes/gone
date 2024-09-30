package api

import api "github.com/David-Antunes/gone/api/Errors"

type ClearNodeResponse struct {
	Id    string    `json:"id"`
	Error api.Error `json:"error"`
}

package api

import api "github.com/David-Antunes/gone/api/Errors"

type RemoveRouterResponse struct {
	Name  string    `json:"name"`
	Error api.Error `json:"err"`
}

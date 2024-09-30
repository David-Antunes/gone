package api

import api "github.com/David-Antunes/gone/api/Errors"

type InspectLinkResponse struct {
	Name  string    `json:"name"`
	Error api.Error `json:"err"`
}

package Operations

import api "github.com/David-Antunes/gone/api/Errors"

type ForgetResponse struct {
	Name  string    `json:"name"`
	Error api.Error `json:"error"`
}

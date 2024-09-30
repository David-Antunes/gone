package Operations

import api "github.com/David-Antunes/gone/api/Errors"

type ListSniffersResponse struct {
	Sniffers []string  `json:"sniffers"`
	Error    api.Error `json:"error"`
}

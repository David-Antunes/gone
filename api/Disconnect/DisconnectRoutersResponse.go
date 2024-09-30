package Disconnect

import api "github.com/David-Antunes/gone/api/Errors"

type DisconnectRoutersResponse struct {
	First  string    `json:"First"`
	Second string    `json:"Second"`
	Error  api.Error `json:"err"`
}

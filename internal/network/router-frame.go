package network

import "github.com/David-Antunes/gone-proxy/xdp"

type RouterFrame struct {
	To    string     `json:"to"`
	From  string     `json:"from"`
	Frame *xdp.Frame `json:"frame"`
}

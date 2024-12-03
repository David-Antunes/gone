package api

type ConnectRouterToRouterRequest struct {
	From      string  `json:"from"`
	To        string  `json:"to"`
	Latency   float64 `json:"latency"`
	Jitter    float64 `json:"jitter"`
	DropRate  float64 `json:"dropRate"`
	Bandwidth int     `json:"bandwidth"`
	Weight    int     `json:"weight"`
}

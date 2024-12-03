package api

type ConnectBridgeToRouterRequest struct {
	Bridge    string  `json:"bridge"`
	Router    string  `json:"router"`
	Latency   float64 `json:"latency"`
	Jitter    float64 `json:"jitter"`
	DropRate  float64 `json:"dropRate"`
	Bandwidth int     `json:"bandwidth"`
	Weight    int     `json:"weight"`
}

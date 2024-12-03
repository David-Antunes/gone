package api

type ConnectNodeToBridgeRequest struct {
	Node      string  `json:"node"`
	Bridge    string  `json:"bridge"`
	Latency   float64 `json:"latency"`
	Jitter    float64 `json:"jitter"`
	DropRate  float64 `json:"dropRate"`
	Bandwidth int     `json:"bandwidth"`
	Weight    int     `json:"weight"`
}

package Operations

type InterceptRoutersRequest struct {
	Router1   string `json:"router1"`
	Router2   string `json:"router2"`
	Direction bool   `json:"direction"`
}

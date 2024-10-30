package Operations

type InterceptRoutersRequest struct {
	Router1   string `json:"router1"`
	Router2   string `json:"router2"`
	Id        string `json:"id"`
	Direction bool   `json:"direction"`
}

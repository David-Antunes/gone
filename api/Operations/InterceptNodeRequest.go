package Operations

type InterceptNodeRequest struct {
	Node      string `json:"node"`
	Id        string `json:"Id"`
	Direction bool   `json:"direction"`
}

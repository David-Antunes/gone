package Operations

type InterceptBridgeRequest struct {
	Bridge    string `json:"bridge"`
	Id        string `json:"id"`
	Direction bool   `json:"direction"`
}

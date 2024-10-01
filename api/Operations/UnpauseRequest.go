package Operations

type UnpauseRequest struct {
	Id  string `json:"id"`
	All bool   `json:"all"`
}

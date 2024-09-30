package topology

import "github.com/David-Antunes/gone/internal/network"

type Link struct {
	Id          string
	NetworkLink *network.Link
	From        Component
	To          Component
}

func (link *Link) ID() string {
	return link.Id
}

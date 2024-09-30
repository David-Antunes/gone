package topology

import "github.com/David-Antunes/gone/internal/network"

type BiLink struct {
	Id            string
	NetworkBILink *network.BiLink
	ConnectsTo    *Link
	ConnectsFrom  *Link
	To            Component
	From          Component
}

func (link *BiLink) ID() string {
	return link.Id
}

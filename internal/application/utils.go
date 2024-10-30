package application

import (
	"github.com/David-Antunes/gone/internal/network"
	"github.com/David-Antunes/gone/internal/topology"
)

const _REMOTE_QUEUESIZE = 1000

func sniffSocketPath(id string) string {
	return "/tmp/" + id + ".sniff"
}
func interceptSocketPath(id string) string {
	return "/tmp/" + id + ".intercept"
}

func getInterceptId(id string, direction bool) string {
	if direction {
		return id + "-tx"
	} else {
		return id + "-rx"
	}
}

func isSpecialLink(link *topology.Link) bool {
	if link.NetworkLink != nil {
		_, ok := link.NetworkLink.GetShaper().(*network.NetworkShaper)
		return ok
	}
	return true
}

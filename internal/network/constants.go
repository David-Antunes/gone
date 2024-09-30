package network

import "github.com/David-Antunes/gone-proxy/xdp"

const queueSize = 10000000
const packetSize = 1500

var nullChan = make(chan *xdp.Frame, queueSize)
var outChan chan *xdp.Frame

func GetNullChan() chan *xdp.Frame {
	return nullChan
}

func ClearNullChan() {
	for {
		for range nullChan {
			continue
		}
	}
}
func SetOutChan(channel chan *xdp.Frame) {
	outChan = channel
}

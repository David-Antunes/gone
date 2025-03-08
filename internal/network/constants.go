package network

import "github.com/David-Antunes/gone-proxy/xdp"

const queueSize = 3000
const packetSize = 1500

var nullChan = make(chan *xdp.Frame, queueSize)

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

var broadcastAddr = []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff}

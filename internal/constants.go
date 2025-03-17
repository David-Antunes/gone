package internal

import (
	"fmt"
	"github.com/David-Antunes/gone-proxy/xdp"
	"strconv"
	"strings"
)

const RemoteQueueSize = 10000

const QueueSize = 5000
const ComponentQueueSize = 10000
const PacketSize = 1500

const ProxyQueueSize = 100000

var nullChan = make(chan *xdp.Frame, QueueSize)

func GetNullChan() chan *xdp.Frame {
	return nullChan
}

var LocalQuery = false

func ClearNullChan() {
	for {
		for range nullChan {
			continue
		}
	}
}

var BroadcastAddr = []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff}

func ConvertMacStringToBytes(macAddr string) []byte {
	parts := strings.Split(macAddr, ":")
	var macBytes []byte

	for _, part := range parts {
		b, err := strconv.ParseUint(part, 16, 8)
		if err != nil {
			fmt.Printf("Error parsing %s: %v\n", part, err)
			panic(err)
		}
		macBytes = append(macBytes, byte(b))

	}
	return macBytes
}

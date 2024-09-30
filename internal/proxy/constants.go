package proxy

import (
	"fmt"
	"strconv"
	"strings"
)

const (
	queueSize = 1000000
)

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

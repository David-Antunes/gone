package network

import (
	"math/rand"
	"time"
)

type LinkProps struct {
	Latency   time.Duration
	FLatency  float64
	Bandwidth int
	Jitter    float64
	DropRate  float64
	Weight    int
}

func (props *LinkProps) PollJitter() time.Duration {
	if props.Jitter == 0 {
		return 0
	} else {
		return time.Duration((rand.NormFloat64() * props.Jitter) / 3 * float64(time.Millisecond))
	}
}
func (props *LinkProps) PollDropRate() bool {
	return rand.Float64() < props.DropRate
}

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
		return props.Latency
	} else {
		random := rand.NormFloat64()
		results := (random * float64(props.Latency)) + float64(time.Duration(props.Jitter)*time.Millisecond)
		duration := time.Duration(results)

		//fmt.Println(random, results, duration)
		return duration
	}
}
func (props *LinkProps) PollDropRate() bool {
	return rand.Float64() < props.DropRate
}

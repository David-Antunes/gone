package network

import "time"

type Shaper interface {
	Start()
	SetDelay(delay *Delay)
	GetDelay() *Delay
	Disrupt() bool
	StopDisrupt() bool
	Stop()
	Close()
	Pause()
	Unpause()
}

type DynamicDelay struct {
	ReceiveDelay  *Delay
	TransmitDelay *Delay
}
type Delay struct {
	Value time.Duration
}

func (d *DynamicDelay) GetReceiveLatency() time.Duration {
	return d.ReceiveDelay.Value
}
func (d *DynamicDelay) GetTransmitLatency() time.Duration {
	return d.TransmitDelay.Value
}

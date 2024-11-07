package network

import "github.com/David-Antunes/gone-proxy/xdp"

type NullShaper struct {
	running  bool
	incoming chan *xdp.Frame
	delay    *Delay
	ctx      chan struct{}
}

func (shaper *NullShaper) GetProps() LinkProps {
	return LinkProps{}
}

func (shaper *NullShaper) GetIncoming() chan *xdp.Frame {
	return shaper.incoming
}

func (shaper *NullShaper) GetOutgoing() chan *xdp.Frame {
	return nil
}

func (shaper *NullShaper) SetDelay(delay *Delay) {
	shaper.delay = delay
}
func (shaper *NullShaper) GetDelay() *Delay {
	return shaper.delay
}
func CreateNullShaper(incoming chan *xdp.Frame) Shaper {
	return &NullShaper{
		running:  false,
		incoming: incoming,
		delay:    &Delay{0},
		ctx:      make(chan struct{}, 1),
	}
}

func (shaper *NullShaper) Stop() {
	if shaper.running {
		shaper.running = false
		shaper.ctx <- struct{}{}
	}

}

func (shaper *NullShaper) Start() {
	if !shaper.running {
		shaper.running = true
		go shaper.receive()
	}
}

func (shaper *NullShaper) receive() {

	for {
		select {
		case <-shaper.ctx:
			return
		case <-shaper.incoming:
		}
	}
}

func (shaper *NullShaper) Disrupt() bool {
	return false
}

func (shaper *NullShaper) StopDisrupt() bool {
	return true
}

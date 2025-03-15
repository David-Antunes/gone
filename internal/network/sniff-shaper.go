package network

import (
	"fmt"
	"github.com/David-Antunes/gone-proxy/xdp"
	"github.com/David-Antunes/gone/internal"
	redirect_traffic "github.com/David-Antunes/gone/internal/redirect-traffic"
	"golang.org/x/time/rate"
	"time"
)

type SniffShaper struct {
	running   bool
	queue     chan *xdp.Frame
	incoming  chan *xdp.Frame
	outgoing  chan *xdp.Frame
	props     LinkProps
	delay     *Delay
	limiter   *rate.Limiter
	tokenSize int
	ctx       chan struct{}
	rt        *redirect_traffic.SniffComponent
	disrupted disruptLogic
}

func (shaper *SniffShaper) GetProps() LinkProps {
	return shaper.props
}

func (shaper *SniffShaper) GetIncoming() chan *xdp.Frame {
	return shaper.incoming
}

func (shaper *SniffShaper) GetOutgoing() chan *xdp.Frame {
	return shaper.outgoing
}

func (shaper *SniffShaper) SetDelay(delay *Delay) {
	shaper.delay = delay
}
func (shaper *SniffShaper) GetDelay() *Delay {
	return shaper.delay
}

func (shaper *SniffShaper) GetRtID() string {
	return shaper.rt.Id
}

func NewSniffShaper(incoming chan *xdp.Frame, outgoing chan *xdp.Frame, props LinkProps, rt *redirect_traffic.SniffComponent) Shaper {
	aux := float64(internal.PacketSize / props.Bandwidth)
	newTime := float64(time.Second) * aux
	return &SniffShaper{
		running:   false,
		queue:     make(chan *xdp.Frame, internal.QueueSize),
		incoming:  incoming,
		outgoing:  outgoing,
		props:     props,
		delay:     &Delay{0},
		limiter:   rate.NewLimiter(rate.Every(time.Duration(newTime)), 1),
		tokenSize: internal.PacketSize,
		ctx:       make(chan struct{}, 2),
		rt:        rt,
		disrupted: disruptLogic{
			disrupted: false,
			ctx:       make(chan struct{}, 1),
		},
	}
}

func (shaper *SniffShaper) Stop() {
	if shaper.running {
		shaper.running = false
		shaper.ctx <- struct{}{}
		shaper.ctx <- struct{}{}
	}
}

func (shaper *SniffShaper) Start() {
	if !shaper.running {
		shaper.running = true
		go shaper.receive()
		go shaper.send()
	}
}

func (shaper *SniffShaper) Disrupt() bool {
	if !shaper.disrupted.disrupted {
		shaper.disrupted.disrupted = true
		shaper.Stop()
		go shaper.null()

		// Clear queue for requests
		go func() {
			go shaper.send()
			time.Sleep(time.Second)
			shaper.ctx <- struct{}{}
		}()
		return true
	} else {
		return false
	}
}

func (shaper *SniffShaper) null() {
	for {
		select {
		case <-shaper.disrupted.ctx:
			return
		case <-shaper.incoming:
			continue
		}
	}
}

func (shaper *SniffShaper) StopDisrupt() bool {

	if shaper.disrupted.disrupted {
		shaper.disrupted.disrupted = false
		shaper.disrupted.ctx <- struct{}{}
		shaper.Start()
		return true
	}
	return false
}

func (shaper *SniffShaper) receive() {

	for {

		select {
		case <-shaper.ctx:
			return
		case <-shaper.rt.Socket.GetIncoming():
			continue
		case frame := <-shaper.incoming:
			if shaper.props.PollDropRate() {
				continue
			}
			shaper.rt.Socket.GetOutgoing() <- frame
			frame.Time = frame.Time.Add(shaper.props.Latency)
			frame.Time = frame.Time.Add(shaper.props.PollJitter())
			frame.Time = frame.Time.Add(-shaper.delay.Value)
			if len(shaper.queue) < internal.QueueSize {
				shaper.queue <- frame
			}
		}
	}
}

func (shaper *SniffShaper) send() {

	for {
		select {
		case <-shaper.ctx:
			return

		case frame := <-shaper.queue:
			var r *rate.Reservation
			if shaper.tokenSize < frame.FrameSize {
				r = shaper.limiter.Reserve()
				if !r.OK() {
					fmt.Println("Something went wrong")
				}
				shaper.tokenSize = shaper.tokenSize - frame.FrameSize + internal.PacketSize
				time.Sleep(r.Delay())
			} else {
				shaper.tokenSize = shaper.tokenSize - frame.FrameSize
			}

			go func() {
				time.Sleep(time.Until(frame.Time))
				if len(shaper.outgoing) < internal.QueueSize {
					shaper.outgoing <- frame
				}
			}()
		}
	}
}

func (shaper *SniffShaper) ConvertToNetworkShaper() *NetworkShaper {
	if shaper.StopDisrupt() {
		shaper.Stop()
	}
	return &NetworkShaper{
		running:   shaper.running,
		queue:     shaper.queue,
		incoming:  shaper.incoming,
		outgoing:  shaper.outgoing,
		props:     shaper.props,
		limiter:   shaper.limiter,
		delay:     shaper.delay,
		tokenSize: shaper.tokenSize,
		ctx:       shaper.ctx,
	}
}
func (shaper *SniffShaper) Close() {
	shaper.Stop()
	shaper.StopDisrupt()
}
func (shaper *SniffShaper) Pause() {
	if shaper.running {
		shaper.ctx <- struct{}{}
		shaper.ctx <- struct{}{}
	} else if shaper.disrupted.disrupted {
		shaper.disrupted.ctx <- struct{}{}
	}
}

func (shaper *SniffShaper) Unpause() {
	if shaper.running {
		go shaper.receive()
		go shaper.send()
	} else if shaper.disrupted.disrupted {
		go shaper.null()
	}
}

package network

import (
	"fmt"
	"github.com/David-Antunes/gone-proxy/xdp"
	redirect_traffic "github.com/David-Antunes/gone/internal/redirect-traffic"
	"golang.org/x/time/rate"
	"time"
)

type InterceptShaper struct {
	running   bool
	queue     chan *xdp.Frame
	incoming  chan *xdp.Frame
	outgoing  chan *xdp.Frame
	props     LinkProps
	delay     *Delay
	limiter   *rate.Limiter
	tokenSize int
	ctx       chan struct{}
	rt        *redirect_traffic.InterceptComponent
	disrupted disruptLogic
}

func (shaper *InterceptShaper) GetProps() LinkProps {
	return shaper.props
}

func (shaper *InterceptShaper) GetIncoming() chan *xdp.Frame {
	return shaper.incoming
}

func (shaper *InterceptShaper) GetOutgoing() chan *xdp.Frame {
	return shaper.outgoing
}

func (shaper *InterceptShaper) SetDelay(delay *Delay) {
	shaper.delay = delay
}
func (shaper *InterceptShaper) GetDelay() *Delay {
	return shaper.delay
}

func (shaper *InterceptShaper) GetRtID() string {
	return shaper.rt.Id
}

func NewInterceptShaper(incoming chan *xdp.Frame, outgoing chan *xdp.Frame, props LinkProps, rt *redirect_traffic.InterceptComponent) Shaper {
	aux := float64(packetSize) / float64(props.Bandwidth)
	newTime := float64(time.Second) * aux

	return &InterceptShaper{
		running:   false,
		queue:     make(chan *xdp.Frame, queueSize),
		incoming:  incoming,
		outgoing:  outgoing,
		props:     props,
		delay:     &Delay{0},
		limiter:   rate.NewLimiter(rate.Every(time.Duration(newTime)), 1),
		tokenSize: packetSize,
		ctx:       make(chan struct{}, 2),
		rt:        rt,
		disrupted: disruptLogic{
			disrupted: false,
			ctx:       make(chan struct{}, 1),
		},
	}
}

func (shaper *InterceptShaper) Stop() {
	if shaper.running {
		shaper.running = false
		shaper.ctx <- struct{}{}
		shaper.ctx <- struct{}{}
	}
}

func (shaper *InterceptShaper) Start() {
	if !shaper.running {
		shaper.running = true
		go shaper.receive()
		go shaper.send()
	}
}

func (shaper *InterceptShaper) Disrupt() bool {
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

func (shaper *InterceptShaper) null() {
	for {
		select {
		case <-shaper.disrupted.ctx:
			return
		case <-shaper.incoming:
			continue
		}
	}
}

func (shaper *InterceptShaper) StopDisrupt() bool {

	if shaper.disrupted.disrupted {
		shaper.disrupted.disrupted = false
		shaper.disrupted.ctx <- struct{}{}
		shaper.Start()
		return true
	}
	return false
}

func (shaper *InterceptShaper) receive() {

	for {
		select {
		case <-shaper.ctx:
			return

		case frame := <-shaper.incoming:
			shaper.rt.Socket.GetOutgoing() <- frame

		case frame := <-shaper.rt.Socket.GetIncoming():

			frame.Time = frame.Time.Add(shaper.props.Latency)
			frame.Time = frame.Time.Add(shaper.props.PollJitter())
			frame.Time = frame.Time.Add(-shaper.delay.Value)
			if shaper.props.PollDropRate() {
				continue
			}
			if len(shaper.queue) < queueSize {
				shaper.queue <- frame
				//} else {
				//	fmt.Println("Queue Full!")
			}
		}
	}
}

func (shaper *InterceptShaper) send() {

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
				shaper.tokenSize = shaper.tokenSize - frame.FrameSize + packetSize
				frame.Time = frame.Time.Add(r.Delay())
			} else {
				shaper.tokenSize = shaper.tokenSize - frame.FrameSize
			}

			//go func() {
			time.Sleep(time.Until(frame.Time))
			if len(shaper.outgoing) < queueSize {
				shaper.outgoing <- frame
				//} else {
				//	fmt.Println("Queue Full!")
			}
			//}()

		}
	}
}

func (shaper *InterceptShaper) ConvertToNetworkShaper() *NetworkShaper {
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
func (shaper *InterceptShaper) Close() {
	shaper.Stop()
	shaper.StopDisrupt()
}

func (shaper *InterceptShaper) Pause() {
	if shaper.running {
		shaper.ctx <- struct{}{}
		shaper.ctx <- struct{}{}
	} else if shaper.disrupted.disrupted {
		shaper.disrupted.ctx <- struct{}{}
	}
}

func (shaper *InterceptShaper) Unpause() {
	if shaper.running {
		go shaper.receive()
		go shaper.send()
	} else if shaper.disrupted.disrupted {
		go shaper.null()
	}
}

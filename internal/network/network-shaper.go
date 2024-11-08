package network

import (
	"fmt"
	"github.com/David-Antunes/gone-proxy/xdp"
	redirect_traffic "github.com/David-Antunes/gone/internal/redirect-traffic"
	"time"

	"golang.org/x/time/rate"
)

type NetworkShaper struct {
	running   bool
	queue     chan *xdp.Frame
	incoming  chan *xdp.Frame
	outgoing  chan *xdp.Frame
	delay     *Delay
	props     LinkProps
	limiter   *rate.Limiter
	tokenSize int
	ctx       chan struct{}
	disrupted disruptLogic
}

func (shaper *NetworkShaper) GetProps() LinkProps {
	return shaper.props
}

func (shaper *NetworkShaper) GetIncoming() chan *xdp.Frame {
	return shaper.incoming
}

func (shaper *NetworkShaper) GetOutgoing() chan *xdp.Frame {
	return shaper.outgoing
}

func (shaper *NetworkShaper) GetDelay() *Delay {
	return shaper.delay
}

func (shaper *NetworkShaper) SetDelay(delay *Delay) {
	shaper.delay = delay
}

func CreateNetworkShaper(incoming chan *xdp.Frame, outgoing chan *xdp.Frame, props LinkProps) Shaper {
	aux := packetSize / float64(props.Bandwidth)
	newTime := float64(time.Second) * aux
	return &NetworkShaper{
		running:   false,
		queue:     make(chan *xdp.Frame, queueSize),
		incoming:  incoming,
		outgoing:  outgoing,
		delay:     &Delay{0},
		props:     props,
		limiter:   rate.NewLimiter(rate.Every(time.Duration(newTime)), 1),
		tokenSize: packetSize,
		ctx:       make(chan struct{}, 2),
		disrupted: disruptLogic{
			disrupted: false,
			ctx:       make(chan struct{}, 1),
		},
	}
}
func (shaper *NetworkShaper) Stop() {
	if shaper.running {
		shaper.running = false
		shaper.ctx <- struct{}{}
		shaper.ctx <- struct{}{}
	}
}

func (shaper *NetworkShaper) Start() {
	if !shaper.running {
		shaper.running = true
		go shaper.send()
		if shaper.props.Latency == 0 && shaper.props.Jitter == 0.0 && shaper.props.DropRate == 0.0 {
			go shaper.receiveNoLatency()
		} else {
			go shaper.receiveLatency()
		}
	}
}

func (shaper *NetworkShaper) Disrupt() bool {
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

func (shaper *NetworkShaper) null() {
	for {
		select {
		case <-shaper.disrupted.ctx:
			return
		case <-shaper.incoming:
			continue
		}
	}
}

func (shaper *NetworkShaper) StopDisrupt() bool {

	if shaper.disrupted.disrupted {
		shaper.disrupted.disrupted = false
		shaper.disrupted.ctx <- struct{}{}
		shaper.Start()
		return true
	}
	return false
}

func (shaper *NetworkShaper) receiveLatency() {

	for {

		select {
		case <-shaper.ctx:
			return

		case frame := <-shaper.incoming:
			//fmt.Println("before:", frame.Time)
			frame.Time = frame.Time.Add(shaper.props.Latency)
			frame.Time = frame.Time.Add(shaper.props.PollJitter())
			frame.Time = frame.Time.Add(-shaper.delay.Value)
			//fmt.Println("after:", frame.Time, shaper.props.Latency)
			if shaper.props.PollDropRate() {
				continue
			}
			if len(shaper.queue) < queueSize {
				shaper.queue <- frame
			} else {
				fmt.Println("Queue Full!")
			}
		}
	}
}
func (shaper *NetworkShaper) receiveNoLatency() {

	for {

		select {
		case <-shaper.ctx:
			return

		case frame := <-shaper.incoming:
			frame.Time = frame.Time.Add(-shaper.delay.Value)
			if len(shaper.queue) < queueSize {
				shaper.queue <- frame
			} else {
				fmt.Println("Queue Full!")
			}
		}
	}
}

func (shaper *NetworkShaper) send() {

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
			go func() {
				time.Sleep(time.Until(frame.Time))
				if len(shaper.outgoing) < queueSize {
					shaper.outgoing <- frame
				} else {
					fmt.Println("Queue Full!")
				}
			}()
		}
	}
}

func (shaper *NetworkShaper) ConvertToSniffShaper(rt *redirect_traffic.SniffComponent) *SniffShaper {
	if shaper.StopDisrupt() {
		shaper.Stop()
	}
	return &SniffShaper{
		running:   false,
		queue:     shaper.queue,
		incoming:  shaper.incoming,
		outgoing:  shaper.outgoing,
		props:     shaper.props,
		delay:     shaper.delay,
		limiter:   shaper.limiter,
		tokenSize: shaper.tokenSize,
		rt:        rt,
		ctx:       shaper.ctx,
		disrupted: shaper.disrupted,
	}
}

func (shaper *NetworkShaper) ConvertToInterceptShaper(rt *redirect_traffic.InterceptComponent) *InterceptShaper {
	if shaper.StopDisrupt() {
		shaper.Stop()
	}
	return &InterceptShaper{
		running:   false,
		queue:     shaper.queue,
		incoming:  shaper.incoming,
		outgoing:  shaper.outgoing,
		props:     shaper.props,
		delay:     shaper.delay,
		limiter:   shaper.limiter,
		tokenSize: shaper.tokenSize,
		rt:        rt,
		ctx:       shaper.ctx,
		disrupted: shaper.disrupted,
	}
}

func (shaper *NetworkShaper) Close() {
	shaper.Stop()
	shaper.StopDisrupt()
}

func (shaper *NetworkShaper) Pause() {
	if shaper.running {
		shaper.ctx <- struct{}{}
		shaper.ctx <- struct{}{}
	} else if shaper.disrupted.disrupted {
		shaper.disrupted.ctx <- struct{}{}
	}
}

func (shaper *NetworkShaper) Unpause() {
	if shaper.running {
		if shaper.props.Latency == 0 && shaper.props.Jitter == 0.0 && shaper.props.DropRate == 0.0 {
			go shaper.receiveNoLatency()
		} else {
			go shaper.receiveLatency()
		}
		go shaper.send()
	} else if shaper.disrupted.disrupted {
		go shaper.null()
	}
}

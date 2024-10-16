package network

import (
	"fmt"
	"github.com/David-Antunes/gone-proxy/xdp"
	"golang.org/x/time/rate"
	"time"
)

type RemoteShaper struct {
	running   bool
	queue     chan *xdp.Frame
	incoming  chan *xdp.Frame
	outgoing  chan *RouterFrame
	delay     *Delay
	props     LinkProps
	limiter   *rate.Limiter
	tokenSize int
	ctx       chan struct{}
	to        string
	from      string
}

func (shaper *RemoteShaper) GetProps() LinkProps {
	return shaper.props
}

func (shaper *RemoteShaper) GetIncoming() chan *xdp.Frame {
	return shaper.incoming
}

func (shaper *RemoteShaper) GetOutgoing() chan *xdp.Frame {
	return nil
}

func (shaper *RemoteShaper) SetDelay(delay *Delay) {
	shaper.delay = delay
}
func (shaper *RemoteShaper) GetDelay() *Delay {
	return shaper.delay
}

func CreateRemoteShaper(to string, from string, incoming chan *xdp.Frame, outgoing chan *RouterFrame, props LinkProps) Shaper {
	aux := float64(packetSize / props.Bandwidth)
	newTime := float64(time.Second) * aux
	return &RemoteShaper{
		running:   false,
		queue:     make(chan *xdp.Frame, queueSize),
		incoming:  incoming,
		outgoing:  outgoing,
		props:     props,
		limiter:   rate.NewLimiter(rate.Every(time.Duration(newTime)), 1),
		tokenSize: packetSize,
		delay:     &Delay{0},
		ctx:       make(chan struct{}),
		to:        to,
		from:      from,
	}
}
func (shaper *RemoteShaper) Stop() {
	if shaper.running {
		shaper.running = false
		shaper.ctx <- struct{}{}
		shaper.ctx <- struct{}{}
	}
}

func (shaper *RemoteShaper) Start() {
	if !shaper.running {
		shaper.running = true
		go shaper.receive()
		go shaper.send()
	}
}

func (shaper *RemoteShaper) receive() {

	for {

		select {
		case <-shaper.ctx:
			return

		case frame := <-shaper.incoming:
			frame.Time = frame.Time.Add(shaper.props.Latency)
			frame.Time = frame.Time.Add(shaper.props.PollJitter())
			frame.Time = frame.Time.Add(-shaper.delay.Value)
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

func (shaper *RemoteShaper) send() {

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
					shaper.outgoing <- &RouterFrame{
						To:    shaper.to,
						From:  shaper.from,
						Frame: frame,
					}
				} else {
					fmt.Println("Queue Full!")
				}
			}()
		}
	}
}

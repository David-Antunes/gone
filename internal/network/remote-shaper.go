package network

import (
	"fmt"
	"github.com/David-Antunes/gone-proxy/xdp"
	"github.com/David-Antunes/gone/internal"
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
	To        string
	From      string
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
	aux := float64(internal.PacketSize / props.Bandwidth)
	newTime := float64(time.Second) * aux

	return &RemoteShaper{
		running:   false,
		queue:     make(chan *xdp.Frame, internal.QueueSize),
		incoming:  incoming,
		outgoing:  outgoing,
		props:     props,
		limiter:   rate.NewLimiter(rate.Every(time.Duration(newTime)), 1),
		tokenSize: internal.PacketSize,
		delay:     &Delay{0},
		ctx:       make(chan struct{}),
		To:        to,
		From:      from,
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
			//if len(shaper.queue) < internal.QueueSize {
			shaper.queue <- frame
			//}
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
				shaper.tokenSize = shaper.tokenSize - frame.FrameSize + internal.PacketSize
				frame.Time = frame.Time.Add(r.Delay())
			} else {
				shaper.tokenSize = shaper.tokenSize - frame.FrameSize
			}
			//go func() {
			time.Sleep(time.Until(frame.Time))
			if len(shaper.outgoing) < internal.RemoteQueueSize {
				shaper.outgoing <- &RouterFrame{
					To:    shaper.To,
					From:  shaper.From,
					Frame: frame,
				}
			}
			//}()
		}
	}
}

func (shaper *RemoteShaper) Disrupt() bool {
	return false
}

func (shaper *RemoteShaper) StopDisrupt() bool {
	return true
}

func (shaper *RemoteShaper) Close() {
	shaper.Stop()
	shaper.StopDisrupt()
}
func (shaper *RemoteShaper) Pause() {
	shaper.Stop()
}

func (shaper *RemoteShaper) Unpause() {
	shaper.Start()
}

package network

import (
	"github.com/David-Antunes/gone-proxy/xdp"
)

type Link struct {
	originChan      chan *xdp.Frame
	destinationChan chan *xdp.Frame
	props           LinkProps
	shaper          Shaper
}

func CreateLink(originChan chan *xdp.Frame, destinationChan chan *xdp.Frame, props LinkProps) *Link {
	return &Link{
		originChan:      originChan,
		destinationChan: destinationChan,
		props:           props,
		shaper:          CreateNetworkShaper(originChan, destinationChan, props)}
}

func CreateNullLink(originChan chan *xdp.Frame) *Link {
	return &Link{
		originChan:      originChan,
		destinationChan: nil,
		props:           LinkProps{},
		shaper:          CreateNullShaper(originChan),
	}
}

func (link *Link) Start() {
	link.shaper.Start()
}
func (link *Link) Stop() {
	link.shaper.Stop()
}

func (link *Link) Pause() {
	link.shaper.Pause()
}
func (link *Link) Unpause() {
	link.shaper.Unpause()
}

func (link *Link) Close() {
	link.shaper.Close()
}

func (link *Link) GetOriginChan() chan *xdp.Frame {
	return link.originChan
}

func (link *Link) GetDestinationChan() chan *xdp.Frame {
	return link.destinationChan
}

func (link *Link) GetProps() LinkProps {
	return link.props
}

func (link *Link) GetShaper() Shaper {
	return link.shaper
}

func (link *Link) SetOriginChan(originChan chan *xdp.Frame) *Link {
	link.originChan = originChan
	return link
}

func (link *Link) SetDestinationChan(destinationChan chan *xdp.Frame) *Link {
	link.destinationChan = destinationChan
	return link
}

func (link *Link) SetProps(props LinkProps) *Link {
	link.props = props
	return link
}

func (link *Link) SetShaper(shaper Shaper) *Link {
	link.shaper = shaper
	return link
}

func (link *Link) Disrupt() bool {
	return link.shaper.Disrupt()
}

func (link *Link) StopDisrupt() bool {
	return link.shaper.StopDisrupt()
}

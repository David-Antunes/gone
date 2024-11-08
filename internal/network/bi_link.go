package network

type BiLink struct {
	Left  *Link
	Right *Link
}

func CreateBILink(left *Link, right *Link) *BiLink {
	return &BiLink{left, right}
}

func (link *BiLink) GetLeft() *Link {
	return link.Left
}

func (link *BiLink) GetRight() *Link {
	return link.Right
}

func (link *BiLink) SetLeft(Left *Link) {
	link.Left = Left
}

func (link *BiLink) SetRight(Right *Link) {
	link.Right = Right
}

func (link *BiLink) Start() {
	link.Left.Start()
	link.Right.Start()
}

func (link *BiLink) Stop() {
	link.Left.Stop()
	link.Right.Stop()
}
func (link *BiLink) Disrupt() bool {
	link.Left.Disrupt()
	return link.Right.Disrupt()
}

func (link *BiLink) StopDisrupt() bool {
	link.Left.StopDisrupt()
	return link.Right.StopDisrupt()
}
func (link *BiLink) Close() {
	link.Left.Close()
	link.Right.Close()
}

func (link *BiLink) Pause() {
	link.Left.Pause()
	link.Right.Pause()
}

func (link *BiLink) Unpause() {
	link.Left.Unpause()
	link.Right.Unpause()
}

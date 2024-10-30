package api

type Node struct {
	Id        string
	MachineId string
	Bridge    string
	BiLink    BiLink
}

type Bridge struct {
	Id        string
	MachineId string
	BiLink    BiLink
	Nodes     []Node
}

type Router struct {
	Id        string
	MachineId string
	Routers   []Router
	Bridges   []Bridge
	BiLinks   map[string]BiLink
	Weights   map[string]map[string]int
}

type Link struct {
	To        string
	From      string
	LinkProps LinkProps
}

type BiLink struct {
	To   Link
	From Link
}

type LinkProps struct {
	latency   int
	bandwidth int
	jitter    float64
	dropRate  float64
	Weight    int
}

type SniffComponent struct {
	Id        string
	MachineId string
	To        string
	From      string
	Path      string
}
type InterceptComponent struct {
	Id        string
	MachineId string
	To        string
	From      string
	Path      string
}

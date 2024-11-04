package api

type Node struct {
	Id        string
	MachineId string
	Bridge    string
	Link      Link
}

type Bridge struct {
	Id        string
	MachineId string
	Router    string
	Link      Link
	Nodes     []Node
}

type Router struct {
	Id        string
	MachineId string
	Routers   map[string]Link
	Bridges   []Bridge
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
	Latency   int
	Bandwidth int
	Jitter    float64
	DropRate  float64
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

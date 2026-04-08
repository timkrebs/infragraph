package graph

import "time"

// NodeID is the unique identifier for a graph node.
// Convention: "<type>/<name>" e.g. "service/user-api", "pod/user-api-abc12"
type NodeID = string

// EdgeType describes the relationship between two nodes.
type EdgeType = string

const (
	EdgeRoutesTo   EdgeType = "routes_to"
	EdgeSelectedBy EdgeType = "selected_by"
	EdgeMounts     EdgeType = "mounts"
	EdgeDependsOn  EdgeType = "depends_on"
)

// Node represents a single infrastructure resource.
type Node struct {
	ID          NodeID            `json:"id"`
	Type        string            `json:"type"`                 // "service", "pod", "secret", "configmap", "ingress", "pvc", "container"
	Provider    string            `json:"provider"`             // "kubernetes", "docker", "static"
	Namespace   string            `json:"namespace,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
	Status      string            `json:"status"`               // "healthy", "unknown", "degraded"
	Discovered  time.Time         `json:"discovered"`
	Updated     time.Time         `json:"updated"`
}

// Edge represents a directed relationship between two nodes.
type Edge struct {
	From   NodeID   `json:"from"`
	To     NodeID   `json:"to"`
	Type   EdgeType `json:"type"`
	Weight float64  `json:"weight"` // 0.0–1.0 criticality score
}

// Graph is a read-only snapshot of the infrastructure graph.
type Graph interface {
	// Node returns a node by ID, and whether it was found.
	Node(id NodeID) (*Node, bool)
	// Nodes returns all nodes in the graph.
	Nodes() []*Node
	// Edges returns all edges in the graph.
	Edges() []*Edge
	// Neighbors returns the nodes directly reachable from id, along with the edges leading to them.
	Neighbors(id NodeID) ([]*Node, []*Edge)
	// Predecessors returns the nodes that have edges pointing to id.
	Predecessors(id NodeID) ([]*Node, []*Edge)
}

// inMemoryGraph is the concrete snapshot type returned by the store.
type inMemoryGraph struct {
	nodes    map[NodeID]*Node
	edges    []*Edge
	forward  map[NodeID][]*Edge // node → outgoing edges
	backward map[NodeID][]*Edge // node → incoming edges
}

// NewInMemoryGraph builds a Graph from a set of nodes and edges.
func NewInMemoryGraph(nodes []*Node, edges []*Edge) Graph {
	g := &inMemoryGraph{
		nodes:    make(map[NodeID]*Node, len(nodes)),
		edges:    edges,
		forward:  make(map[NodeID][]*Edge),
		backward: make(map[NodeID][]*Edge),
	}
	for _, n := range nodes {
		g.nodes[n.ID] = n
	}
	for _, e := range edges {
		e := e
		g.forward[e.From] = append(g.forward[e.From], e)
		g.backward[e.To] = append(g.backward[e.To], e)
	}
	return g
}

func (g *inMemoryGraph) Node(id NodeID) (*Node, bool) {
	n, ok := g.nodes[id]
	return n, ok
}

func (g *inMemoryGraph) Nodes() []*Node {
	out := make([]*Node, 0, len(g.nodes))
	for _, n := range g.nodes {
		out = append(out, n)
	}
	return out
}

func (g *inMemoryGraph) Edges() []*Edge {
	return g.edges
}

func (g *inMemoryGraph) Neighbors(id NodeID) ([]*Node, []*Edge) {
	edges := g.forward[id]
	nodes := make([]*Node, 0, len(edges))
	for _, e := range edges {
		if n, ok := g.nodes[e.To]; ok {
			nodes = append(nodes, n)
		}
	}
	return nodes, edges
}

func (g *inMemoryGraph) Predecessors(id NodeID) ([]*Node, []*Edge) {
	edges := g.backward[id]
	nodes := make([]*Node, 0, len(edges))
	for _, e := range edges {
		if n, ok := g.nodes[e.From]; ok {
			nodes = append(nodes, n)
		}
	}
	return nodes, edges
}

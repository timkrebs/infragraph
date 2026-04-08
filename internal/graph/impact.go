package graph

import (
	"context"
	"fmt"
)

// ImpactNode is a node reached during traversal, with context about how it was reached.
type ImpactNode struct {
	Node  *Node  `json:"node"`
	ViaID NodeID `json:"via_id"` // ID of the edge source node
	Edge  *Edge  `json:"edge"`
	Depth int    `json:"depth"`
}

// ImpactResult holds the full result of a forward or reverse traversal.
type ImpactResult struct {
	Root     *Node         `json:"root"`
	Affected []*ImpactNode `json:"affected"`
}

// Forward walks "what does this resource affect?" — follows outgoing edges.
// It respects the supplied context: if ctx is cancelled mid-traversal the
// function returns early with the partial result and no error.
func Forward(ctx context.Context, g Graph, id NodeID, maxDepth int) (*ImpactResult, error) {
	root, ok := g.Node(id)
	if !ok {
		return nil, fmt.Errorf("node %q not found", id)
	}
	affected := bfs(ctx, g, id, maxDepth, func(n NodeID) ([]*Node, []*Edge) {
		return g.Neighbors(n)
	})
	return &ImpactResult{Root: root, Affected: affected}, nil
}

// Reverse walks "what does this resource depend on?" — follows incoming edges.
// It respects the supplied context: if ctx is cancelled mid-traversal the
// function returns early with the partial result and no error.
func Reverse(ctx context.Context, g Graph, id NodeID, maxDepth int) (*ImpactResult, error) {
	root, ok := g.Node(id)
	if !ok {
		return nil, fmt.Errorf("node %q not found", id)
	}
	affected := bfs(ctx, g, id, maxDepth, func(n NodeID) ([]*Node, []*Edge) {
		return g.Predecessors(n)
	})
	return &ImpactResult{Root: root, Affected: affected}, nil
}

// bfs performs a breadth-first traversal using the provided next function.
// It checks ctx on each iteration so expensive traversals can be cancelled.
func bfs(ctx context.Context, g Graph, startID NodeID, maxDepth int, next func(NodeID) ([]*Node, []*Edge)) []*ImpactNode {
	visited := map[NodeID]bool{startID: true}
	var result []*ImpactNode

	type item struct {
		id    NodeID
		depth int
	}
	queue := []item{{id: startID, depth: 0}}

	for len(queue) > 0 {
		if ctx.Err() != nil {
			return result
		}

		cur := queue[0]
		queue = queue[1:]

		if cur.depth >= maxDepth {
			continue
		}

		nodes, edges := next(cur.id)
		for i, n := range nodes {
			if visited[n.ID] {
				continue
			}
			visited[n.ID] = true
			result = append(result, &ImpactNode{
				Node:  n,
				ViaID: cur.id,
				Edge:  edges[i],
				Depth: cur.depth + 1,
			})
			queue = append(queue, item{id: n.ID, depth: cur.depth + 1})
		}
	}

	return result
}

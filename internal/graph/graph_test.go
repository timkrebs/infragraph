package graph

import (
	"testing"
)

// --- Graph construction and queries ---

func buildTestGraph() Graph {
	nodes := []*Node{
		{ID: "ingress/frontend", Type: "ingress", Status: "healthy"},
		{ID: "service/api", Type: "service", Status: "healthy"},
		{ID: "service/payments", Type: "service", Status: "healthy"},
		{ID: "pod/api-abc12", Type: "pod", Status: "healthy"},
		{ID: "pod/payments-xyz", Type: "pod", Status: "degraded"},
		{ID: "secret/db-creds", Type: "secret", Status: "healthy"},
	}
	edges := []*Edge{
		{From: "ingress/frontend", To: "service/api", Type: EdgeRoutesTo, Weight: 1.0},
		{From: "ingress/frontend", To: "service/payments", Type: EdgeRoutesTo, Weight: 1.0},
		{From: "service/api", To: "pod/api-abc12", Type: EdgeSelectedBy, Weight: 0.9},
		{From: "service/payments", To: "pod/payments-xyz", Type: EdgeSelectedBy, Weight: 0.9},
		{From: "pod/api-abc12", To: "secret/db-creds", Type: EdgeMounts, Weight: 0.8},
	}
	return NewInMemoryGraph(nodes, edges)
}

func TestNode_Found(t *testing.T) {
	g := buildTestGraph()
	n, ok := g.Node("service/api")
	if !ok {
		t.Fatal("expected to find service/api")
	}
	if n.Type != "service" {
		t.Errorf("expected type=service, got %q", n.Type)
	}
}

func TestNode_NotFound(t *testing.T) {
	g := buildTestGraph()
	_, ok := g.Node("service/nonexistent")
	if ok {
		t.Fatal("expected not found")
	}
}

func TestNodes_Count(t *testing.T) {
	g := buildTestGraph()
	if len(g.Nodes()) != 6 {
		t.Errorf("expected 6 nodes, got %d", len(g.Nodes()))
	}
}

func TestEdges_Count(t *testing.T) {
	g := buildTestGraph()
	if len(g.Edges()) != 5 {
		t.Errorf("expected 5 edges, got %d", len(g.Edges()))
	}
}

func TestNeighbors(t *testing.T) {
	g := buildTestGraph()

	neighbors, edges := g.Neighbors("ingress/frontend")
	if len(neighbors) != 2 {
		t.Fatalf("expected 2 neighbors, got %d", len(neighbors))
	}
	if len(edges) != 2 {
		t.Fatalf("expected 2 outgoing edges, got %d", len(edges))
	}

	ids := map[string]bool{}
	for _, n := range neighbors {
		ids[n.ID] = true
	}
	if !ids["service/api"] || !ids["service/payments"] {
		t.Errorf("expected service/api and service/payments as neighbors, got %v", ids)
	}
}

func TestNeighbors_NoOutgoing(t *testing.T) {
	g := buildTestGraph()
	neighbors, edges := g.Neighbors("secret/db-creds")
	if len(neighbors) != 0 {
		t.Errorf("expected 0 neighbors for leaf node, got %d", len(neighbors))
	}
	if len(edges) != 0 {
		t.Errorf("expected 0 edges for leaf node, got %d", len(edges))
	}
}

func TestPredecessors(t *testing.T) {
	g := buildTestGraph()

	preds, edges := g.Predecessors("secret/db-creds")
	if len(preds) != 1 {
		t.Fatalf("expected 1 predecessor, got %d", len(preds))
	}
	if preds[0].ID != "pod/api-abc12" {
		t.Errorf("expected pod/api-abc12 as predecessor, got %q", preds[0].ID)
	}
	if len(edges) != 1 {
		t.Errorf("expected 1 incoming edge, got %d", len(edges))
	}
}

func TestPredecessors_NoIncoming(t *testing.T) {
	g := buildTestGraph()
	preds, edges := g.Predecessors("ingress/frontend")
	if len(preds) != 0 {
		t.Errorf("expected 0 predecessors for root, got %d", len(preds))
	}
	if len(edges) != 0 {
		t.Errorf("expected 0 incoming edges for root, got %d", len(edges))
	}
}

func TestEmptyGraph(t *testing.T) {
	g := NewInMemoryGraph(nil, nil)
	if len(g.Nodes()) != 0 {
		t.Errorf("expected 0 nodes, got %d", len(g.Nodes()))
	}
	if len(g.Edges()) != 0 {
		t.Errorf("expected 0 edges, got %d", len(g.Edges()))
	}
	_, ok := g.Node("anything")
	if ok {
		t.Error("expected not found in empty graph")
	}
}

func TestSingleNodeGraph(t *testing.T) {
	g := NewInMemoryGraph([]*Node{{ID: "pod/solo", Type: "pod"}}, nil)
	n, ok := g.Node("pod/solo")
	if !ok {
		t.Fatal("expected to find solo node")
	}
	if n.Type != "pod" {
		t.Errorf("expected type=pod, got %q", n.Type)
	}
	neighbors, _ := g.Neighbors("pod/solo")
	if len(neighbors) != 0 {
		t.Errorf("expected 0 neighbors, got %d", len(neighbors))
	}
}

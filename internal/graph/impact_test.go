package graph

import (
	"context"
	"testing"
)

// --- Forward traversal ---

func TestForward_SingleHop(t *testing.T) {
	g := buildTestGraph()
	ctx := context.Background()

	result, err := Forward(ctx, g, "service/api", 1)
	if err != nil {
		t.Fatalf("forward: %v", err)
	}
	if result.Root.ID != "service/api" {
		t.Errorf("expected root=service/api, got %q", result.Root.ID)
	}
	if len(result.Affected) != 1 {
		t.Fatalf("expected 1 affected at depth=1, got %d", len(result.Affected))
	}
	if result.Affected[0].Node.ID != "pod/api-abc12" {
		t.Errorf("expected pod/api-abc12, got %q", result.Affected[0].Node.ID)
	}
	if result.Affected[0].Depth != 1 {
		t.Errorf("expected depth=1, got %d", result.Affected[0].Depth)
	}
}

func TestForward_MultiHop(t *testing.T) {
	g := buildTestGraph()
	ctx := context.Background()

	result, err := Forward(ctx, g, "ingress/frontend", 10)
	if err != nil {
		t.Fatalf("forward: %v", err)
	}
	// ingress → service/api, service/payments → pod/api-abc12, pod/payments-xyz → secret/db-creds
	if len(result.Affected) != 5 {
		t.Errorf("expected 5 affected nodes, got %d", len(result.Affected))
	}
}

func TestForward_DepthZero(t *testing.T) {
	g := buildTestGraph()
	ctx := context.Background()

	result, err := Forward(ctx, g, "ingress/frontend", 0)
	if err != nil {
		t.Fatalf("forward: %v", err)
	}
	if len(result.Affected) != 0 {
		t.Errorf("expected 0 affected at depth=0, got %d", len(result.Affected))
	}
}

func TestForward_LeafNode(t *testing.T) {
	g := buildTestGraph()
	ctx := context.Background()

	result, err := Forward(ctx, g, "secret/db-creds", 10)
	if err != nil {
		t.Fatalf("forward: %v", err)
	}
	if len(result.Affected) != 0 {
		t.Errorf("expected 0 affected from leaf, got %d", len(result.Affected))
	}
}

func TestForward_NodeNotFound(t *testing.T) {
	g := buildTestGraph()
	_, err := Forward(context.Background(), g, "service/ghost", 10)
	if err == nil {
		t.Fatal("expected error for missing node")
	}
}

// --- Reverse traversal ---

func TestReverse_SingleHop(t *testing.T) {
	g := buildTestGraph()
	ctx := context.Background()

	result, err := Reverse(ctx, g, "secret/db-creds", 1)
	if err != nil {
		t.Fatalf("reverse: %v", err)
	}
	if len(result.Affected) != 1 {
		t.Fatalf("expected 1 predecessor, got %d", len(result.Affected))
	}
	if result.Affected[0].Node.ID != "pod/api-abc12" {
		t.Errorf("expected pod/api-abc12, got %q", result.Affected[0].Node.ID)
	}
}

func TestReverse_MultiHop(t *testing.T) {
	g := buildTestGraph()
	ctx := context.Background()

	result, err := Reverse(ctx, g, "secret/db-creds", 10)
	if err != nil {
		t.Fatalf("reverse: %v", err)
	}
	// secret ← pod/api-abc12 ← service/api ← ingress/frontend
	if len(result.Affected) != 3 {
		t.Errorf("expected 3 predecessors, got %d", len(result.Affected))
	}
}

func TestReverse_RootNode(t *testing.T) {
	g := buildTestGraph()
	ctx := context.Background()

	result, err := Reverse(ctx, g, "ingress/frontend", 10)
	if err != nil {
		t.Fatalf("reverse: %v", err)
	}
	if len(result.Affected) != 0 {
		t.Errorf("expected 0 predecessors for root, got %d", len(result.Affected))
	}
}

func TestReverse_NodeNotFound(t *testing.T) {
	g := buildTestGraph()
	_, err := Reverse(context.Background(), g, "pod/ghost", 10)
	if err == nil {
		t.Fatal("expected error for missing node")
	}
}

// --- BFS cycle handling ---

func TestForward_CycleDoesNotLoop(t *testing.T) {
	// A → B → C → A (cycle)
	nodes := []*Node{
		{ID: "service/a", Type: "service"},
		{ID: "service/b", Type: "service"},
		{ID: "service/c", Type: "service"},
	}
	edges := []*Edge{
		{From: "service/a", To: "service/b", Type: EdgeDependsOn},
		{From: "service/b", To: "service/c", Type: EdgeDependsOn},
		{From: "service/c", To: "service/a", Type: EdgeDependsOn},
	}
	g := NewInMemoryGraph(nodes, edges)
	ctx := context.Background()

	result, err := Forward(ctx, g, "service/a", 100)
	if err != nil {
		t.Fatalf("forward: %v", err)
	}
	// BFS should visit B and C, but not re-visit A.
	if len(result.Affected) != 2 {
		t.Errorf("expected 2 affected in cycle, got %d", len(result.Affected))
	}
}

// --- ViaID correctness ---

func TestForward_ViaID(t *testing.T) {
	g := buildTestGraph()
	ctx := context.Background()

	result, err := Forward(ctx, g, "service/api", 10)
	if err != nil {
		t.Fatalf("forward: %v", err)
	}

	for _, a := range result.Affected {
		if a.ViaID == "" {
			t.Errorf("expected ViaID to be set for %q", a.Node.ID)
		}
		if a.Depth == 1 && a.ViaID != "service/api" {
			t.Errorf("expected ViaID=service/api at depth 1, got %q", a.ViaID)
		}
	}
}

// --- Context cancellation ---

func TestForward_CancelledContext(t *testing.T) {
	g := buildTestGraph()
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	result, err := Forward(ctx, g, "ingress/frontend", 10)
	if err != nil {
		t.Fatalf("forward should not error on cancelled ctx: %v", err)
	}
	// With an already-cancelled context the BFS should return early
	// with zero or partial results.
	if len(result.Affected) > 5 {
		t.Errorf("expected partial result, got %d", len(result.Affected))
	}
}

package store

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/timkrebs/infragraph/internal/graph"
)

func openTestStore(t *testing.T) *BboltStore {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	st, err := Open(dbPath)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { st.Close() })
	return st
}

func TestOpen_CreatesFile(t *testing.T) {
	st := openTestStore(t)
	if st.Path() == "" {
		t.Fatal("expected non-empty path")
	}
}

func TestOpen_InvalidPath(t *testing.T) {
	_, err := Open("/nonexistent/dir/test.db")
	if err == nil {
		t.Fatal("expected error for invalid path")
	}
}

// --- UpsertNode / LoadGraph ---

func TestUpsertNode_And_LoadGraph(t *testing.T) {
	st := openTestStore(t)

	node := &graph.Node{ID: "service/api", Type: "service", Provider: "test", Status: "healthy"}
	if err := st.UpsertNode(node); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	g, err := st.LoadGraph()
	if err != nil {
		t.Fatalf("load graph: %v", err)
	}

	n, ok := g.Node("service/api")
	if !ok {
		t.Fatal("expected node to be found")
	}
	if n.Type != "service" {
		t.Errorf("expected type=service, got %q", n.Type)
	}
	if n.Updated.IsZero() {
		t.Error("expected Updated to be set")
	}
	if n.Discovered.IsZero() {
		t.Error("expected Discovered to be set")
	}
}

func TestUpsertNode_Replaces(t *testing.T) {
	st := openTestStore(t)

	node := &graph.Node{ID: "service/api", Type: "service", Status: "healthy"}
	st.UpsertNode(node)

	node.Status = "degraded"
	st.UpsertNode(node)

	g, _ := st.LoadGraph()
	n, _ := g.Node("service/api")
	if n.Status != "degraded" {
		t.Errorf("expected status=degraded after update, got %q", n.Status)
	}
}

// --- NodeCount / EdgeCount ---

func TestNodeCount(t *testing.T) {
	st := openTestStore(t)

	count, err := st.NodeCount()
	if err != nil {
		t.Fatalf("node count: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0, got %d", count)
	}

	st.UpsertNode(&graph.Node{ID: "pod/a", Type: "pod"})
	st.UpsertNode(&graph.Node{ID: "pod/b", Type: "pod"})

	count, _ = st.NodeCount()
	if count != 2 {
		t.Errorf("expected 2, got %d", count)
	}
}

func TestEdgeCount(t *testing.T) {
	st := openTestStore(t)

	st.UpsertEdge(&graph.Edge{From: "service/a", To: "pod/b", Type: graph.EdgeSelectedBy, Weight: 1.0})

	count, _ := st.EdgeCount()
	if count != 1 {
		t.Errorf("expected 1, got %d", count)
	}
}

// --- DeleteNode ---

func TestDeleteNode_RemovesNodeAndEdges(t *testing.T) {
	st := openTestStore(t)

	st.UpsertNode(&graph.Node{ID: "service/api", Type: "service"})
	st.UpsertNode(&graph.Node{ID: "pod/a", Type: "pod"})
	st.UpsertNode(&graph.Node{ID: "pod/b", Type: "pod"})
	st.UpsertEdge(&graph.Edge{From: "service/api", To: "pod/a", Type: graph.EdgeSelectedBy})
	st.UpsertEdge(&graph.Edge{From: "service/api", To: "pod/b", Type: graph.EdgeSelectedBy})
	st.UpsertEdge(&graph.Edge{From: "pod/a", To: "service/api", Type: graph.EdgeDependsOn})

	if err := st.DeleteNode("service/api"); err != nil {
		t.Fatalf("delete node: %v", err)
	}

	nc, _ := st.NodeCount()
	if nc != 2 {
		t.Errorf("expected 2 remaining nodes, got %d", nc)
	}

	ec, _ := st.EdgeCount()
	if ec != 0 {
		t.Errorf("expected 0 remaining edges (all referenced service/api), got %d", ec)
	}
}

func TestDeleteNode_DoesNotDeleteUnrelatedEdges(t *testing.T) {
	st := openTestStore(t)

	st.UpsertNode(&graph.Node{ID: "service/api", Type: "service"})
	st.UpsertNode(&graph.Node{ID: "service/api-secondary", Type: "service"})
	st.UpsertNode(&graph.Node{ID: "pod/x", Type: "pod"})

	// Edge between api-secondary and pod/x should NOT be deleted when we delete service/api.
	st.UpsertEdge(&graph.Edge{From: "service/api-secondary", To: "pod/x", Type: graph.EdgeSelectedBy})
	st.UpsertEdge(&graph.Edge{From: "service/api", To: "pod/x", Type: graph.EdgeSelectedBy})

	st.DeleteNode("service/api")

	ec, _ := st.EdgeCount()
	if ec != 1 {
		t.Errorf("expected 1 remaining edge (api-secondary→pod/x), got %d", ec)
	}

	// Verify the surviving edge is the right one.
	g, _ := st.LoadGraph()
	edges := g.Edges()
	if len(edges) != 1 {
		t.Fatalf("expected 1 edge in graph, got %d", len(edges))
	}
	if edges[0].From != "service/api-secondary" {
		t.Errorf("wrong surviving edge: %s → %s", edges[0].From, edges[0].To)
	}
}

// --- UpsertEdge / DeleteEdge ---

func TestUpsertEdge_And_DeleteEdge(t *testing.T) {
	st := openTestStore(t)

	e := &graph.Edge{From: "service/a", To: "pod/b", Type: graph.EdgeSelectedBy, Weight: 0.9}
	if err := st.UpsertEdge(e); err != nil {
		t.Fatalf("upsert edge: %v", err)
	}

	g, _ := st.LoadGraph()
	edges := g.Edges()
	if len(edges) != 1 {
		t.Fatalf("expected 1 edge, got %d", len(edges))
	}
	if edges[0].Weight != 0.9 {
		t.Errorf("expected weight 0.9, got %f", edges[0].Weight)
	}

	if err := st.DeleteEdge("service/a", "pod/b", graph.EdgeSelectedBy); err != nil {
		t.Fatalf("delete edge: %v", err)
	}

	ec, _ := st.EdgeCount()
	if ec != 0 {
		t.Errorf("expected 0 edges after delete, got %d", ec)
	}
}

// --- LoadGraph ---

func TestLoadGraph_EmptyStore(t *testing.T) {
	st := openTestStore(t)

	g, err := st.LoadGraph()
	if err != nil {
		t.Fatalf("load graph: %v", err)
	}
	if len(g.Nodes()) != 0 {
		t.Errorf("expected 0 nodes, got %d", len(g.Nodes()))
	}
	if len(g.Edges()) != 0 {
		t.Errorf("expected 0 edges, got %d", len(g.Edges()))
	}
}

func TestLoadGraph_FullGraph(t *testing.T) {
	st := openTestStore(t)

	st.UpsertNode(&graph.Node{ID: "service/a", Type: "service"})
	st.UpsertNode(&graph.Node{ID: "pod/b", Type: "pod"})
	st.UpsertEdge(&graph.Edge{From: "service/a", To: "pod/b", Type: graph.EdgeSelectedBy})

	g, _ := st.LoadGraph()

	if len(g.Nodes()) != 2 {
		t.Errorf("expected 2 nodes, got %d", len(g.Nodes()))
	}
	if len(g.Edges()) != 1 {
		t.Errorf("expected 1 edge, got %d", len(g.Edges()))
	}

	neighbors, edges := g.Neighbors("service/a")
	if len(neighbors) != 1 || neighbors[0].ID != "pod/b" {
		t.Errorf("expected neighbor pod/b, got %v", neighbors)
	}
	if len(edges) != 1 {
		t.Errorf("expected 1 outgoing edge, got %d", len(edges))
	}
}

// --- EdgeKey separator ---

func TestEdgeKey_UsesPipeSeparator(t *testing.T) {
	key := edgeKey("service/a", "pod/b", graph.EdgeSelectedBy)
	expected := "service/a|pod/b|selected_by"
	if string(key) != expected {
		t.Errorf("expected %q, got %q", expected, string(key))
	}
}

// --- Backup ---

func TestBackup_WritesNonEmptySnapshot(t *testing.T) {
	st := openTestStore(t)
	st.UpsertNode(&graph.Node{ID: "service/a", Type: "service"})
	st.UpsertNode(&graph.Node{ID: "pod/b", Type: "pod"})

	var buf bytes.Buffer
	if err := st.Backup(&buf); err != nil {
		t.Fatalf("backup: %v", err)
	}
	if buf.Len() == 0 {
		t.Error("expected non-empty backup")
	}

	// The backup should be a valid bbolt database.
	// Verify by opening it and checking node count.
	backupPath := filepath.Join(t.TempDir(), "backup.db")
	if err := os.WriteFile(backupPath, buf.Bytes(), 0o600); err != nil {
		t.Fatalf("write backup file: %v", err)
	}
	st2, err := Open(backupPath)
	if err != nil {
		t.Fatalf("open backup: %v", err)
	}
	defer st2.Close()

	n, err := st2.NodeCount()
	if err != nil {
		t.Fatalf("count nodes: %v", err)
	}
	if n != 2 {
		t.Errorf("expected 2 nodes in backup, got %d", n)
	}
}

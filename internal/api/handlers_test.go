package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/timkrebs/infragraph/internal/graph"
	"github.com/timkrebs/infragraph/internal/store"
)

// seedStore inserts the static collector's data into the given store.
func seedStore(t *testing.T, st store.Store) {
	t.Helper()
	now := time.Now().UTC()

	nodes := []*graph.Node{
		{ID: "ingress/frontend", Type: "ingress", Provider: "static", Namespace: "default", Status: "healthy", Discovered: now, Updated: now},
		{ID: "service/user-api", Type: "service", Provider: "static", Namespace: "default", Status: "healthy", Discovered: now, Updated: now},
		{ID: "service/order-api", Type: "service", Provider: "static", Namespace: "default", Status: "healthy", Discovered: now, Updated: now},
		{ID: "pod/user-api-abc12", Type: "pod", Provider: "static", Namespace: "default", Status: "healthy", Discovered: now, Updated: now},
		{ID: "pod/user-api-def34", Type: "pod", Provider: "static", Namespace: "default", Status: "healthy", Discovered: now, Updated: now},
		{ID: "pod/order-api-xyz99", Type: "pod", Provider: "static", Namespace: "default", Status: "healthy", Discovered: now, Updated: now},
		{ID: "secret/db-credentials", Type: "secret", Provider: "static", Namespace: "default", Status: "healthy", Discovered: now, Updated: now},
		{ID: "configmap/app-config", Type: "configmap", Provider: "static", Namespace: "default", Status: "healthy", Discovered: now, Updated: now},
		{ID: "pvc/user-data", Type: "pvc", Provider: "static", Namespace: "default", Status: "healthy", Discovered: now, Updated: now},
	}

	edges := []*graph.Edge{
		{From: "ingress/frontend", To: "service/user-api", Type: graph.EdgeRoutesTo, Weight: 1.0},
		{From: "ingress/frontend", To: "service/order-api", Type: graph.EdgeRoutesTo, Weight: 1.0},
		{From: "service/user-api", To: "pod/user-api-abc12", Type: graph.EdgeSelectedBy, Weight: 0.9},
		{From: "service/user-api", To: "pod/user-api-def34", Type: graph.EdgeSelectedBy, Weight: 0.9},
		{From: "service/order-api", To: "pod/order-api-xyz99", Type: graph.EdgeSelectedBy, Weight: 0.9},
		{From: "pod/user-api-abc12", To: "secret/db-credentials", Type: graph.EdgeMounts, Weight: 0.8},
		{From: "pod/user-api-def34", To: "secret/db-credentials", Type: graph.EdgeMounts, Weight: 0.8},
		{From: "pod/user-api-abc12", To: "configmap/app-config", Type: graph.EdgeMounts, Weight: 0.5},
		{From: "pod/user-api-abc12", To: "pvc/user-data", Type: graph.EdgeMounts, Weight: 0.7},
		{From: "pod/order-api-xyz99", To: "service/user-api", Type: graph.EdgeDependsOn, Weight: 0.9},
	}

	for _, n := range nodes {
		if err := st.UpsertNode(n); err != nil {
			t.Fatalf("seed UpsertNode %s: %v", n.ID, err)
		}
	}
	for _, e := range edges {
		if err := st.UpsertEdge(e); err != nil {
			t.Fatalf("seed UpsertEdge %s→%s: %v", e.From, e.To, err)
		}
	}
}

// newSeededRouter builds a router backed by a temp bbolt store pre-populated
// with static collector data. Returns the router and cleanup function.
func newSeededRouter(t *testing.T) http.Handler {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	st, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("open test store: %v", err)
	}
	t.Cleanup(func() { st.Close() })

	seedStore(t, st)

	g, err := st.LoadGraph()
	if err != nil {
		t.Fatalf("load graph: %v", err)
	}
	graphPtr := &atomic.Pointer[graph.Graph]{}
	graphPtr.Store(&g)

	return NewRouter(st, graphPtr, slog.Default(), RouterOpts{})
}

func decodeJSON(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode JSON: %v", err)
	}
	return body
}

// --- Health ---

func TestHealth_Returns200(t *testing.T) {
	h := newSeededRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	body := decodeJSON(t, rec)
	if body["status"] != "ok" {
		t.Errorf("expected status=ok, got %v", body["status"])
	}
}

// --- SysStatus ---

func TestSysStatus_ReturnsCounts(t *testing.T) {
	h := newSeededRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/v1/sys/status", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	body := decodeJSON(t, rec)
	if body["node_count"].(float64) != 9 {
		t.Errorf("expected 9 nodes, got %v", body["node_count"])
	}
	if body["edge_count"].(float64) != 10 {
		t.Errorf("expected 10 edges, got %v", body["edge_count"])
	}
}

func TestSysStatus_MethodNotAllowed(t *testing.T) {
	h := newSeededRouter(t)
	req := httptest.NewRequest(http.MethodPost, "/v1/sys/status", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}

// --- GraphFull ---

func TestGraphFull_ReturnsAllNodesAndEdges(t *testing.T) {
	h := newSeededRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/v1/graph", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	body := decodeJSON(t, rec)
	nodes := body["nodes"].([]any)
	edges := body["edges"].([]any)
	if len(nodes) != 9 {
		t.Errorf("expected 9 nodes, got %d", len(nodes))
	}
	if len(edges) != 10 {
		t.Errorf("expected 10 edges, got %d", len(edges))
	}
}

func TestGraphFull_MethodNotAllowed(t *testing.T) {
	h := newSeededRouter(t)
	req := httptest.NewRequest(http.MethodPost, "/v1/graph", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}

// --- ResourcesList ---

func TestResourcesList_All(t *testing.T) {
	h := newSeededRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/v1/resources", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	body := decodeJSON(t, rec)
	nodes := body["nodes"].([]any)
	if len(nodes) != 9 {
		t.Errorf("expected 9 nodes, got %d", len(nodes))
	}
}

func TestResourcesList_FilterByType(t *testing.T) {
	h := newSeededRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/v1/resources?type=pod", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	body := decodeJSON(t, rec)
	nodes := body["nodes"].([]any)
	if len(nodes) != 3 {
		t.Errorf("expected 3 pods, got %d", len(nodes))
	}
}

func TestResourcesList_FilterByNamespace(t *testing.T) {
	h := newSeededRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/v1/resources?namespace=default", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	body := decodeJSON(t, rec)
	nodes := body["nodes"].([]any)
	if len(nodes) != 9 {
		t.Errorf("expected 9 nodes in default namespace, got %d", len(nodes))
	}
}

func TestResourcesList_FilterByTypeNoMatch(t *testing.T) {
	h := newSeededRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/v1/resources?type=deployment", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	body := decodeJSON(t, rec)
	nodes := body["nodes"].([]any)
	if len(nodes) != 0 {
		t.Errorf("expected 0 nodes for type=deployment, got %d", len(nodes))
	}
}

func TestResourcesList_Pagination(t *testing.T) {
	h := newSeededRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/v1/resources?limit=3&offset=0", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	body := decodeJSON(t, rec)
	nodes := body["nodes"].([]any)
	if len(nodes) != 3 {
		t.Errorf("expected 3 nodes with limit=3, got %d", len(nodes))
	}
	if body["total"].(float64) != 9 {
		t.Errorf("expected total=9, got %v", body["total"])
	}
	if body["limit"].(float64) != 3 {
		t.Errorf("expected limit=3, got %v", body["limit"])
	}
	if body["offset"].(float64) != 0 {
		t.Errorf("expected offset=0, got %v", body["offset"])
	}
}

func TestResourcesList_PaginationOffset(t *testing.T) {
	h := newSeededRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/v1/resources?limit=5&offset=7", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	body := decodeJSON(t, rec)
	nodes := body["nodes"].([]any)
	// 9 total, offset 7 → only 2 remaining
	if len(nodes) != 2 {
		t.Errorf("expected 2 nodes with offset=7, got %d", len(nodes))
	}
}

func TestResourcesList_PaginationBadLimit(t *testing.T) {
	h := newSeededRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/v1/resources?limit=-1", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for limit=-1, got %d", rec.Code)
	}
}

func TestResourcesList_PaginationBadOffset(t *testing.T) {
	h := newSeededRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/v1/resources?offset=-1", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for offset=-1, got %d", rec.Code)
	}
}

func TestResourcesList_MethodNotAllowed(t *testing.T) {
	h := newSeededRouter(t)
	req := httptest.NewRequest(http.MethodPost, "/v1/resources", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}

// --- GraphNode ---

func TestGraphNode_Found(t *testing.T) {
	h := newSeededRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/v1/graph/node/service/user-api", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	body := decodeJSON(t, rec)
	node := body["node"].(map[string]any)
	if node["id"] != "service/user-api" {
		t.Errorf("expected node id=service/user-api, got %v", node["id"])
	}
	// service/user-api has 2 outgoing (selected_by), 2 incoming (routes_to + depends_on).
	outgoing := body["outgoing"].([]any)
	if len(outgoing) != 2 {
		t.Errorf("expected 2 outgoing edges, got %d", len(outgoing))
	}
	incoming := body["incoming"].([]any)
	if len(incoming) != 2 {
		t.Errorf("expected 2 incoming edges, got %d", len(incoming))
	}
}

func TestGraphNode_NotFound(t *testing.T) {
	h := newSeededRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/v1/graph/node/service/nonexistent", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestGraphNode_InvalidID(t *testing.T) {
	h := newSeededRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/v1/graph/node/INVALID", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestGraphNode_MethodNotAllowed(t *testing.T) {
	h := newSeededRouter(t)
	req := httptest.NewRequest(http.MethodPost, "/v1/graph/node/service/user-api", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}

// --- GraphImpact ---

func TestGraphImpact_Forward(t *testing.T) {
	h := newSeededRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/v1/graph/impact/ingress/frontend?direction=forward&depth=3", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	body := decodeJSON(t, rec)
	affected := body["affected"].([]any)
	if len(affected) == 0 {
		t.Error("expected affected nodes for forward impact from ingress")
	}
}

func TestGraphImpact_Reverse(t *testing.T) {
	h := newSeededRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/v1/graph/impact/secret/db-credentials?direction=reverse&depth=5", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	body := decodeJSON(t, rec)
	affected := body["affected"].([]any)
	if len(affected) == 0 {
		t.Error("expected affected nodes for reverse impact from secret")
	}
}

func TestGraphImpact_DefaultDirection(t *testing.T) {
	h := newSeededRouter(t)
	// No direction param → defaults to forward.
	req := httptest.NewRequest(http.MethodGet, "/v1/graph/impact/service/user-api", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 (default forward), got %d", rec.Code)
	}
}

func TestGraphImpact_InvalidDirection(t *testing.T) {
	h := newSeededRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/v1/graph/impact/service/user-api?direction=sideways", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for bad direction, got %d", rec.Code)
	}
}

func TestGraphImpact_BadDepth(t *testing.T) {
	h := newSeededRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/v1/graph/impact/service/user-api?depth=xyz", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for non-numeric depth, got %d", rec.Code)
	}
}

func TestGraphImpact_NegativeDepth(t *testing.T) {
	h := newSeededRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/v1/graph/impact/service/user-api?depth=-1", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for negative depth, got %d", rec.Code)
	}
}

func TestGraphImpact_NodeNotFound(t *testing.T) {
	h := newSeededRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/v1/graph/impact/service/nonexistent", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestGraphImpact_InvalidID(t *testing.T) {
	h := newSeededRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/v1/graph/impact/BADID", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid ID, got %d", rec.Code)
	}
}

func TestGraphImpact_MethodNotAllowed(t *testing.T) {
	h := newSeededRouter(t)
	req := httptest.NewRequest(http.MethodPost, "/v1/graph/impact/service/user-api", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}

// --- SysShutdown ---

func TestSysShutdown_MethodNotAllowed(t *testing.T) {
	h := newSeededRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/v1/sys/shutdown", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405 for GET /shutdown, got %d", rec.Code)
	}
}

func TestSysShutdown_Post(t *testing.T) {
	h := newSeededRouter(t)
	req := httptest.NewRequest(http.MethodPost, "/v1/sys/shutdown", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	body := decodeJSON(t, rec)
	if body["status"] != "shutting down" {
		t.Errorf("expected status='shutting down', got %v", body["status"])
	}
}

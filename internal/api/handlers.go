package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync/atomic"

	"github.com/timkrebs/infragraph/internal/collector"
	"github.com/timkrebs/infragraph/internal/graph"
	"github.com/timkrebs/infragraph/internal/store"
)

const version = "0.1.0"

// maxTraversalDepth is the server-side cap on impact analysis depth to prevent
// excessive BFS traversals (DoS protection).
const maxTraversalDepth = 100

// Handlers holds the dependencies shared across all HTTP handlers.
type Handlers struct {
	store    store.Store
	graph    *atomic.Pointer[graph.Graph]
	log      *slog.Logger
	shutdown context.CancelFunc     // set by the server to allow /v1/sys/shutdown
	emit     collector.EventFunc    // set by the server for collector push endpoint
}

// SetShutdown registers the cancel function that /v1/sys/shutdown will call.
func (h *Handlers) SetShutdown(cancel context.CancelFunc) {
	h.shutdown = cancel
}

// writeJSON encodes v as JSON and writes it with the given status code.
// If encoding fails the error is logged but cannot be reported to the client
// because the HTTP header has already been sent.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("writeJSON: failed to encode response", "err", err)
	}
}

// writeError writes a JSON error response.
func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// Health handles GET /health — used by integration tests and load balancers.
// Performs a deeper check: verifies the store is accessible and the graph is loaded.
func (h *Handlers) Health(w http.ResponseWriter, r *http.Request) {
	// Check that the store can serve a simple query.
	if _, err := h.store.NodeCount(); err != nil {
		h.log.Error("health check: store unreachable", "err", err)
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{
			"status": "unhealthy",
			"reason": "store unreachable",
		})
		return
	}

	// Check that the graph snapshot is loaded.
	if h.currentGraph() == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{
			"status": "unhealthy",
			"reason": "graph not loaded",
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// SysStatus handles GET /v1/sys/status.
func (h *Handlers) SysStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	nodeCount, err := h.store.NodeCount()
	if err != nil {
		h.log.Error("store: count nodes failed", "err", err)
		writeError(w, http.StatusInternalServerError, "could not count nodes")
		return
	}
	edgeCount, err := h.store.EdgeCount()
	if err != nil {
		h.log.Error("store: count edges failed", "err", err)
		writeError(w, http.StatusInternalServerError, "could not count edges")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"version":    version,
		"node_count": nodeCount,
		"edge_count": edgeCount,
		"store_path": h.store.Path(),
	})
}

// SysShutdown handles POST /v1/sys/shutdown — triggers graceful server shutdown.
func (h *Handlers) SysShutdown(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	h.log.Warn("shutdown requested", "remote_addr", r.RemoteAddr)
	writeJSON(w, http.StatusOK, map[string]string{"status": "shutting down"})
	if h.shutdown != nil {
		go h.shutdown()
	}
}

// GraphFull handles GET /v1/graph — returns all nodes and edges for visualization.
func (h *Handlers) GraphFull(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	g := h.currentGraph()
	if g == nil {
		writeJSON(w, http.StatusOK, map[string]any{"nodes": []*graph.Node{}, "edges": []*graph.Edge{}})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"nodes": g.Nodes(),
		"edges": g.Edges(),
	})
}

// defaultPageLimit is the default number of resources returned per page.
const defaultPageLimit = 100

// ResourcesList handles GET /v1/resources?type=<type>&namespace=<ns>&limit=N&offset=M.
func (h *Handlers) ResourcesList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	filterType := r.URL.Query().Get("type")
	filterNS := r.URL.Query().Get("namespace")

	// Parse optional pagination params.
	limit := 0 // 0 means "no pagination requested"
	offset := 0
	paginate := false

	if l := r.URL.Query().Get("limit"); l != "" {
		n, err := strconv.Atoi(l)
		if err != nil || n < 1 {
			writeError(w, http.StatusBadRequest, "limit must be a positive integer")
			return
		}
		limit = n
		paginate = true
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		n, err := strconv.Atoi(o)
		if err != nil || n < 0 {
			writeError(w, http.StatusBadRequest, "offset must be a non-negative integer")
			return
		}
		offset = n
		paginate = true
	}
	if paginate && limit == 0 {
		limit = defaultPageLimit
	}

	g := h.currentGraph()
	if g == nil {
		if paginate {
			writeJSON(w, http.StatusOK, map[string]any{"nodes": []*graph.Node{}, "total": 0, "limit": limit, "offset": offset})
		} else {
			writeJSON(w, http.StatusOK, map[string]any{"nodes": []*graph.Node{}})
		}
		return
	}

	all := g.Nodes()
	filtered := make([]*graph.Node, 0, len(all))
	for _, n := range all {
		if filterType != "" && n.Type != filterType {
			continue
		}
		if filterNS != "" && n.Namespace != filterNS {
			continue
		}
		filtered = append(filtered, n)
	}

	if !paginate {
		writeJSON(w, http.StatusOK, map[string]any{"nodes": filtered})
		return
	}

	total := len(filtered)
	// Apply offset/limit to the filtered slice.
	if offset >= total {
		filtered = []*graph.Node{}
	} else {
		end := offset + limit
		if end > total {
			end = total
		}
		filtered = filtered[offset:end]
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"nodes":  filtered,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// GraphNode handles GET /v1/graph/node/{id} — returns a node and its direct edges.
func (h *Handlers) GraphNode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	id := strings.TrimPrefix(r.URL.Path, "/v1/graph/node/")
	if !validNodeID(id) {
		writeError(w, http.StatusBadRequest, "invalid node id")
		return
	}

	g := h.currentGraph()
	if g == nil {
		writeError(w, http.StatusServiceUnavailable, "graph not ready")
		return
	}

	node, ok := g.Node(id)
	if !ok {
		writeError(w, http.StatusNotFound, fmt.Sprintf("node %q not found", id))
		return
	}

	neighbors, edges := g.Neighbors(id)
	preds, inEdges := g.Predecessors(id)

	writeJSON(w, http.StatusOK, map[string]any{
		"node":         node,
		"outgoing":     edges,
		"incoming":     inEdges,
		"neighbors":    neighbors,
		"predecessors": preds,
	})
}

// GraphImpact handles GET /v1/graph/impact/{id}?direction=forward&depth=10.
func (h *Handlers) GraphImpact(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	id := strings.TrimPrefix(r.URL.Path, "/v1/graph/impact/")
	if !validNodeID(id) {
		writeError(w, http.StatusBadRequest, "invalid node id")
		return
	}

	direction := r.URL.Query().Get("direction")
	if direction == "" {
		direction = "forward"
	}
	if direction != "forward" && direction != "reverse" {
		writeError(w, http.StatusBadRequest, "direction must be 'forward' or 'reverse'")
		return
	}

	depth := 10
	if d := r.URL.Query().Get("depth"); d != "" {
		n, err := strconv.Atoi(d)
		if err != nil || n < 1 {
			writeError(w, http.StatusBadRequest, "depth must be a positive integer")
			return
		}
		depth = n
	}
	if depth > maxTraversalDepth {
		depth = maxTraversalDepth
	}

	g := h.currentGraph()
	if g == nil {
		writeError(w, http.StatusServiceUnavailable, "graph not ready")
		return
	}

	var result *graph.ImpactResult
	var err error
	if direction == "forward" {
		result, err = graph.Forward(r.Context(), g, id, depth)
	} else {
		result, err = graph.Reverse(r.Context(), g, id, depth)
	}
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// currentGraph safely loads the current graph snapshot.
func (h *Handlers) currentGraph() graph.Graph {
	p := h.graph.Load()
	if p == nil {
		return nil
	}
	return *p
}

// validNodeIDRe enforces the "type/name" convention:
// - type segment: lowercase alpha, digits, underscores, hyphens
// - one or more name segments separated by "/": same chars plus colons and dots
var validNodeIDRe = regexp.MustCompile(`^[a-z][a-z0-9_-]*/[a-z0-9][a-z0-9_.:-]*$`)

// validNodeID checks that id matches the "type/name" convention.
func validNodeID(id string) bool {
	if len(id) > 512 {
		return false
	}
	return validNodeIDRe.MatchString(id)
}

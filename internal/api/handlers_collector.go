package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/timkrebs/infragraph/internal/collector"
	"github.com/timkrebs/infragraph/internal/graph"
	"github.com/timkrebs/infragraph/version"
)

// maxEventBatch is the maximum number of events accepted in a single push.
const maxEventBatch = 1000

// maxEventBodyBytes limits the request body size for collector push (5 MB).
const maxEventBodyBytes = 5 * 1024 * 1024

// CollectorEvents handles POST /v1/collector/events — accepts event batches
// from remote InfraGraph agents. This is the Vault Agent-style push endpoint.
func (h *Handlers) CollectorEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	if h.emit == nil {
		writeError(w, http.StatusServiceUnavailable, "collector events not configured")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxEventBodyBytes)

	var req collector.PushEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	if len(req.Events) == 0 {
		writeError(w, http.StatusBadRequest, "no events in request")
		return
	}
	if len(req.Events) > maxEventBatch {
		writeError(w, http.StatusBadRequest, "too many events (max 1000)")
		return
	}

	accepted := 0
	now := time.Now().UTC()

	for _, pe := range req.Events {
		if pe.Node == nil {
			continue
		}
		// Ensure timestamps are set for nodes without them.
		if pe.Node.Discovered.IsZero() {
			pe.Node.Discovered = now
		}
		if pe.Node.Updated.IsZero() {
			pe.Node.Updated = now
		}

		var kind collector.EventKind
		switch pe.Kind {
		case "upsert":
			kind = collector.EventUpsert
		case "delete":
			kind = collector.EventDelete
		default:
			continue // skip unknown kinds
		}

		h.emit(collector.Event{
			Kind:  kind,
			Node:  pe.Node,
			Edges: pe.Edges,
		})
		accepted++
	}

	h.log.Info("collector events received",
		"agent", req.Agent,
		"submitted", len(req.Events),
		"accepted", accepted,
	)

	writeJSON(w, http.StatusOK, map[string]any{
		"accepted": accepted,
	})
}

// CollectorRegister handles POST /v1/collector/register — agent registration
// and heartbeat endpoint.
func (h *Handlers) CollectorRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var body struct {
		Agent string `json:"agent"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	// Track the remote agent in the registry so it appears in GET /v1/collectors.
	if h.registry != nil && body.Agent != "" {
		h.registry.Register(body.Agent, "agent")
		h.registry.SetStatus(body.Agent, collector.StatusRunning, "")
	}

	h.log.Info("agent registered", "agent", body.Agent)

	// Respond with server status so the agent can verify connectivity.
	nodeCount, _ := h.store.NodeCount()
	edgeCount, _ := h.store.EdgeCount()

	writeJSON(w, http.StatusOK, map[string]any{
		"status":     "ok",
		"version":    version.Version,
		"node_count": nodeCount,
		"edge_count": edgeCount,
	})
}

// graphNodeFromPushEvent validates and converts a push event node. This is a
// helper for the CollectorEvents handler to sanitize incoming data.
func validPushNode(n *graph.Node) bool {
	if n == nil || n.ID == "" || n.Type == "" {
		return false
	}
	return true
}

// CollectorList handles GET /v1/collectors — returns the status of all
// registered collectors (both local and remote agents).
func (h *Handlers) CollectorList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var list []collector.CollectorInfo
	if h.registry != nil {
		list = h.registry.List()
	}
	if list == nil {
		list = []collector.CollectorInfo{}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"collectors": list,
	})
}

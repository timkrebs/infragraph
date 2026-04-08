package collector

import (
	"context"
	"time"

	"github.com/timkrebs/infragraph/internal/graph"
)

// StaticCollector emits a fixed set of seed resources on startup.
// It is used for development and integration testing so the server is
// immediately queryable without a real Kubernetes cluster or Docker daemon.
type StaticCollector struct{}

// Name returns "static".
func (c *StaticCollector) Name() string { return "static" }

// Run emits seed nodes and edges once, then blocks until ctx is cancelled.
func (c *StaticCollector) Run(ctx context.Context, emit EventFunc) error {
	now := time.Now().UTC()

	nodes := []*graph.Node{
		{ID: "ingress/frontend", Type: "ingress", Provider: "static", Namespace: "default", Status: "healthy", Discovered: now, Updated: now},
		{ID: "service/user-api", Type: "service", Provider: "static", Namespace: "default", Status: "healthy", Discovered: now, Updated: now,
			Labels: map[string]string{"app": "user-api"}},
		{ID: "service/order-api", Type: "service", Provider: "static", Namespace: "default", Status: "healthy", Discovered: now, Updated: now,
			Labels: map[string]string{"app": "order-api"}},
		{ID: "pod/user-api-abc12", Type: "pod", Provider: "static", Namespace: "default", Status: "healthy", Discovered: now, Updated: now,
			Labels: map[string]string{"app": "user-api"}},
		{ID: "pod/user-api-def34", Type: "pod", Provider: "static", Namespace: "default", Status: "healthy", Discovered: now, Updated: now,
			Labels: map[string]string{"app": "user-api"}},
		{ID: "pod/order-api-xyz99", Type: "pod", Provider: "static", Namespace: "default", Status: "healthy", Discovered: now, Updated: now,
			Labels: map[string]string{"app": "order-api"}},
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

	// Emit all nodes and their edges once.
	for _, n := range nodes {
		emit(Event{Kind: EventUpsert, Node: n, Edges: edges})
	}

	// Block until context is cancelled (server shutdown).
	<-ctx.Done()
	return nil
}

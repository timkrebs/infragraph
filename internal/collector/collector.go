package collector

import (
	"context"

	"github.com/timkrebs/infragraph/internal/graph"
)

// EventKind describes the type of change a collector observed.
type EventKind int

const (
	// EventUpsert indicates a resource was created or updated.
	EventUpsert EventKind = iota
	// EventDelete indicates a resource was removed.
	EventDelete
)

// Event is emitted by a collector for each resource change.
type Event struct {
	Kind  EventKind
	Node  *graph.Node
	Edges []*graph.Edge
}

// EventFunc is the callback a collector calls for each event.
type EventFunc func(Event)

// Collector discovers infrastructure resources and emits them as events.
// All built-in collectors (static, kubernetes, docker) implement this interface.
type Collector interface {
	// Name returns the collector's type identifier (matches the HCL label).
	Name() string

	// Run starts the collector loop. It calls emit for each discovered resource
	// and blocks until ctx is cancelled. Implementations must be cancellable.
	Run(ctx context.Context, emit EventFunc) error
}

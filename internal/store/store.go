package store

import (
	"io"

	"github.com/timkrebs/infragraph/internal/graph"
)

// Store is the persistence layer for graph nodes and edges.
// The bbolt implementation is the default; additional backends can implement this interface.
type Store interface {
	// UpsertNode inserts or replaces a node.
	UpsertNode(n *graph.Node) error
	// DeleteNode removes a node and any edges referencing it.
	DeleteNode(id graph.NodeID) error
	// UpsertEdge inserts or replaces an edge.
	UpsertEdge(e *graph.Edge) error
	// DeleteEdge removes a specific edge.
	DeleteEdge(from, to graph.NodeID, edgeType graph.EdgeType) error

	// LoadGraph reads all nodes and edges into a new in-memory snapshot.
	// Call this after collector reconciles to refresh the live graph pointer.
	LoadGraph() (graph.Graph, error)

	// NodeCount and EdgeCount return the current counts without a full load.
	NodeCount() (int, error)
	EdgeCount() (int, error)

	// Path returns the store's filesystem path (empty for in-memory stores).
	Path() string

	// Backup writes a consistent snapshot of the database to w.
	// The snapshot is taken inside a read transaction so writes can proceed
	// concurrently. This is safe to call while the server is running.
	Backup(w io.Writer) error

	// Close releases resources held by the store.
	Close() error
}

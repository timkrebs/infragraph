package store

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	bolt "go.etcd.io/bbolt"

	"github.com/timkrebs/infragraph/internal/graph"
)

var (
	bucketNodes = []byte("nodes")
	bucketEdges = []byte("edges")
)

// BboltStore implements Store using bbolt (an embedded key-value database).
type BboltStore struct {
	db   *bolt.DB
	path string
}

// Open opens or creates the bbolt database at path.
// It fails fast (1-second timeout) if another process already holds the file lock.
func Open(path string) (*BboltStore, error) {
	db, err := bolt.Open(path, 0o600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return nil, fmt.Errorf("open bbolt %q: %w", path, err)
	}

	// Ensure required buckets exist.
	err = db.Update(func(tx *bolt.Tx) error {
		if _, err := tx.CreateBucketIfNotExists(bucketNodes); err != nil {
			return fmt.Errorf("create nodes bucket: %w", err)
		}
		if _, err := tx.CreateBucketIfNotExists(bucketEdges); err != nil {
			return fmt.Errorf("create edges bucket: %w", err)
		}
		return nil
	})
	if err != nil {
		db.Close()
		return nil, err
	}

	return &BboltStore{db: db, path: path}, nil
}

// UpsertNode inserts or replaces a node, updating its Updated timestamp.
func (s *BboltStore) UpsertNode(n *graph.Node) error {
	n.Updated = time.Now().UTC()
	if n.Discovered.IsZero() {
		n.Discovered = n.Updated
	}
	data, err := json.Marshal(n)
	if err != nil {
		return fmt.Errorf("marshal node: %w", err)
	}
	return s.db.Update(func(tx *bolt.Tx) error {
		return tx.Bucket(bucketNodes).Put([]byte(n.ID), data)
	})
}

// DeleteNode removes a node and all edges where it is the From or To endpoint.
func (s *BboltStore) DeleteNode(id graph.NodeID) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		if err := tx.Bucket(bucketNodes).Delete([]byte(id)); err != nil {
			return err
		}
		// Remove edges referencing this node by exact segment match on "|"-separated keys.
		eb := tx.Bucket(bucketEdges)
		var toDelete [][]byte
		if err := eb.ForEach(func(k, _ []byte) error {
			parts := strings.SplitN(string(k), "|", 3)
			if len(parts) == 3 && (parts[0] == id || parts[1] == id) {
				toDelete = append(toDelete, append([]byte(nil), k...))
			}
			return nil
		}); err != nil {
			return fmt.Errorf("scan edges for node %q: %w", id, err)
		}
		for _, k := range toDelete {
			if err := eb.Delete(k); err != nil {
				return err
			}
		}
		return nil
	})
}

// UpsertEdge inserts or replaces an edge.
func (s *BboltStore) UpsertEdge(e *graph.Edge) error {
	data, err := json.Marshal(e)
	if err != nil {
		return fmt.Errorf("marshal edge: %w", err)
	}
	key := edgeKey(e.From, e.To, e.Type)
	return s.db.Update(func(tx *bolt.Tx) error {
		return tx.Bucket(bucketEdges).Put(key, data)
	})
}

// DeleteEdge removes a specific directed edge.
func (s *BboltStore) DeleteEdge(from, to graph.NodeID, edgeType graph.EdgeType) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		return tx.Bucket(bucketEdges).Delete(edgeKey(from, to, edgeType))
	})
}

// LoadGraph reads all nodes and edges into an in-memory snapshot.
func (s *BboltStore) LoadGraph() (graph.Graph, error) {
	var nodes []*graph.Node
	var edges []*graph.Edge

	err := s.db.View(func(tx *bolt.Tx) error {
		if err := tx.Bucket(bucketNodes).ForEach(func(k, v []byte) error {
			var n graph.Node
			if err := json.Unmarshal(v, &n); err != nil {
				return fmt.Errorf("unmarshal node %q: %w", string(k), err)
			}
			nodes = append(nodes, &n)
			return nil
		}); err != nil {
			return err
		}
		if err := tx.Bucket(bucketEdges).ForEach(func(k, v []byte) error {
			var e graph.Edge
			if err := json.Unmarshal(v, &e); err != nil {
				return fmt.Errorf("unmarshal edge %q: %w", string(k), err)
			}
			edges = append(edges, &e)
			return nil
		}); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("load graph: %w", err)
	}
	return graph.NewInMemoryGraph(nodes, edges), nil
}

// NodeCount returns the number of nodes in the store.
func (s *BboltStore) NodeCount() (int, error) {
	var count int
	err := s.db.View(func(tx *bolt.Tx) error {
		count = tx.Bucket(bucketNodes).Stats().KeyN
		return nil
	})
	return count, err
}

// EdgeCount returns the number of edges in the store.
func (s *BboltStore) EdgeCount() (int, error) {
	var count int
	err := s.db.View(func(tx *bolt.Tx) error {
		count = tx.Bucket(bucketEdges).Stats().KeyN
		return nil
	})
	return count, err
}

// Path returns the filesystem path of the database file.
func (s *BboltStore) Path() string { return s.path }

// Backup writes a consistent point-in-time snapshot of the database to w.
// It uses a read-only transaction so concurrent writes are not blocked.
func (s *BboltStore) Backup(w io.Writer) error {
	return s.db.View(func(tx *bolt.Tx) error {
		_, err := tx.WriteTo(w)
		return err
	})
}

// Close closes the underlying bbolt database.
func (s *BboltStore) Close() error { return s.db.Close() }

// edgeKey builds the composite key "from|to|type" using "|" as separator
// to avoid ambiguity with node IDs that may contain colons.
func edgeKey(from, to graph.NodeID, edgeType graph.EdgeType) []byte {
	return []byte(from + "|" + to + "|" + edgeType)
}

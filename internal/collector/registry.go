package collector

import (
	"sort"
	"sync"
	"time"
)

// CollectorStatus describes the current lifecycle state of a running collector.
type CollectorStatus string

const (
	StatusStarting CollectorStatus = "starting"
	StatusRunning  CollectorStatus = "running"
	StatusStopped  CollectorStatus = "stopped"
	StatusError    CollectorStatus = "error"
)

// CollectorInfo is the externally-visible snapshot of a single collector.
type CollectorInfo struct {
	Name      string          `json:"name"`
	Type      string          `json:"type"`
	Status    CollectorStatus `json:"status"`
	StartedAt time.Time       `json:"started_at"`
	LastSync  time.Time       `json:"last_sync,omitempty"`
	Resources int             `json:"resources"`
	Error     string          `json:"error,omitempty"`
}

// Registry tracks the set of running collectors and their status.
// All methods are safe for concurrent use.
type Registry struct {
	mu         sync.RWMutex
	collectors map[string]*CollectorInfo
}

// NewRegistry creates an empty collector registry.
func NewRegistry() *Registry {
	return &Registry{
		collectors: make(map[string]*CollectorInfo),
	}
}

// Register adds a collector to the registry with StatusStarting.
func (r *Registry) Register(name, typ string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.collectors[name] = &CollectorInfo{
		Name:      name,
		Type:      typ,
		Status:    StatusStarting,
		StartedAt: time.Now().UTC(),
	}
}

// SetStatus updates the status and optional error for a collector.
func (r *Registry) SetStatus(name string, status CollectorStatus, errMsg string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if info, ok := r.collectors[name]; ok {
		info.Status = status
		info.Error = errMsg
	}
}

// RecordSync updates the last sync timestamp and resource count for a collector.
func (r *Registry) RecordSync(name string, resources int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if info, ok := r.collectors[name]; ok {
		info.LastSync = time.Now().UTC()
		info.Resources = resources
	}
}

// List returns a snapshot of all collectors sorted by name.
func (r *Registry) List() []CollectorInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]CollectorInfo, 0, len(r.collectors))
	for _, info := range r.collectors {
		out = append(out, *info) // copy
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// Get returns a snapshot of a single collector, or nil if not found.
func (r *Registry) Get(name string) *CollectorInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if info, ok := r.collectors[name]; ok {
		cp := *info
		return &cp
	}
	return nil
}

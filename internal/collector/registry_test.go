package collector

import (
	"testing"
	"time"
)

func TestRegistryRegisterAndList(t *testing.T) {
	r := NewRegistry()

	r.Register("kubernetes", "kubernetes")
	r.Register("docker-local", "docker")

	list := r.List()
	if len(list) != 2 {
		t.Fatalf("expected 2 collectors, got %d", len(list))
	}

	// List is sorted by name.
	if list[0].Name != "docker-local" {
		t.Errorf("expected first collector 'docker-local', got %q", list[0].Name)
	}
	if list[1].Name != "kubernetes" {
		t.Errorf("expected second collector 'kubernetes', got %q", list[1].Name)
	}

	// Both should be in starting status.
	for _, c := range list {
		if c.Status != StatusStarting {
			t.Errorf("collector %q: expected status %q, got %q", c.Name, StatusStarting, c.Status)
		}
	}
}

func TestRegistrySetStatus(t *testing.T) {
	r := NewRegistry()
	r.Register("k8s", "kubernetes")

	r.SetStatus("k8s", StatusRunning, "")
	info := r.Get("k8s")
	if info == nil {
		t.Fatal("expected collector info, got nil")
	}
	if info.Status != StatusRunning {
		t.Errorf("expected status %q, got %q", StatusRunning, info.Status)
	}
	if info.Error != "" {
		t.Errorf("expected empty error, got %q", info.Error)
	}

	r.SetStatus("k8s", StatusError, "connection refused")
	info = r.Get("k8s")
	if info.Status != StatusError {
		t.Errorf("expected status %q, got %q", StatusError, info.Status)
	}
	if info.Error != "connection refused" {
		t.Errorf("expected error 'connection refused', got %q", info.Error)
	}
}

func TestRegistryRecordSync(t *testing.T) {
	r := NewRegistry()
	r.Register("static", "static")

	before := time.Now().UTC()
	r.RecordSync("static", 9)
	after := time.Now().UTC()

	info := r.Get("static")
	if info == nil {
		t.Fatal("expected collector info, got nil")
	}
	if info.Resources != 9 {
		t.Errorf("expected 9 resources, got %d", info.Resources)
	}
	if info.LastSync.Before(before) || info.LastSync.After(after) {
		t.Errorf("LastSync %v not in expected range [%v, %v]", info.LastSync, before, after)
	}
}

func TestRegistryGetNonExistent(t *testing.T) {
	r := NewRegistry()
	if info := r.Get("missing"); info != nil {
		t.Errorf("expected nil for missing collector, got %+v", info)
	}
}

func TestRegistryListEmpty(t *testing.T) {
	r := NewRegistry()
	list := r.List()
	if len(list) != 0 {
		t.Errorf("expected empty list, got %d items", len(list))
	}
}

func TestRegistryGetReturnsCopy(t *testing.T) {
	r := NewRegistry()
	r.Register("test", "static")
	r.SetStatus("test", StatusRunning, "")

	info := r.Get("test")
	info.Status = StatusError // mutate copy

	original := r.Get("test")
	if original.Status != StatusRunning {
		t.Error("Get() returned a reference instead of a copy")
	}
}

func TestRegistrySetStatusNonExistent(t *testing.T) {
	r := NewRegistry()
	// Should not panic.
	r.SetStatus("missing", StatusRunning, "")
	r.RecordSync("missing", 5)
}

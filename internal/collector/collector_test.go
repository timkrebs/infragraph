package collector

import (
	"context"
	"testing"
	"time"
)

func TestStaticCollector_Name(t *testing.T) {
	c := &StaticCollector{}
	if c.Name() != "static" {
		t.Errorf("expected name=static, got %q", c.Name())
	}
}

func TestStaticCollector_EmitsNodesAndEdges(t *testing.T) {
	c := &StaticCollector{}
	ctx, cancel := context.WithCancel(context.Background())

	var events []Event
	go func() {
		c.Run(ctx, func(ev Event) {
			events = append(events, ev)
		})
	}()

	// Give collector time to emit, then cancel.
	time.Sleep(100 * time.Millisecond)
	cancel()
	time.Sleep(50 * time.Millisecond)

	if len(events) == 0 {
		t.Fatal("expected at least one event")
	}

	// Each event should have a node with a valid ID.
	for _, ev := range events {
		if ev.Kind != EventUpsert {
			t.Errorf("expected upsert event, got %d", ev.Kind)
		}
		if ev.Node == nil {
			t.Fatal("expected non-nil node")
		}
		if ev.Node.ID == "" {
			t.Error("expected non-empty node ID")
		}
		if ev.Node.Type == "" {
			t.Error("expected non-empty node type")
		}
		if ev.Node.Provider != "static" {
			t.Errorf("expected provider=static, got %q", ev.Node.Provider)
		}
	}
}

func TestStaticCollector_BlocksUntilCancelled(t *testing.T) {
	c := &StaticCollector{}
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() {
		done <- c.Run(ctx, func(ev Event) {})
	}()

	// Ensure Run doesn't return immediately.
	select {
	case <-done:
		t.Fatal("expected Run to block until context cancelled")
	case <-time.After(50 * time.Millisecond):
		// Good — still running.
	}

	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Errorf("expected nil error after cancel, got %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Run did not return after cancel")
	}
}

func TestStaticCollector_EmitsExpectedNodeTypes(t *testing.T) {
	c := &StaticCollector{}
	ctx, cancel := context.WithCancel(context.Background())

	types := map[string]bool{}
	go func() {
		c.Run(ctx, func(ev Event) {
			if ev.Node != nil {
				types[ev.Node.Type] = true
			}
		})
	}()

	time.Sleep(100 * time.Millisecond)
	cancel()
	time.Sleep(50 * time.Millisecond)

	expected := []string{"ingress", "service", "pod", "secret", "configmap", "pvc"}
	for _, typ := range expected {
		if !types[typ] {
			t.Errorf("expected node type %q to be emitted", typ)
		}
	}
}

package collector

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/timkrebs/infragraph/internal/graph"
)

// PushClient sends collector events to a remote InfraGraph server.
// It mirrors the Vault Agent pattern: a local process discovers resources
// and pushes them to the server over an authenticated HTTPS connection.
type PushClient struct {
	ServerAddr string // e.g. "https://infragraph.example.com:7800"
	Token      string // Bearer token for authentication
	AgentName  string // human-readable agent identifier

	TLSCert       string // optional client TLS cert for mTLS
	TLSKey        string // optional client TLS key
	TLSCACert     string // optional CA cert to verify server
	TLSSkipVerify bool   // skip TLS verification (dev only)

	Logger     *slog.Logger
	httpClient *http.Client
}

// Init configures the HTTP client with TLS settings. Must be called before Push.
func (p *PushClient) Init() error {
	if p.Logger == nil {
		p.Logger = slog.Default()
	}

	tlsCfg := &tls.Config{MinVersion: tls.VersionTLS12}

	if p.TLSSkipVerify {
		tlsCfg.InsecureSkipVerify = true //nolint:gosec // user-configured dev mode
	}

	if p.TLSCACert != "" {
		pool, err := loadCACertPool(p.TLSCACert)
		if err != nil {
			return fmt.Errorf("push client CA: %w", err)
		}
		tlsCfg.RootCAs = pool
	}

	if p.TLSCert != "" && p.TLSKey != "" {
		cert, err := tls.LoadX509KeyPair(expandHome(p.TLSCert), expandHome(p.TLSKey))
		if err != nil {
			return fmt.Errorf("push client cert: %w", err)
		}
		tlsCfg.Certificates = []tls.Certificate{cert}
	}

	p.httpClient = &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: tlsCfg,
		},
	}
	return nil
}

// PushEventRequest is the JSON body sent to POST /v1/collector/events.
type PushEventRequest struct {
	Agent  string      `json:"agent"`
	Events []PushEvent `json:"events"`
}

// PushEvent is a single event in the push batch.
type PushEvent struct {
	Kind  string        `json:"kind"` // "upsert" or "delete"
	Node  *graph.Node   `json:"node"`
	Edges []*graph.Edge `json:"edges,omitempty"`
}

// Push sends a batch of events to the InfraGraph server.
func (p *PushClient) Push(ctx context.Context, events []Event) error {
	if len(events) == 0 {
		return nil
	}

	req := PushEventRequest{
		Agent:  p.AgentName,
		Events: make([]PushEvent, len(events)),
	}
	for i, ev := range events {
		kind := "upsert"
		if ev.Kind == EventDelete {
			kind = "delete"
		}
		req.Events[i] = PushEvent{
			Kind:  kind,
			Node:  ev.Node,
			Edges: ev.Edges,
		}
	}

	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal events: %w", err)
	}

	url := p.ServerAddr + "/v1/collector/events"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if p.Token != "" {
		httpReq.Header.Set("Authorization", "Bearer "+p.Token)
	}

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("push events: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errBody, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("server returned %d: %s", resp.StatusCode, string(errBody))
	}

	return nil
}

// Register sends a registration/heartbeat to the server.
func (p *PushClient) Register(ctx context.Context) error {
	payload, _ := json.Marshal(map[string]string{
		"agent": p.AgentName,
	})

	url := p.ServerAddr + "/v1/collector/register"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if p.Token != "" {
		req.Header.Set("Authorization", "Bearer "+p.Token)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("register: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errBody, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("register failed (%d): %s", resp.StatusCode, string(errBody))
	}

	return nil
}

// WrapWithPush creates an EventFunc that collects events into a batch and
// pushes them to the server periodically or when the batch reaches maxBatch.
// The returned cancel function flushes any remaining events.
func (p *PushClient) WrapWithPush(ctx context.Context, maxBatch int, flushInterval time.Duration) (EventFunc, func()) {
	if maxBatch <= 0 {
		maxBatch = 100
	}
	if flushInterval <= 0 {
		flushInterval = 5 * time.Second
	}

	ch := make(chan Event, maxBatch*2)
	done := make(chan struct{})

	go func() {
		defer close(done)
		batch := make([]Event, 0, maxBatch)
		timer := time.NewTimer(flushInterval)
		defer timer.Stop()

		flush := func() {
			if len(batch) == 0 {
				return
			}
			if err := p.Push(ctx, batch); err != nil {
				p.Logger.Warn("push events failed", "count", len(batch), "err", err)
			} else {
				p.Logger.Debug("pushed events", "count", len(batch))
			}
			batch = batch[:0]
		}

		for {
			select {
			case ev, ok := <-ch:
				if !ok {
					flush()
					return
				}
				batch = append(batch, ev)
				if len(batch) >= maxBatch {
					flush()
					timer.Reset(flushInterval)
				}
			case <-timer.C:
				flush()
				timer.Reset(flushInterval)
			case <-ctx.Done():
				flush()
				return
			}
		}
	}()

	emit := func(ev Event) {
		select {
		case ch <- ev:
		case <-ctx.Done():
		}
	}

	cancel := func() {
		close(ch)
		<-done
	}

	return emit, cancel
}

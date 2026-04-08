package collector

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/timkrebs/infragraph/internal/graph"
)

const defaultDockerSocket = "/var/run/docker.sock"

// DockerCollector discovers containers, networks, and volumes from a Docker
// daemon by polling its REST API over a unix socket.
type DockerCollector struct {
	Socket            string        // unix socket path (default: /var/run/docker.sock)
	ReconcileInterval time.Duration // polling interval (default 60s)
	Logger            *slog.Logger

	// baseURL overrides the Docker API base URL. Used for testing.
	// In production this is always "http://localhost" (over unix socket).
	baseURL string
}

// Name returns "docker".
func (c *DockerCollector) Name() string { return "docker" }

// Run starts the Docker collector loop.
func (c *DockerCollector) Run(ctx context.Context, emit EventFunc) error {
	if c.Logger == nil {
		c.Logger = slog.Default()
	}
	sock := c.Socket
	if sock == "" {
		sock = defaultDockerSocket
	}
	// Strip unix:// prefix if present.
	sock = strings.TrimPrefix(sock, "unix://")

	if c.ReconcileInterval == 0 {
		c.ReconcileInterval = 60 * time.Second
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				return net.DialTimeout("unix", sock, 5*time.Second)
			},
		},
	}

	c.Logger.Info("docker collector configured",
		"socket", sock,
		"interval", c.ReconcileInterval,
	)

	c.reconcile(ctx, client, emit)

	ticker := time.NewTicker(c.ReconcileInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			c.reconcile(ctx, client, emit)
		}
	}
}

// reconcile performs a full discovery of Docker resources.
func (c *DockerCollector) reconcile(ctx context.Context, client *http.Client, emit EventFunc) {
	now := time.Now().UTC()
	c.discoverContainers(ctx, client, now, emit)
	c.discoverNetworks(ctx, client, now, emit)
	c.discoverVolumes(ctx, client, now, emit)
}

// ─── Docker API response types ─────────────────────────────────────────────

type dockerContainer struct {
	ID              string            `json:"Id"`
	Names           []string          `json:"Names"`
	Image           string            `json:"Image"`
	State           string            `json:"State"`
	Labels          map[string]string `json:"Labels"`
	Mounts          []dockerMount     `json:"Mounts"`
	NetworkSettings struct {
		Networks map[string]struct {
			NetworkID string `json:"NetworkID"`
		} `json:"Networks"`
	} `json:"NetworkSettings"`
}

type dockerMount struct {
	Type        string `json:"Type"`
	Name        string `json:"Name"`
	Source      string `json:"Source"`
	Destination string `json:"Destination"`
}

type dockerNetwork struct {
	ID     string            `json:"Id"`
	Name   string            `json:"Name"`
	Driver string            `json:"Driver"`
	Labels map[string]string `json:"Labels"`
}

type dockerVolume struct {
	Name       string            `json:"Name"`
	Driver     string            `json:"Driver"`
	Labels     map[string]string `json:"Labels"`
	Mountpoint string            `json:"Mountpoint"`
}

type dockerVolumeList struct {
	Volumes []dockerVolume `json:"Volumes"`
}

// ─── Discovery functions ────────────────────────────────────────────────────

func (c *DockerCollector) discoverContainers(ctx context.Context, client *http.Client, now time.Time, emit EventFunc) {
	var containers []dockerContainer
	if err := c.dockerGet(ctx, client, "/containers/json?all=true", &containers); err != nil {
		c.Logger.Warn("list containers failed", "err", err)
		return
	}

	for _, ctr := range containers {
		name := containerName(ctr)
		ctrID := "container/" + name

		node := &graph.Node{
			ID:         ctrID,
			Type:       "container",
			Provider:   "docker",
			Labels:     ctr.Labels,
			Status:     mapContainerStatus(ctr.State),
			Discovered: now,
			Updated:    now,
		}

		var edges []*graph.Edge

		// Container → volume edges.
		for _, mount := range ctr.Mounts {
			if mount.Type == "volume" && mount.Name != "" {
				edges = append(edges, &graph.Edge{
					From:   ctrID,
					To:     "volume/" + mount.Name,
					Type:   graph.EdgeMounts,
					Weight: 0.7,
				})
			}
		}

		// Container → network edges.
		for netName := range ctr.NetworkSettings.Networks {
			edges = append(edges, &graph.Edge{
				From:   ctrID,
				To:     "network/" + netName,
				Type:   graph.EdgeDependsOn,
				Weight: 0.6,
			})
		}

		emit(Event{Kind: EventUpsert, Node: node, Edges: edges})
	}
}

func (c *DockerCollector) discoverNetworks(ctx context.Context, client *http.Client, now time.Time, emit EventFunc) {
	var networks []dockerNetwork
	if err := c.dockerGet(ctx, client, "/networks", &networks); err != nil {
		c.Logger.Warn("list networks failed", "err", err)
		return
	}

	for _, net := range networks {
		emit(Event{Kind: EventUpsert, Node: &graph.Node{
			ID:         "network/" + net.Name,
			Type:       "network",
			Provider:   "docker",
			Labels:     net.Labels,
			Status:     "healthy",
			Discovered: now,
			Updated:    now,
		}})
	}
}

func (c *DockerCollector) discoverVolumes(ctx context.Context, client *http.Client, now time.Time, emit EventFunc) {
	var volList dockerVolumeList
	if err := c.dockerGet(ctx, client, "/volumes", &volList); err != nil {
		c.Logger.Warn("list volumes failed", "err", err)
		return
	}

	for _, vol := range volList.Volumes {
		emit(Event{Kind: EventUpsert, Node: &graph.Node{
			ID:         "volume/" + vol.Name,
			Type:       "volume",
			Provider:   "docker",
			Labels:     vol.Labels,
			Status:     "healthy",
			Discovered: now,
			Updated:    now,
		}})
	}
}

// ─── HTTP helper ────────────────────────────────────────────────────────────

func (c *DockerCollector) dockerGet(ctx context.Context, client *http.Client, path string, v any) error {
	// Docker API requests go over the unix socket; the Host header is ignored
	// but required. We use API version 1.47 (Docker Engine 28.x).
	base := c.baseURL
	if base == "" {
		base = "http://localhost"
	}
	url := base + "/v1.47" + path
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("GET %s: %w", path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("GET %s: status %d: %s", path, resp.StatusCode, string(body))
	}

	return json.NewDecoder(resp.Body).Decode(v)
}

// ─── Helpers ────────────────────────────────────────────────────────────────

// containerName extracts a clean name from the Docker container.
func containerName(ctr dockerContainer) string {
	if len(ctr.Names) > 0 {
		// Docker names start with "/", strip it.
		return strings.TrimPrefix(ctr.Names[0], "/")
	}
	// Fallback to short ID.
	if len(ctr.ID) > 12 {
		return ctr.ID[:12]
	}
	return ctr.ID
}

// mapContainerStatus converts Docker state to our status vocabulary.
func mapContainerStatus(state string) string {
	switch state {
	case "running":
		return "healthy"
	case "created", "restarting":
		return "unknown"
	default: // exited, paused, dead, removing
		return "degraded"
	}
}

package collector

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/timkrebs/infragraph/internal/graph"
)

// ─── Kubernetes collector tests ─────────────────────────────────────────────

func TestKubernetesCollector_Name(t *testing.T) {
	c := &KubernetesCollector{}
	if c.Name() != "kubernetes" {
		t.Errorf("expected name=kubernetes, got %q", c.Name())
	}
}

func TestKubernetesCollector_DiscoversPods(t *testing.T) {
	// Fake K8s API server returning pods.
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/namespaces/default/pods":
			json.NewEncoder(w).Encode(map[string]any{
				"items": []map[string]any{
					{
						"metadata": map[string]any{
							"name":      "web-abc12",
							"namespace": "default",
							"labels":    map[string]string{"app": "web"},
						},
						"spec": map[string]any{
							"containers": []map[string]any{
								{"name": "web", "volumeMounts": []any{}},
							},
							"volumes": []map[string]any{
								{"name": "config", "configMap": map[string]string{"name": "app-config"}},
								{"name": "secret", "secret": map[string]string{"secretName": "db-creds"}},
							},
						},
						"status": map[string]string{"phase": "Running"},
					},
				},
			})
		default:
			// Return empty list for other resource types.
			json.NewEncoder(w).Encode(map[string]any{"items": []any{}})
		}
	}))
	defer srv.Close()

	c := &KubernetesCollector{
		Namespaces:        []string{"default"},
		Resources:         []string{"pods"},
		ReconcileInterval: time.Hour, // don't reconcile again during test
	}

	ctx, cancel := context.WithCancel(context.Background())

	var events []Event
	// Directly test reconcile with a fake kubeTransport.
	kt := &kubeTransport{
		Server:     srv.URL,
		HTTPClient: srv.Client(),
	}

	c.reconcile(ctx, kt, func(ev Event) {
		events = append(events, ev)
	})
	cancel()

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	ev := events[0]
	if ev.Kind != EventUpsert {
		t.Errorf("expected upsert, got %d", ev.Kind)
	}
	if ev.Node.ID != "pod/default/web-abc12" {
		t.Errorf("unexpected node ID: %q", ev.Node.ID)
	}
	if ev.Node.Type != "pod" {
		t.Errorf("unexpected type: %q", ev.Node.Type)
	}
	if ev.Node.Provider != "kubernetes" {
		t.Errorf("unexpected provider: %q", ev.Node.Provider)
	}
	if ev.Node.Status != "healthy" {
		t.Errorf("expected healthy, got %q", ev.Node.Status)
	}

	// Should have 2 edges: configmap mount + secret mount.
	if len(ev.Edges) != 2 {
		t.Fatalf("expected 2 edges, got %d", len(ev.Edges))
	}
	edgeMap := map[string]string{}
	for _, e := range ev.Edges {
		edgeMap[e.To] = e.Type
	}
	if edgeMap["configmap/default/app-config"] != graph.EdgeMounts {
		t.Error("expected configmap mount edge")
	}
	if edgeMap["secret/default/db-creds"] != graph.EdgeMounts {
		t.Error("expected secret mount edge")
	}
}

func TestKubernetesCollector_DiscoversServices(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/namespaces/default/services":
			json.NewEncoder(w).Encode(map[string]any{
				"items": []map[string]any{
					{
						"metadata": map[string]any{
							"name": "user-api", "namespace": "default",
							"labels": map[string]string{"app": "user-api"},
						},
						"spec": map[string]any{
							"selector": map[string]string{"app": "user-api"},
							"type":     "ClusterIP",
						},
					},
				},
			})
		default:
			json.NewEncoder(w).Encode(map[string]any{"items": []any{}})
		}
	}))
	defer srv.Close()

	c := &KubernetesCollector{
		Namespaces: []string{"default"},
		Resources:  []string{"services"},
	}

	kt := &kubeTransport{Server: srv.URL, HTTPClient: srv.Client()}
	var events []Event
	c.reconcile(context.Background(), kt, func(ev Event) {
		events = append(events, ev)
	})

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Node.ID != "service/default/user-api" {
		t.Errorf("unexpected ID: %q", events[0].Node.ID)
	}
}

func TestKubernetesCollector_DiscoversIngresses(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/apis/networking.k8s.io/v1/namespaces/default/ingresses":
			json.NewEncoder(w).Encode(map[string]any{
				"items": []map[string]any{
					{
						"metadata": map[string]any{
							"name": "frontend", "namespace": "default",
						},
						"spec": map[string]any{
							"rules": []map[string]any{
								{
									"host": "app.example.com",
									"http": map[string]any{
										"paths": []map[string]any{
											{
												"backend": map[string]any{
													"service": map[string]string{"name": "user-api"},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			})
		default:
			json.NewEncoder(w).Encode(map[string]any{"items": []any{}})
		}
	}))
	defer srv.Close()

	c := &KubernetesCollector{
		Namespaces: []string{"default"},
		Resources:  []string{"ingresses"},
	}

	kt := &kubeTransport{Server: srv.URL, HTTPClient: srv.Client()}
	var events []Event
	c.reconcile(context.Background(), kt, func(ev Event) {
		events = append(events, ev)
	})

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Node.ID != "ingress/default/frontend" {
		t.Errorf("unexpected ID: %q", events[0].Node.ID)
	}
	if len(events[0].Edges) != 1 {
		t.Fatalf("expected 1 edge, got %d", len(events[0].Edges))
	}
	if events[0].Edges[0].To != "service/default/user-api" {
		t.Errorf("expected routes_to service, got %q", events[0].Edges[0].To)
	}
	if events[0].Edges[0].Type != graph.EdgeRoutesTo {
		t.Errorf("expected routes_to, got %q", events[0].Edges[0].Type)
	}
}

func TestKubernetesCollector_ContextCancellation(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"items": []any{}})
	}))
	defer srv.Close()

	c := &KubernetesCollector{
		Namespaces:        []string{"default"},
		Resources:         []string{"pods"},
		ReconcileInterval: time.Hour,
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Manually set the transport to bypass kubeconfig resolution.
	kt := &kubeTransport{Server: srv.URL, HTTPClient: srv.Client()}

	// Cancel before reconcile — should check ctx.Err().
	cancel()
	c.reconcile(ctx, kt, func(ev Event) {
		t.Error("should not emit events after cancellation")
	})
}

func TestNodeID(t *testing.T) {
	tests := []struct {
		resType, ns, name string
		want              string
	}{
		{"pod", "default", "web-abc", "pod/default/web-abc"},
		{"service", "production", "api", "service/production/api"},
		{"configmap", "", "app-config", "configmap/default/app-config"},
	}
	for _, tt := range tests {
		got := nodeID(tt.resType, tt.ns, tt.name)
		if got != tt.want {
			t.Errorf("nodeID(%q, %q, %q) = %q, want %q", tt.resType, tt.ns, tt.name, got, tt.want)
		}
	}
}

func TestMapPodStatus(t *testing.T) {
	tests := []struct {
		phase string
		want  string
	}{
		{"Running", "healthy"},
		{"Succeeded", "healthy"},
		{"Pending", "unknown"},
		{"Failed", "degraded"},
		{"CrashLoopBackOff", "degraded"},
	}
	for _, tt := range tests {
		got := mapPodStatus(tt.phase)
		if got != tt.want {
			t.Errorf("mapPodStatus(%q) = %q, want %q", tt.phase, got, tt.want)
		}
	}
}

// ─── Docker collector tests ─────────────────────────────────────────────────

func TestDockerCollector_Name(t *testing.T) {
	c := &DockerCollector{}
	if c.Name() != "docker" {
		t.Errorf("expected name=docker, got %q", c.Name())
	}
}

func TestDockerCollector_DiscoversContainers(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/v1.41/containers/json":
			json.NewEncoder(w).Encode([]map[string]any{
				{
					"Id":     "abc123def456",
					"Names":  []string{"/my-app"},
					"Image":  "nginx:latest",
					"State":  "running",
					"Labels": map[string]string{"com.docker.compose.service": "web"},
					"Mounts": []map[string]any{
						{"Type": "volume", "Name": "app-data", "Destination": "/data"},
					},
					"NetworkSettings": map[string]any{
						"Networks": map[string]any{
							"bridge": map[string]any{"NetworkID": "net123"},
						},
					},
				},
			})
		case r.URL.Path == "/v1.41/networks":
			json.NewEncoder(w).Encode([]map[string]any{
				{"Id": "net123", "Name": "bridge", "Driver": "bridge"},
			})
		case r.URL.Path == "/v1.41/volumes":
			json.NewEncoder(w).Encode(map[string]any{
				"Volumes": []map[string]any{
					{"Name": "app-data", "Driver": "local"},
				},
			})
		default:
			w.WriteHeader(404)
		}
	}))
	defer srv.Close()

	c := &DockerCollector{
		ReconcileInterval: time.Hour,
		baseURL:           srv.URL,
	}

	var events []Event
	client := srv.Client()
	c.reconcile(context.Background(), client, func(ev Event) {
		events = append(events, ev)
	})

	// Should discover: 1 container + 1 network + 1 volume = 3 events.
	if len(events) != 3 {
		t.Fatalf("expected 3 events, got %d", len(events))
	}

	// Find the container event — it should have edges.
	var found bool
	for _, ev := range events {
		if ev.Node.ID == "container/my-app" {
			found = true
			if ev.Node.Type != "container" {
				t.Errorf("expected type=container, got %q", ev.Node.Type)
			}
			if ev.Node.Provider != "docker" {
				t.Errorf("expected provider=docker, got %q", ev.Node.Provider)
			}
			if ev.Node.Status != "healthy" {
				t.Errorf("expected healthy, got %q", ev.Node.Status)
			}
			if len(ev.Edges) != 2 {
				t.Fatalf("expected 2 edges (volume + network), got %d", len(ev.Edges))
			}
			edgeMap := map[string]string{}
			for _, e := range ev.Edges {
				edgeMap[e.To] = e.Type
			}
			if edgeMap["volume/app-data"] != graph.EdgeMounts {
				t.Error("expected volume mount edge")
			}
			if edgeMap["network/bridge"] != graph.EdgeDependsOn {
				t.Error("expected network depends_on edge")
			}
		}
	}
	if !found {
		t.Error("container/my-app event not found")
	}
}

func TestMapContainerStatus(t *testing.T) {
	tests := []struct {
		state string
		want  string
	}{
		{"running", "healthy"},
		{"created", "unknown"},
		{"restarting", "unknown"},
		{"exited", "degraded"},
		{"paused", "degraded"},
		{"dead", "degraded"},
	}
	for _, tt := range tests {
		got := mapContainerStatus(tt.state)
		if got != tt.want {
			t.Errorf("mapContainerStatus(%q) = %q, want %q", tt.state, got, tt.want)
		}
	}
}

func TestContainerName(t *testing.T) {
	tests := []struct {
		ctr  dockerContainer
		want string
	}{
		{dockerContainer{Names: []string{"/my-app"}}, "my-app"},
		{dockerContainer{Names: []string{"/web-server"}}, "web-server"},
		{dockerContainer{ID: "abc123def456789"}, "abc123def456"},
		{dockerContainer{ID: "short"}, "short"},
	}
	for _, tt := range tests {
		got := containerName(tt.ctr)
		if got != tt.want {
			t.Errorf("containerName(%+v) = %q, want %q", tt.ctr, got, tt.want)
		}
	}
}

// ─── Push client tests ──────────────────────────────────────────────────────

func TestPushClient_Push(t *testing.T) {
	var received PushEventRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/collector/events" {
			w.WriteHeader(404)
			return
		}
		if r.Method != http.MethodPost {
			w.WriteHeader(405)
			return
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			w.WriteHeader(401)
			return
		}
		json.NewDecoder(r.Body).Decode(&received)
		json.NewEncoder(w).Encode(map[string]any{"accepted": len(received.Events)})
	}))
	defer srv.Close()

	push := &PushClient{
		ServerAddr: srv.URL,
		Token:      "test-token",
		AgentName:  "test-agent",
	}
	push.Init()

	now := time.Now().UTC()
	events := []Event{
		{
			Kind: EventUpsert,
			Node: &graph.Node{
				ID: "pod/default/web", Type: "pod", Provider: "kubernetes",
				Discovered: now, Updated: now,
			},
			Edges: []*graph.Edge{
				{From: "pod/default/web", To: "service/default/api", Type: graph.EdgeDependsOn, Weight: 0.9},
			},
		},
	}

	if err := push.Push(context.Background(), events); err != nil {
		t.Fatalf("push failed: %v", err)
	}

	if received.Agent != "test-agent" {
		t.Errorf("expected agent=test-agent, got %q", received.Agent)
	}
	if len(received.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(received.Events))
	}
	if received.Events[0].Kind != "upsert" {
		t.Errorf("expected upsert, got %q", received.Events[0].Kind)
	}
}

func TestPushClient_Register(t *testing.T) {
	var agent string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/collector/register" {
			w.WriteHeader(404)
			return
		}
		var body map[string]string
		json.NewDecoder(r.Body).Decode(&body)
		agent = body["agent"]
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer srv.Close()

	push := &PushClient{ServerAddr: srv.URL, AgentName: "k8s-agent"}
	push.Init()

	if err := push.Register(context.Background()); err != nil {
		t.Fatalf("register failed: %v", err)
	}
	if agent != "k8s-agent" {
		t.Errorf("expected agent=k8s-agent, got %q", agent)
	}
}

func TestPushClient_WrapWithPush(t *testing.T) {
	var received PushEventRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&received)
		json.NewEncoder(w).Encode(map[string]any{"accepted": len(received.Events)})
	}))
	defer srv.Close()

	push := &PushClient{ServerAddr: srv.URL, AgentName: "batch-agent"}
	push.Init()

	ctx, cancel := context.WithCancel(context.Background())
	emit, flush := push.WrapWithPush(ctx, 10, 50*time.Millisecond)

	now := time.Now().UTC()
	emit(Event{Kind: EventUpsert, Node: &graph.Node{
		ID: "pod/default/test", Type: "pod", Provider: "kubernetes",
		Discovered: now, Updated: now,
	}})

	// Wait for flush interval.
	time.Sleep(200 * time.Millisecond)

	cancel()
	flush()

	if len(received.Events) == 0 {
		t.Fatal("expected events to be pushed")
	}
}

func TestPushClient_PushEmpty(t *testing.T) {
	push := &PushClient{ServerAddr: "http://localhost:1"}
	push.Init()

	// Pushing empty batch should be a no-op.
	if err := push.Push(context.Background(), nil); err != nil {
		t.Fatalf("expected nil error for empty push, got %v", err)
	}
}

// ─── Kubeconfig tests ───────────────────────────────────────────────────────

func TestExpandHome(t *testing.T) {
	// Non-home path should be returned as-is.
	got := expandHome("/etc/kubernetes/config")
	if got != "/etc/kubernetes/config" {
		t.Errorf("expected unchanged path, got %q", got)
	}

	// Empty path.
	got = expandHome("")
	if got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

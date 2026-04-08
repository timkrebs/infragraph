package collector

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/timkrebs/infragraph/internal/graph"
)

// Default Kubernetes resource types to discover.
var defaultK8sResources = []string{
	"pods",
	"services",
	"ingresses",
	"configmaps",
	"secrets",
	"persistentvolumeclaims",
}

// KubernetesCollector discovers resources from a Kubernetes cluster by
// periodically polling the API server. It supports kubeconfig-based and
// in-cluster authentication.
type KubernetesCollector struct {
	KubeConfig        string        // path to kubeconfig (empty = in-cluster)
	Context           string        // kubeconfig context (empty = current-context)
	Namespaces        []string      // namespaces to watch (empty = all namespaces)
	Resources         []string      // resource types to discover
	ReconcileInterval time.Duration // polling interval (default 60s)
	Logger            *slog.Logger
}

// Name returns "kubernetes".
func (c *KubernetesCollector) Name() string { return "kubernetes" }

// Run starts the Kubernetes collector loop. It performs a full reconciliation
// on startup and then at every ReconcileInterval until ctx is cancelled.
func (c *KubernetesCollector) Run(ctx context.Context, emit EventFunc) error {
	if c.Logger == nil {
		c.Logger = slog.Default()
	}
	if len(c.Resources) == 0 {
		c.Resources = defaultK8sResources
	}
	if c.ReconcileInterval == 0 {
		c.ReconcileInterval = 60 * time.Second
	}
	if len(c.Namespaces) == 0 {
		c.Namespaces = []string{""} // empty string = all namespaces
	}

	// Resolve kubeconfig or in-cluster credentials.
	var kt *kubeTransport
	var err error
	if c.KubeConfig != "" {
		kt, err = resolveKubeconfig(c.KubeConfig, c.Context)
	} else {
		kt, err = resolveInCluster()
	}
	if err != nil {
		return fmt.Errorf("kubernetes auth: %w", err)
	}

	c.Logger.Info("kubernetes collector configured",
		"server", kt.Server,
		"namespaces", c.Namespaces,
		"resources", c.Resources,
		"interval", c.ReconcileInterval,
	)

	// Initial reconciliation.
	c.reconcile(ctx, kt, emit)

	ticker := time.NewTicker(c.ReconcileInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			c.reconcile(ctx, kt, emit)
		}
	}
}

// reconcile performs a full discovery pass across all configured namespaces
// and resource types.
func (c *KubernetesCollector) reconcile(ctx context.Context, kt *kubeTransport, emit EventFunc) {
	now := time.Now().UTC()

	for _, ns := range c.Namespaces {
		for _, res := range c.Resources {
			if ctx.Err() != nil {
				return
			}
			switch res {
			case "pods":
				c.discoverPods(ctx, kt, ns, now, emit)
			case "services":
				c.discoverServices(ctx, kt, ns, now, emit)
			case "ingresses":
				c.discoverIngresses(ctx, kt, ns, now, emit)
			case "configmaps":
				c.discoverConfigMaps(ctx, kt, ns, now, emit)
			case "secrets":
				c.discoverSecrets(ctx, kt, ns, now, emit)
			case "persistentvolumeclaims":
				c.discoverPVCs(ctx, kt, ns, now, emit)
			default:
				c.Logger.Warn("unknown resource type, skipping", "resource", res)
			}
		}
	}
}

// ─── Kubernetes API response types (minimal) ───────────────────────────────

type k8sList struct {
	Items []json.RawMessage `json:"items"`
}

type k8sMeta struct {
	Name      string            `json:"name"`
	Namespace string            `json:"namespace"`
	Labels    map[string]string `json:"labels"`
}

type k8sPod struct {
	Metadata k8sMeta    `json:"metadata"`
	Spec     k8sPodSpec `json:"spec"`
	Status   struct {
		Phase string `json:"phase"`
	} `json:"status"`
}

type k8sPodSpec struct {
	Containers []struct {
		Name         string `json:"name"`
		VolumeMounts []struct {
			Name string `json:"name"`
		} `json:"volumeMounts"`
		EnvFrom []struct {
			ConfigMapRef *struct{ Name string } `json:"configMapRef"`
			SecretRef    *struct{ Name string } `json:"secretRef"`
		} `json:"envFrom"`
	} `json:"containers"`
	Volumes []struct {
		Name                  string                       `json:"name"`
		ConfigMap             *struct{ Name string }       `json:"configMap"`
		Secret                *struct{ SecretName string } `json:"secret"`
		PersistentVolumeClaim *struct{ ClaimName string }  `json:"persistentVolumeClaim"`
	} `json:"volumes"`
}

type k8sService struct {
	Metadata k8sMeta `json:"metadata"`
	Spec     struct {
		Selector  map[string]string `json:"selector"`
		ClusterIP string            `json:"clusterIP"`
		Type      string            `json:"type"`
	} `json:"spec"`
}

type k8sIngress struct {
	Metadata k8sMeta `json:"metadata"`
	Spec     struct {
		Rules []struct {
			Host string `json:"host"`
			HTTP *struct {
				Paths []struct {
					Backend struct {
						Service *struct {
							Name string `json:"name"`
						} `json:"service"`
					} `json:"backend"`
				} `json:"paths"`
			} `json:"http"`
		} `json:"rules"`
	} `json:"spec"`
}

type k8sConfigMap struct {
	Metadata k8sMeta `json:"metadata"`
}

type k8sSecret struct {
	Metadata k8sMeta `json:"metadata"`
	Type     string  `json:"type"`
}

type k8sPVC struct {
	Metadata k8sMeta `json:"metadata"`
	Status   struct {
		Phase string `json:"phase"`
	} `json:"status"`
}

// ─── Discovery functions ────────────────────────────────────────────────────

func (c *KubernetesCollector) discoverPods(ctx context.Context, kt *kubeTransport, ns string, now time.Time, emit EventFunc) {
	var list k8sList
	if err := c.apiGet(ctx, kt, podPath(ns), &list); err != nil {
		c.Logger.Warn("list pods failed", "namespace", ns, "err", err)
		return
	}

	for _, raw := range list.Items {
		var pod k8sPod
		if err := json.Unmarshal(raw, &pod); err != nil {
			c.Logger.Warn("unmarshal pod", "err", err)
			continue
		}

		nodeID := nodeID("pod", pod.Metadata.Namespace, pod.Metadata.Name)
		node := &graph.Node{
			ID:         nodeID,
			Type:       "pod",
			Provider:   "kubernetes",
			Namespace:  pod.Metadata.Namespace,
			Labels:     pod.Metadata.Labels,
			Status:     mapPodStatus(pod.Status.Phase),
			Discovered: now,
			Updated:    now,
		}

		edges := c.podEdges(pod, nodeID)
		emit(Event{Kind: EventUpsert, Node: node, Edges: edges})
	}
}

func (c *KubernetesCollector) podEdges(pod k8sPod, podID string) []*graph.Edge {
	var edges []*graph.Edge
	ns := pod.Metadata.Namespace

	// Volume-based edges: pod → configmap, secret, pvc.
	for _, vol := range pod.Spec.Volumes {
		if vol.ConfigMap != nil {
			edges = append(edges, &graph.Edge{
				From: podID, To: nodeID("configmap", ns, vol.ConfigMap.Name),
				Type: graph.EdgeMounts, Weight: 0.5,
			})
		}
		if vol.Secret != nil {
			edges = append(edges, &graph.Edge{
				From: podID, To: nodeID("secret", ns, vol.Secret.SecretName),
				Type: graph.EdgeMounts, Weight: 0.8,
			})
		}
		if vol.PersistentVolumeClaim != nil {
			edges = append(edges, &graph.Edge{
				From: podID, To: nodeID("pvc", ns, vol.PersistentVolumeClaim.ClaimName),
				Type: graph.EdgeMounts, Weight: 0.7,
			})
		}
	}

	// EnvFrom-based edges.
	for _, container := range pod.Spec.Containers {
		for _, envFrom := range container.EnvFrom {
			if envFrom.ConfigMapRef != nil {
				edges = append(edges, &graph.Edge{
					From: podID, To: nodeID("configmap", ns, envFrom.ConfigMapRef.Name),
					Type: graph.EdgeDependsOn, Weight: 0.5,
				})
			}
			if envFrom.SecretRef != nil {
				edges = append(edges, &graph.Edge{
					From: podID, To: nodeID("secret", ns, envFrom.SecretRef.Name),
					Type: graph.EdgeDependsOn, Weight: 0.8,
				})
			}
		}
	}

	return edges
}

func (c *KubernetesCollector) discoverServices(ctx context.Context, kt *kubeTransport, ns string, now time.Time, emit EventFunc) {
	var list k8sList
	if err := c.apiGet(ctx, kt, servicePath(ns), &list); err != nil {
		c.Logger.Warn("list services failed", "namespace", ns, "err", err)
		return
	}

	for _, raw := range list.Items {
		var svc k8sService
		if err := json.Unmarshal(raw, &svc); err != nil {
			c.Logger.Warn("unmarshal service", "err", err)
			continue
		}

		svcID := nodeID("service", svc.Metadata.Namespace, svc.Metadata.Name)
		node := &graph.Node{
			ID:         svcID,
			Type:       "service",
			Provider:   "kubernetes",
			Namespace:  svc.Metadata.Namespace,
			Labels:     svc.Metadata.Labels,
			Status:     "healthy",
			Discovered: now,
			Updated:    now,
		}

		// Edges are built later by matchServicesToPods.
		emit(Event{Kind: EventUpsert, Node: node})
	}
}

func (c *KubernetesCollector) discoverIngresses(ctx context.Context, kt *kubeTransport, ns string, now time.Time, emit EventFunc) {
	var list k8sList
	if err := c.apiGet(ctx, kt, ingressPath(ns), &list); err != nil {
		c.Logger.Warn("list ingresses failed", "namespace", ns, "err", err)
		return
	}

	for _, raw := range list.Items {
		var ing k8sIngress
		if err := json.Unmarshal(raw, &ing); err != nil {
			c.Logger.Warn("unmarshal ingress", "err", err)
			continue
		}

		ingID := nodeID("ingress", ing.Metadata.Namespace, ing.Metadata.Name)
		node := &graph.Node{
			ID:         ingID,
			Type:       "ingress",
			Provider:   "kubernetes",
			Namespace:  ing.Metadata.Namespace,
			Labels:     ing.Metadata.Labels,
			Status:     "healthy",
			Discovered: now,
			Updated:    now,
		}

		var edges []*graph.Edge
		for _, rule := range ing.Spec.Rules {
			if rule.HTTP == nil {
				continue
			}
			for _, path := range rule.HTTP.Paths {
				if path.Backend.Service != nil {
					edges = append(edges, &graph.Edge{
						From:   ingID,
						To:     nodeID("service", ing.Metadata.Namespace, path.Backend.Service.Name),
						Type:   graph.EdgeRoutesTo,
						Weight: 1.0,
					})
				}
			}
		}

		emit(Event{Kind: EventUpsert, Node: node, Edges: edges})
	}
}

func (c *KubernetesCollector) discoverConfigMaps(ctx context.Context, kt *kubeTransport, ns string, now time.Time, emit EventFunc) {
	var list k8sList
	if err := c.apiGet(ctx, kt, configMapPath(ns), &list); err != nil {
		c.Logger.Warn("list configmaps failed", "namespace", ns, "err", err)
		return
	}
	for _, raw := range list.Items {
		var cm k8sConfigMap
		if err := json.Unmarshal(raw, &cm); err != nil {
			c.Logger.Warn("unmarshal configmap", "err", err)
			continue
		}
		emit(Event{Kind: EventUpsert, Node: &graph.Node{
			ID:   nodeID("configmap", cm.Metadata.Namespace, cm.Metadata.Name),
			Type: "configmap", Provider: "kubernetes",
			Namespace: cm.Metadata.Namespace, Labels: cm.Metadata.Labels,
			Status: "healthy", Discovered: now, Updated: now,
		}})
	}
}

func (c *KubernetesCollector) discoverSecrets(ctx context.Context, kt *kubeTransport, ns string, now time.Time, emit EventFunc) {
	var list k8sList
	if err := c.apiGet(ctx, kt, secretPath(ns), &list); err != nil {
		c.Logger.Warn("list secrets failed", "namespace", ns, "err", err)
		return
	}
	for _, raw := range list.Items {
		var sec k8sSecret
		if err := json.Unmarshal(raw, &sec); err != nil {
			c.Logger.Warn("unmarshal secret", "err", err)
			continue
		}
		emit(Event{Kind: EventUpsert, Node: &graph.Node{
			ID:   nodeID("secret", sec.Metadata.Namespace, sec.Metadata.Name),
			Type: "secret", Provider: "kubernetes",
			Namespace: sec.Metadata.Namespace, Labels: sec.Metadata.Labels,
			Status: "healthy", Discovered: now, Updated: now,
		}})
	}
}

func (c *KubernetesCollector) discoverPVCs(ctx context.Context, kt *kubeTransport, ns string, now time.Time, emit EventFunc) {
	var list k8sList
	if err := c.apiGet(ctx, kt, pvcPath(ns), &list); err != nil {
		c.Logger.Warn("list pvcs failed", "namespace", ns, "err", err)
		return
	}
	for _, raw := range list.Items {
		var pvc k8sPVC
		if err := json.Unmarshal(raw, &pvc); err != nil {
			c.Logger.Warn("unmarshal pvc", "err", err)
			continue
		}
		status := "healthy"
		if pvc.Status.Phase != "Bound" {
			status = "degraded"
		}
		emit(Event{Kind: EventUpsert, Node: &graph.Node{
			ID:   nodeID("pvc", pvc.Metadata.Namespace, pvc.Metadata.Name),
			Type: "pvc", Provider: "kubernetes",
			Namespace: pvc.Metadata.Namespace, Labels: pvc.Metadata.Labels,
			Status: status, Discovered: now, Updated: now,
		}})
	}
}

// ─── API helpers ────────────────────────────────────────────────────────────

// apiGet performs an authenticated GET to the K8s API and JSON-decodes the response.
func (c *KubernetesCollector) apiGet(ctx context.Context, kt *kubeTransport, path string, v any) error {
	url := strings.TrimRight(kt.Server, "/") + path
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	if kt.Token != "" {
		req.Header.Set("Authorization", "Bearer "+kt.Token)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := kt.HTTPClient.Do(req)
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

// ─── Path builders ──────────────────────────────────────────────────────────

func podPath(ns string) string {
	if ns == "" {
		return "/api/v1/pods"
	}
	return "/api/v1/namespaces/" + ns + "/pods"
}

func servicePath(ns string) string {
	if ns == "" {
		return "/api/v1/services"
	}
	return "/api/v1/namespaces/" + ns + "/services"
}

func ingressPath(ns string) string {
	if ns == "" {
		return "/apis/networking.k8s.io/v1/ingresses"
	}
	return "/apis/networking.k8s.io/v1/namespaces/" + ns + "/ingresses"
}

func configMapPath(ns string) string {
	if ns == "" {
		return "/api/v1/configmaps"
	}
	return "/api/v1/namespaces/" + ns + "/configmaps"
}

func secretPath(ns string) string {
	if ns == "" {
		return "/api/v1/secrets"
	}
	return "/api/v1/namespaces/" + ns + "/secrets"
}

func pvcPath(ns string) string {
	if ns == "" {
		return "/api/v1/persistentvolumeclaims"
	}
	return "/api/v1/namespaces/" + ns + "/persistentvolumeclaims"
}

// ─── Helpers ────────────────────────────────────────────────────────────────

// nodeID builds a graph NodeID in the format "type/namespace/name".
func nodeID(resourceType, namespace, name string) string {
	if namespace == "" {
		namespace = "default"
	}
	return resourceType + "/" + namespace + "/" + name
}

// mapPodStatus converts K8s pod phase to our status vocabulary.
func mapPodStatus(phase string) string {
	switch phase {
	case "Running", "Succeeded":
		return "healthy"
	case "Pending":
		return "unknown"
	default:
		return "degraded"
	}
}

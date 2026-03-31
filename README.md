<p align="center">
  <img src="docs/assets/infragraph-logo.svg" alt="InfraGraph" width="120" />
</p>

<h1 align="center">InfraGraph</h1>

<p align="center">
  <strong>Know your blast radius before you pull the trigger.</strong>
</p>

<p align="center">
  <a href="https://github.com/timkrebs/infragraph/actions"><img src="https://img.shields.io/github/actions/workflow/status/timkrebs/infragraph/ci.yml?branch=main&style=flat-square" alt="Build Status"></a>
  <a href="https://goreportcard.com/report/github.com/timkrebs/infragraph"><img src="https://goreportcard.com/badge/github.com/timkrebs/infragraph?style=flat-square" alt="Go Report Card"></a>
  <a href="https://pkg.go.dev/github.com/timkrebs/infragraph"><img src="https://img.shields.io/badge/go.dev-reference-007d9c?style=flat-square&logo=go&logoColor=white" alt="Go Reference"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-Apache%202.0-blue?style=flat-square" alt="License"></a>
  <a href="https://github.com/timkrebs/infragraph/releases"><img src="https://img.shields.io/github/v/release/timkrebs/infragraph?style=flat-square&color=orange" alt="Latest Release"></a>
</p>

<p align="center">
  <a href="#quickstart">Quickstart</a> ·
  <a href="#why-infragraph">Why InfraGraph</a> ·
  <a href="#how-it-works">How It Works</a> ·
  <a href="#installation">Installation</a> ·
  <a href="#usage">Usage</a> ·
  <a href="#plugins">Plugins</a> ·
  <a href="#roadmap">Roadmap</a> ·
  <a href="#contributing">Contributing</a>
</p>

---

InfraGraph automatically discovers infrastructure resources and their dependencies across Kubernetes, cloud providers, VMs, and services — then builds a live dependency graph you can query, traverse, and analyze. Before you rotate a certificate, scale down a node, or push a config change, InfraGraph tells you exactly what will be affected.

```
$ infragraph impact cert/tls-wildcard-prod

 cert/tls-wildcard-prod
 ├── ingress/api-gateway          (routes_to)
 │   ├── service/user-api         (routes_to)
 │   │   ├── pod/user-api-6f7b8   (selected_by)
 │   │   └── pod/user-api-9a3c1   (selected_by)
 │   └── service/order-api        (routes_to)
 │       ├── pod/order-api-4d5e2  (selected_by)
 │       └── configmap/order-cfg  (mounts)
 ├── ingress/admin-dashboard      (routes_to)
 │   └── service/admin-ui         (routes_to)
 └── secret/tls-wildcard-prod     (encrypted_by)

 Blast radius: 3 services, 3 pods, 1 configmap, 2 ingresses
 Estimated impact: HIGH (production workloads affected)
```

## Why InfraGraph

Monitoring tells you *something is down*. InfraGraph tells you *what will break before you touch it*.

Today, infrastructure dependencies live in tribal knowledge, outdated wiki pages, and the heads of senior engineers who happen to remember that the payment service also depends on that Redis cluster. Every team has experienced the moment where a "simple certificate rotation" cascades into a production incident because nobody mapped the full dependency chain.

Existing tools solve adjacent problems but not this one:

| Tool | What it does | What it doesn't do |
|------|-------------|-------------------|
| Service meshes | Route traffic between services | Map non-service dependencies (certs, secrets, DNS, storage) |
| APM tools | Trace request paths | Discover infrastructure-level dependencies below the application |
| Cloud consoles | Show resources in one provider | Cross-provider, cross-platform dependency graph |
| CMDB systems | Store manually entered asset records | Automatically discover and maintain live relationships |

InfraGraph is **graph-first infrastructure intelligence**. It auto-discovers resources and relationships across your entire stack, maintains a live dependency graph, and gives you instant impact analysis for any change — before you make it.

## How It Works

InfraGraph has four layers:

**Discovery** — Lightweight collectors watch your infrastructure in real time. The Kubernetes collector watches the API server for pods, services, ingresses, configmaps, secrets, and PVCs. Cloud collectors query provider APIs. The plugin system lets you extend discovery to anything — Vault secret engines, Consul services, DNS records, Terraform state.

**Graph engine** — Collectors emit resource events. The graph engine normalizes them into a unified node/edge model, deduplicates across sources, and maintains a live directed graph. Every resource is a node. Every relationship is a typed, weighted edge.

**Analysis** — Graph traversal algorithms compute impact analysis (forward dependencies), root cause candidates (reverse dependencies), blast radius scoring, and change risk assessment. All queries are instant — the graph is always in memory.

**Interfaces** — A CLI for operators, a REST/gRPC API for automation, and an event stream for integrations. Query the graph, run impact analysis, export to Graphviz DOT, or subscribe to topology changes.

```
┌─────────────────────────────────────────────────────┐
│  Discovery layer                                    │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌────────┐ │
│  │ K8s      │ │ Cloud    │ │ Docker   │ │ Plugin │ │
│  │collector │ │collector │ │collector │ │ system │ │
│  └────┬─────┘ └────┬─────┘ └────┬─────┘ └───┬────┘ │
└───────┼────────────┼────────────┼────────────┼──────┘
        └────────────┴─────┬──────┴────────────┘
                           ▼
┌─────────────────────────────────────────────────────┐
│  Core engine                                        │
│  ┌────────────┐  ┌─────────────┐  ┌──────────────┐  │
│  │ Aggregator │─▶│Graph engine │─▶│Impact        │  │
│  │            │  │             │  │analyzer      │  │
│  └────────────┘  └──────┬──────┘  └──────────────┘  │
│                  ┌──────┴──────┐                     │
│                  │ Graph store │                     │
│                  │  (bbolt)    │                     │
│                  └─────────────┘                     │
└─────────────────────────────────────────────────────┘
                           │
        ┌──────────────────┼──────────────────┐
        ▼                  ▼                  ▼
   ┌─────────┐      ┌───────────┐      ┌──────────┐
   │  CLI    │      │ REST/gRPC │      │  Events  │
   └─────────┘      └───────────┘      └──────────┘
```

## Quickstart

Get InfraGraph running against your Kubernetes cluster in under 5 minutes.

### Prerequisites

- A running Kubernetes cluster (kind, k3s, minikube, or any managed cluster)
- `kubectl` configured with cluster access
- Go 1.22+ (if building from source)

### Install

```bash
# Binary (Linux/macOS)
curl -fsSL https://github.com/timkrebs/infragraph/releases/latest/download/install.sh | bash

# Homebrew
brew install timkrebs/tap/infragraph

# From source
go install github.com/timkrebs/infragraph/cmd/infragraph@latest
```

### Discover

```bash
# Start discovery against your current kubeconfig context
infragraph discover --collector kubernetes

# Watch mode — continuously update the graph as resources change
infragraph discover --collector kubernetes --watch
```

### Query

```bash
# List all discovered resources
infragraph query list

# Filter by type
infragraph query list --type service
infragraph query list --type pod --namespace production

# Show a specific resource and its direct neighbors
infragraph query show service/user-api

# Find all paths between two resources
infragraph query path pod/user-api-6f7b8 secret/db-credentials
```

### Analyze

```bash
# Impact analysis — what depends on this resource?
infragraph impact service/redis-primary

# Reverse impact — what does this resource depend on?
infragraph impact --reverse pod/order-api-4d5e2

# Blast radius summary
infragraph impact cert/tls-wildcard-prod --format summary

# Export full graph as Graphviz DOT
infragraph export --format dot > infra.dot
dot -Tsvg infra.dot -o infra.svg

# Export as JSON for custom processing
infragraph export --format json > infra.json
```

## Installation

### Binary releases

Pre-built binaries are available for Linux, macOS, and Windows on the [Releases](https://github.com/timkrebs/infragraph/releases) page.

```bash
# Linux (amd64)
curl -Lo infragraph https://github.com/timkrebs/infragraph/releases/latest/download/infragraph_linux_amd64
chmod +x infragraph
sudo mv infragraph /usr/local/bin/

# macOS (arm64)
curl -Lo infragraph https://github.com/timkrebs/infragraph/releases/latest/download/infragraph_darwin_arm64
chmod +x infragraph
sudo mv infragraph /usr/local/bin/
```

### Helm (in-cluster server mode)

```bash
helm repo add infragraph https://timkrebs.github.io/infragraph
helm repo update
helm install infragraph infragraph/infragraph \
  --namespace infragraph-system \
  --create-namespace
```

### From source

```bash
git clone https://github.com/timkrebs/infragraph.git
cd infragraph
make build
```

## Usage

### Configuration

InfraGraph uses HCL configuration files, following the convention established by HashiCorp tools.

```hcl
# infragraph.hcl

server {
  bind_addr = "0.0.0.0"
  port      = 7800
  
  log_level = "info"
}

store {
  path = "/var/lib/infragraph/graph.db"
}

collector "kubernetes" {
  kubeconfig = "~/.kube/config"
  context    = "production"
  
  namespaces = ["default", "production", "staging"]
  
  # Resource types to discover
  resources = [
    "pods",
    "services",
    "ingresses",
    "configmaps",
    "secrets",
    "persistentvolumeclaims",
  ]
  
  # Refresh interval for full reconciliation
  reconcile_interval = "5m"
}

collector "docker" {
  socket = "unix:///var/run/docker.sock"
}
```

### Server mode

For continuous discovery and a persistent graph, run InfraGraph as a server:

```bash
# Start the server
infragraph server --config infragraph.hcl

# In another terminal, query via CLI
infragraph query list --server localhost:7800
infragraph impact service/user-api --server localhost:7800
```

### API

The server exposes both gRPC and REST endpoints:

```bash
# REST — list all nodes
curl http://localhost:7800/v1/graph/nodes

# REST — impact analysis
curl http://localhost:7800/v1/analysis/impact?resource=service/user-api

# REST — export graph
curl http://localhost:7800/v1/graph/export?format=dot
```

```go
// Go client
import "github.com/timkrebs/infragraph/api/client"

c, _ := client.New("localhost:7800")
result, _ := c.Impact(ctx, "service/user-api", &client.ImpactOptions{
    Depth:     5,
    Direction: client.Forward,
})

for _, node := range result.AffectedNodes {
    fmt.Printf("%s (%s)\n", node.ID, node.EdgeType)
}
```

### Graph data model

Every resource in InfraGraph is a **node** with a typed **edge** connecting it to related resources.

```
Node {
  id:          "service/user-api"       # unique identifier
  type:        "service"                # resource type
  provider:    "kubernetes"             # source collector
  namespace:   "production"             # logical grouping
  labels:      {"app": "user-api"}      # key-value metadata
  annotations: {"team": "platform"}     # extended metadata
  status:      "healthy"                # current health
  discovered:  "2026-03-31T10:00:00Z"   # first seen
  updated:     "2026-03-31T14:30:00Z"   # last updated
}

Edge {
  from:   "ingress/api-gateway"
  to:     "service/user-api"
  type:   "routes_to"
  weight: 0.9                           # criticality score
}
```

## Plugins

InfraGraph is built to be extended. Collectors are pluggable — each one discovers resources from a specific source and emits them into the graph.

### Built-in collectors

| Collector | Resources | Status |
|-----------|-----------|--------|
| `kubernetes` | Pods, Services, Ingresses, ConfigMaps, Secrets, PVCs, Deployments | ✅ Stable |
| `docker` | Containers, Networks, Volumes | ✅ Stable |
| `static` | Any resource declared in YAML | ✅ Stable |

### Community collectors (planned)

| Collector | Resources | Status |
|-----------|-----------|--------|
| `vault` | Secret engines, auth methods, policies, leases | 🚧 In progress |
| `consul` | Services, nodes, KV paths, intentions | 📋 Planned |
| `aws` | EC2, RDS, ELB, Route53, ACM, S3 | 📋 Planned |
| `azure` | VMs, AKS, Key Vault, DNS, App Gateway | 📋 Planned |
| `gcp` | GKE, Cloud SQL, Cloud DNS, GCE | 📋 Planned |
| `terraform` | Resources from state files | 📋 Planned |
| `dns` | Records, zones, resolution chains | 📋 Planned |

### Writing a collector plugin

Collectors implement a simple gRPC interface. You can write plugins in any language — they run as separate processes and communicate over gRPC, following the pattern established by [HashiCorp go-plugin](https://github.com/hashicorp/go-plugin).

```go
package main

import (
    "github.com/timkrebs/infragraph/pkg/plugin"
    "github.com/timkrebs/infragraph/pkg/graph"
)

type MyCollector struct{}

func (c *MyCollector) Discover(ctx context.Context) ([]*graph.Resource, error) {
    // Discover resources from your source
    return []*graph.Resource{
        {
            ID:       "myapp/web-server",
            Type:     "application",
            Provider: "my-collector",
            Labels:   map[string]string{"env": "prod"},
            Edges: []graph.Edge{
                {To: "service/user-api", Type: "depends_on"},
            },
        },
    }, nil
}

func (c *MyCollector) Watch(ctx context.Context, ch chan<- *graph.ResourceEvent) error {
    // Stream resource changes in real time
    return nil
}

func main() {
    plugin.Serve(&plugin.ServeConfig{
        Collector: &MyCollector{},
    })
}
```

Build your plugin, drop it in the plugins directory, and reference it in the config:

```hcl
collector "plugin" "my-collector" {
  command = "./plugins/my-collector"
}
```

See the [Plugin Development Guide](docs/plugins.md) for the full reference.

## Roadmap

| Version | Focus | Key features |
|---------|-------|-------------|
| **v0.1** | **Foundation** | K8s + Docker collectors, graph store, CLI (`discover`, `query`, `impact`), DOT export |
| **v0.2** | **Extensibility** | Plugin framework, static YAML collector, REST API, JSON export |
| **v0.3** | **HashiCorp ecosystem** | Vault collector, Consul collector, live watch mode |
| **v0.4** | **Visualization** | Web UI with interactive graph viewer (React + Cytoscape.js) |
| **v0.5** | **Cloud** | AWS + Azure collectors, drift detection, change risk scoring |
| **v1.0** | **Production** | Stable API, multi-cluster federation, event streams, webhook notifications |

See [ROADMAP.md](ROADMAP.md) for the detailed breakdown with milestones.

## Contributing

InfraGraph is in its early stages and contributions are very welcome. Whether it's a new collector plugin, a bug fix, documentation improvement, or a feature idea — we'd love your help.

```bash
# Clone and build
git clone https://github.com/timkrebs/infragraph.git
cd infragraph
make build

# Run tests
make test

# Run linter
make lint

# Run against a local kind cluster
kind create cluster --name infragraph-dev
make run-dev
```

Please read [CONTRIBUTING.md](CONTRIBUTING.md) before submitting a PR. Key points:

- All code must pass `golangci-lint` and have test coverage
- New collectors should include integration tests with a mock or containerized target
- Commit messages follow [Conventional Commits](https://www.conventionalcommits.org/)
- Discussion happens in [GitHub Discussions](https://github.com/timkrebs/infragraph/discussions) — open an issue or discussion before starting large changes

## Community

- **GitHub Discussions** — Questions, ideas, and general conversation
- **Issues** — Bug reports and feature requests
- **Discord** — Real-time chat (link coming soon)

## License

InfraGraph is licensed under the [Apache License 2.0](LICENSE).

---

<p align="center">
  <sub>Built with care by <a href="https://github.com/timkrebs">Tim Krebs</a> and contributors.</sub>
</p>

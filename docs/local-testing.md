# Local Testing Guide

This guide walks you through testing InfraGraph with real infrastructure data
from Docker and/or Kubernetes running on your local machine.

---

## Prerequisites

```bash
# Build the binary (includes embedded UI)
make build/full
```

---

## Option 1 — Docker

Discovers containers, networks, and volumes from the local Docker daemon.

### 1. Spin up some example containers

```bash
# Create a network and run a few containers so InfraGraph has something to discover
docker network create myapp
docker run -d --name redis   --network myapp redis:7-alpine
docker run -d --name postgres --network myapp -e POSTGRES_PASSWORD=dev postgres:16-alpine
docker run -d --name nginx    --network myapp -p 8080:80 nginx:alpine
```

### 2. Start InfraGraph with the Docker collector

```bash
./bin/infragraph server start --config example/docker-dev.hcl
```

### 3. Open the UI

Navigate to [http://localhost:7800/ui/](http://localhost:7800/ui/).

You should see:
- **Dashboard** — node/edge counts, resource type breakdown (container, network, volume), Docker collector status
- **Resources** — every container, network, and volume listed with status
- **Graph** — visual map showing container → network and container → volume edges
- **Collectors** — the `docker` collector showing as "running" with resource count and last sync time

### 4. Verify via CLI

```bash
# System status
./bin/infragraph status

# List all resources
./bin/infragraph resources list

# List only containers
./bin/infragraph resources list --type container

# Query a specific node
./bin/infragraph graph query container/default/nginx

# Show blast radius
./bin/infragraph graph impact container/default/nginx
```

### 5. Cleanup

```bash
docker rm -f redis postgres nginx
docker network rm myapp
```

---

## Option 2 — Kubernetes

Discovers pods, services, ingresses, configmaps, secrets, and PVCs from your
current kubectl context.

### 1. Ensure you have a running cluster

Any local Kubernetes works: [minikube](https://minikube.sigs.k8s.io/),
[kind](https://kind.sigs.k8s.io/), [Docker Desktop Kubernetes](https://docs.docker.com/desktop/kubernetes/), or [Rancher Desktop](https://rancherdesktop.io/).

```bash
# Verify connectivity
kubectl cluster-info
kubectl get nodes
```

### 2. Deploy sample workloads

```bash
# Create a namespace with a simple app
kubectl create namespace infragraph-demo

kubectl -n infragraph-demo create configmap app-config --from-literal=ENV=dev
kubectl -n infragraph-demo create secret generic db-creds --from-literal=password=s3cret

kubectl -n infragraph-demo apply -f - <<'EOF'
apiVersion: apps/v1
kind: Deployment
metadata:
  name: web
spec:
  replicas: 2
  selector:
    matchLabels:
      app: web
  template:
    metadata:
      labels:
        app: web
    spec:
      containers:
      - name: nginx
        image: nginx:alpine
        ports:
        - containerPort: 80
---
apiVersion: v1
kind: Service
metadata:
  name: web
spec:
  selector:
    app: web
  ports:
  - port: 80
    targetPort: 80
EOF

# Wait for pods to be ready
kubectl -n infragraph-demo wait --for=condition=ready pod -l app=web --timeout=60s
```

### 3. Start InfraGraph with the Kubernetes collector

Edit `example/k8s-dev.hcl` to include your namespace:

```hcl
collector "kubernetes" {
  kubeconfig         = "~/.kube/config"
  namespaces         = ["default", "infragraph-demo"]
  reconcile_interval = "30s"
  resources          = ["pods", "services", "configmaps", "secrets"]
}
```

Then start the server:

```bash
./bin/infragraph server start --config example/k8s-dev.hcl
```

### 4. Open the UI

Navigate to [http://localhost:7800/ui/](http://localhost:7800/ui/).

You should see:
- **Dashboard** — nodes representing pods, services, configmaps, secrets; the `kubernetes` collector in the Collectors card
- **Resources** — filter by type (pod, service, configmap, secret) or namespace
- **Graph** — service → pod (`selected_by`) edges, pod → configmap/secret (`mounts`) edges
- **Impact Analysis** — select `service/infragraph-demo/web` and see its downstream blast radius

### 5. Verify via CLI

```bash
./bin/infragraph status
./bin/infragraph resources list --namespace infragraph-demo
./bin/infragraph resources list --type pod
./bin/infragraph graph query service/infragraph-demo/web
./bin/infragraph graph impact service/infragraph-demo/web
./bin/infragraph graph impact service/infragraph-demo/web --reverse
```

### 6. Cleanup

```bash
kubectl delete namespace infragraph-demo
```

---

## Option 3 — Both Docker + Kubernetes

Use the combined config to discover from both sources simultaneously:

```bash
./bin/infragraph server start --config example/combined-dev.hcl
```

The Collectors page will show both `docker` and `kubernetes` collectors
running side by side with independent sync times and resource counts.

---

## Option 4 — Static (no infra required)

If you don't have Docker or Kubernetes available, the static collector
provides hardcoded sample data:

```bash
./bin/infragraph server start --config example/dev.hcl
```

This emits 9 nodes and 10 edges representing a sample microservice topology
(ingresses, services, pods, configmaps, secrets).

---

## Troubleshooting

| Symptom | Cause | Fix |
|---|---|---|
| Dashboard shows 0 nodes | Collector hasn't completed first sync | Wait 30s for reconcile interval, check server logs |
| Docker collector error | Can't reach Docker socket | Ensure Docker Desktop is running; check `docker ps` works |
| K8s collector error | Invalid kubeconfig or context | Run `kubectl cluster-info` to verify; check kubeconfig path |
| Collectors page is empty | Server started without collector blocks | Check your `.hcl` config has `collector` blocks |
| Port 7800 already in use | Another InfraGraph instance running | `pkill -f "infragraph server"` or change the port |

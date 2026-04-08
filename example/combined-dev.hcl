# Combined config: discovers both Docker and Kubernetes resources.
#
# Usage:
#   ./bin/infragraph server start --config example/combined-dev.hcl

server {
  bind_addr  = "127.0.0.1"
  port       = 7800
  log_level  = "debug"
  log_format = "text"
}

store {
  path = "/tmp/infragraph-combined.db"
}

collector "kubernetes" {
  kubeconfig         = "~/.kube/config"
  namespaces         = ["default"]
  reconcile_interval = "30s"
  resources = [
    "pods",
    "services",
    "ingresses",
    "configmaps",
    "secrets",
    "persistentvolumeclaims",
  ]
}

collector "docker" {
  socket             = "unix:///var/run/docker.sock"
  reconcile_interval = "30s"
}

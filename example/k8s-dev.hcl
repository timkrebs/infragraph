# Kubernetes local development config.
# Discovers pods, services, ingresses, configmaps, secrets, and PVCs
# from the current kubectl context.
#
# Usage:
#   ./bin/infragraph server start --config example/k8s-dev.hcl

server {
  bind_addr  = "127.0.0.1"
  port       = 7800
  log_level  = "debug"
  log_format = "text"
}

store {
  path = "/tmp/infragraph-k8s.db"
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

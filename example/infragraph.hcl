# infragraph.hcl
// Infragraph Configuration File

server {
  bind_addr = "0.0.0.0"
  port      = 7800
  
  // Log levels: debug, info, warn, trace, error
  log_level = "info" 
  log_format = "json"
  log_file  = "/var/log/infragraph/infragraph.log"

}

# TLS listener (Vault-style). Omit this block or set tls_disable = true for plain HTTP.
listener "tcp" {
  address         = "0.0.0.0:7800"
  tls_cert_file   = "/etc/infragraph/tls/cert.pem"
  tls_key_file    = "/etc/infragraph/tls/key.pem"
  tls_min_version = "tls13"
  tls_max_version = "tls13"
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
  socket             = "unix:///var/run/docker.sock"
  reconcile_interval = "1m"
}
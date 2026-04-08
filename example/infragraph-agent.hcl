# infragraph-agent.hcl
// InfraGraph Agent Configuration
//
// The agent runs locally near your infrastructure and pushes discovered
// resources to a remote InfraGraph server — similar to HashiCorp Vault Agent.

server {
  // Address of the InfraGraph server (required).
  address = "https://infragraph.example.com:7800"

  // Bearer token for authentication (must match the server's api_token).
  token = "my-secret-token"

  // Human-readable name for this agent (default: "<collector>-<hostname>").
  agent_name = "prod-k8s-agent"

  // Optional mTLS: client certificate + key for mutual TLS.
  // tls_cert = "/etc/infragraph/agent/tls/client.pem"
  // tls_key  = "/etc/infragraph/agent/tls/client-key.pem"

  // CA certificate to verify the server's TLS certificate.
  // tls_ca_cert = "/etc/infragraph/agent/tls/ca.pem"

  // Skip TLS verification (development only — never use in production).
  // tls_skip_verify = true
}

// The collector block defines what the agent discovers locally.
// Supported types: "kubernetes", "docker", "static".

collector "kubernetes" {
  kubeconfig = "~/.kube/config"
  context    = "production"

  namespaces = ["default", "production", "staging"]

  resources = [
    "pods",
    "services",
    "ingresses",
    "configmaps",
    "secrets",
    "persistentvolumeclaims",
  ]

  reconcile_interval = "5m"
}

// Example: Docker collector
// collector "docker" {
//   socket             = "unix:///var/run/docker.sock"
//   reconcile_interval = "1m"
// }

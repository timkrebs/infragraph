# Docker local development config.
# Discovers containers, networks, and volumes from the local Docker daemon.
#
# Usage:
#   ./bin/infragraph server start --config example/docker-dev.hcl

server {
  bind_addr  = "127.0.0.1"
  port       = 8080
  log_level  = "debug"
  log_format = "text"
}

store {
  path = "/tmp/infragraph-docker.db"
}

collector "docker" {
  socket             = "unix:///var/run/docker.sock"
  reconcile_interval = "30s"
}

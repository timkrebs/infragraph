# Minimal dev config — uses the static collector so the UI has sample data.
server {
  bind_addr  = "127.0.0.1"
  port       = 7800
  log_level  = "debug"
  log_format = "text"
}

store {
  path = "/tmp/infragraph-dev.db"
}

collector "static" {}

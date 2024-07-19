ui = true

# Loop-back listener configuration:
listener "tcp" {
  address = "127.0.0.1:8200"
  tls_disable = "true"
}

storage "file" {
  path = "/vault/file"
}

default_lease_ttl = "786h"
max_lease_ttl = "7860h"

plugin_directory = "/vault/plugins"
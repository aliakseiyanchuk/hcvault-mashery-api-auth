ui = true

listener "tcp" {
  address = "0.0.0.0:8200"
  tls_disable = "true"
}

storage "file" {
  path = "/vault/file"
}

default_lease_ttl = "786h"
max_lease_ttl = "7860h"

plugin_directory = "/etc/vault/vault_plugins"
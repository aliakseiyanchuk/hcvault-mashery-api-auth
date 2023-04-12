ui = true

listener "tcp" {
  address = "0.0.0.0:8200"
  tls_disable = "false"
  tls_cert_file = "/vault/tls/vault-container.pem"
  tls_key_file = "/vault/tls/vault-container.key"
}

storage "file" {
  path = "/vault/file"
}

default_lease_ttl = "786h"
max_lease_ttl = "7860h"

plugin_directory = "/vault/plugins"
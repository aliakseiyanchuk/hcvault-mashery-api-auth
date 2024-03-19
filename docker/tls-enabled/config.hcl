ui = true

listener "tcp" {
  address = "0.0.0.0:8200"
  tls_disable = "false"
  tls_cert_file = "/vault/tls/vault-container.pem"
  tls_key_file = "/vault/tls/vault-container.key"
}

listener "tcp" {
  address = "127.0.0.1:8973"
  tls_disable = "true"
}

api_addr = "http://127.0.0.1:8973"

storage "file" {
  path = "/vault/file"
}

default_lease_ttl = "786h"
max_lease_ttl = "7860h"

plugin_directory = "/vault/plugins"
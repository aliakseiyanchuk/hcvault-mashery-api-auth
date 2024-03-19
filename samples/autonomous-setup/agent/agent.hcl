# Vault agent configuration to automatically authenticate to Mashery API

pid_file="./agent.pid"

vault {
  address="https://localhost:8200"
}

auto_auth {
  method {
    type = "approle"

    config = {
      role_id_file_path = "./agent_role_id"
      secret_id_file_path = "./agent_secret_id"
      remove_secret_id_file_after_reading = true
    }

    sink {
      type = "file"
      wrap_ttl = "30m"
      config = {
        path = "./sink.txt"
      }
    }
  }
}

api_proxy {
  use_auto_auth_token = true
}

listener "tcp" {
  address = "127.0.0.1:8100"
  tls_disable = true
}

log_file="./agent.log"
log_level="info"
log_rotate_bytes=52428800
log_rotate_max_files=3
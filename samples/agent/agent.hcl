pid_file = "./pidfile"

vault {
  address = "http://localhost:8200"
  retry {
    num_tries = 5
  }
}

auto_auth {
  method "approle" {
    config = {
      role_id_file_path="./.secrets/role-id.txt"
      secret_id_file_path="./.secrets/secret-id.txt"
      remove_secret_id_file_after_reading= false
    }
  }
}

cache {
  use_auto_auth_token = true
}

listener "tcp" {
  address = "127.0.0.1:8300"
  tls_disable = true
}
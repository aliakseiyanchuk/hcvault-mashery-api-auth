path "mash-auth/roles/demoRoleV2/proxy/v2" {
  capabilities = [ "create" ]
#  allowed_parameters = {
#    method = ["oauth2.fetchApplication"]
#  }
}

path "mash-auth/roles/demoRoleV2/grant" {
  capabilities = [ "read" ]
}

# Proxy mode allowed for V3 API in read-only mode
path "mash-auth/roles/demoRoleV3/proxy/v3/*" {
  capabilities = [ "list", "read" ]
}

# Grants are allowed only for V3 API, and lease is mandatory.
path "mash-auth/roles/demoRoleV3/grant" {
  capabilities = [ "read" ]
  allowed_parameters = {
    api = ["3"]
    lease=["true"]
  }
}
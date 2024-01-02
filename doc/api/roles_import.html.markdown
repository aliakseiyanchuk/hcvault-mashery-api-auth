---
layout: api 
page_title: /role/:roleName/import - HTTP API 
description: |-
  The `/role/:roleName/import` endpoint is used to import the encrypted  data received from another
  Vault administrator
---

# `/roles/:roleName/import`

The `/role/:roleName/import` endpoint is used to import the encrypted  data received from another
Vault administrator

### Parameters

- `roleName` `(string, <required>)` - name of the role.
- `pem` `(string, <required>)` - PEM-encoded encrypted data for this role. This value is obtained with 
  `/role/:roleName/export` [method](./roles_export.html.markdown). 

### Sample Payload

```json
{
  "pem": "------BEGIN MASHERY ROLE DATA-----\n[....PEM Data.....]\n-----END MASHERY ROLE DATA-----\n"
}
```

### Sample Request

**cURL**:

```shell
curl \
  --header 'X-Vault-Token: ...' \
  --data=@paylaod.json \
  --request PUT 'http://127.0.0.1:8200/v1/mash-creds/roles/sample/import'
```

**Vault CLI:**

```shell
cat payload.json | vault write mash-creds/roles/sample/import -
```

WHere the PEM file is stored directly in the file system, it can be imported by supplying `pem` property with the
CLI key-value notation:
```shell
vault write mash-creds/roles/sample/import pem=@pem_file.pem
```

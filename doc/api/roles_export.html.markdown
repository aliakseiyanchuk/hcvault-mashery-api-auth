---
layout: api 
page_title: /role/:roleName/export - HTTP API 
description: |-
The `/role/:roleName/export` endpoint is used to encrypt role's data for secure transfer to another
Vault administrator
---

# `/roles/:roleName/export`

The `/role/:roleName/pem` endpoint is used to export role's data in the encrypted form for the secure hand-over to
the intended recipient.

### Parameters

- `roleName` `(string, <required>)` - name of the role.
- 'pem' `(string, <required>)` - PEM-encoded receipient role identity. This value is obtained with 
  `/role/:roleName/pem` [method](./roles_pem.html.markdown). 
- `explicit_term` `(string, "")`: term for which the recipient can use the exported data. The value accepts the following formats:
  - `yyyy-MM-dd`, an explicit year-month-date format;
  - `(\d+)w`, number of weeks since current time
  - `(\d+)d`, number of days since current time
  - any valid Go language [ParseDuration](https://pkg.go.dev/time#example-ParseDuration) function accepts
- `explicit_num_uses` `(number, 0)` - if greater than zero, number of times this record can be used by recipient
- `explicit_qps` `(number, 0)` - if supplied, will override this role's QPS. This allows an administrator to create
   limited-qps grants from data records that are allowed high QPS.
- `v2_only` `(bool, false)` - only export data sufficient for V2 calls
- `v3_only` `(bool, false)` - only export data sufficient for V3 calls
- `force_proxy_mode` `(bool, false)` - if set to `true`, the recipient cannot use `/role/:roleName/grant` method to 
   extract Mashery credentials by value.

### Sample Payload

```json
{
  "pem": "-----BEGIN MASHERY ROLE RECIPIENT-----\n[....PEM DATA....]\n-----END MASHERY ROLE RECIPIENT-----\n",
  "explicit_term": "3w",
  "explicit_num_uses": 5000,
  "explicit_qps": 5,
  "force_proxy_mode": true
}
```

### Sample Request

**cURL**:

```shell
curl \
  --header 'X-Vault-Token: ...' \
  --data=@paylaod.json \
  --request PUT 'http://127.0.0.1:8200/v1/mash-creds/roles/sample/export'
```

**Vault CLI:**

```shell
cat payload.json | vault write mash-creds/roles/sample/export -
```

To read the PEM contents with `jq`, execute
```shell
cat payload.json | vault  write mash-creds/roles/sample/export - | jq -r .data.pem > file_in_file_system.pem
```

### Sample Response

```json
{
  "pem": "-----BEGIN MASHERY ROLE DATA-----\n[....PEM Data.....]\n-----END MASHERY ROLE DATA-----\n"
}
```


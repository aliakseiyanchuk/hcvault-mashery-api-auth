---
layout: api 
page_title: /role - HTTP API 
description: The `/role` endpoint is used to configure roles
---

# `/roles`

The `/roles` endpoint is used to configure the key parameters how Vault should connect to Mashery V2 and V3 APIs. The
base path supports a standard list response, indicating the roles that are defined.

```shell
vault list mash-creds/roles
```

# `/roles/:roleName`

The `/roles/:roleName' endpoint is used to store the Mashery credentials and retrieve the capabilities of the role at
hand

## Get Role Capabilities

| Method  | Path                          |
|:--------|:------------------------------|
| GET     | `/mash-creds/roles/:roleName` |

### Parameters

- `roleName` `(string, <required>)` name of the role.

### Sample Request

**cURL**:

```shell
curl \
  --header 'X-Vault-Token: ...' \
  --request GET 'http://127.0.0.1:8200/v1/mash-creds/roles/sample'
```

**Vault CLI:**

```shell
vault read mash-creds/roles/sample
```

### Sample Response

```json
{
  "exportable": true,
  "forced_proxy_mode": false,
  "qps": 2,
  "term": "∞",
  "term_remaining": "∞",
  "use_remaining": "∞",
  "v2_capable": false,
  "v3_capable": false
}
```

## Create/Update Connectivity Configuration

| Method | Path                          | Purpose |
|:-------|:------------------------------| ------- |
| POST   | `/mash-creds/roles/:roleName` | Create  | 
| PUT    | `/mash-creds/roles/:roleName` | Update |

The operation creates a new role (with `POST` verb) or updates specified fields for `PUT` operation. For `PUT` operations,
fields omitted in the requested are left unchanged. To clear the field, specify an empty string explicitly.

### Parameters

- `roleName` `(string, <required>)` - name of the role.
- `area_id` `(string, "")` - Mashery Area identifier
- `area_nid` `(string, "")` - Mashery Area numeric identifier
- `api_key` `(string, "")` - api key
- `secret` `(string, "")` - secret part of the api key
- `username` `(string, "")` - Mashery developer portal login
- `password` `(string, false)` - Mashery developer portal password
- `qps` `(number, 2 unless other specified)` - QPS the application using these credentials needs to observe

Depending on the intended use, a subset of elements needs be provided as indicated in the table below.

| Field | Required for V2 API | Required for V3 API |
|-------------------|-----|-----|
| `area_id`         |     | Yes |
| `area_nid`        | Yes |     |
| `api_key`         | Yes | Yes | 
| `secret`          | Yes | Yes |
| `username`        |     | Yes |
| `password`        |     | Yes |
| `qps`             | Optional (defaults to 2) | Optional (defaults to 2) |

### Sample payload

```json
{
  "area_id": "a-b-c-d-",
  "api_key": "aaaa",
  "secret": "bbbb",
  "qps": 15
}
```

### Sample Request

**cURL**

```shell
curl \
  --header 'X-Vault-Token: ...' \
  --header 'Content-Type: application/json' \
  --request POST \
  --data @payload.json \
  'http://127.0.0.1:8200/v1/mash-creds/roles/sample' 
```

**Vault CLI**

```shell
cat payload.json | vault write mash-creds/roles/sample
```
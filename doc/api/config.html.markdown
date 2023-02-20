---
layout: api 
page_title: /config - HTTP API 
description: The `/config` endpoint is used to adjust configuration
---

# `/config`

The `/config` endpoint is used to configure the key parameters how Vault should connect to Mashery V2 and V3 APIs.

## Get Connectivity Configuration

| Method  | Path                |
|:--------|:--------------------|
| GET     | `/mash-auth/config` |

### Sample Request

**cURL**:

```shell
curl \
  --header 'X-Vault-Token: ...' \
  --request GET 'http://127.0.0.1:8200/v1/mash-auth/config'
```

**Vault CLI:**

```shell
vault read mash-auth/config
```

### Sample Response

```json
{
  "enable_cli_v3_write": false,
  "mashery issuer cert": "",
  "mashery leaf cert": "",
  "mashery root cert": "",
  "net_latency (effective)": "147ms",
  "oaep_label (effective)": "sha256:d7cd1ff4cd116846fb90cc0843490d5fef80c2f19352849dbb518d36cf080f31",
  "proxy_server": "",
  "proxy_server_auth": "",
  "proxy_server_creds": "",
  "tls_pinning (desired)": "default",
  "tls_pinning (effective)": "default"
}
```

## Update Connectivity Configuration

| Method | Path                 |
|:-------|:---------------------|
| PUT    | `/mash-creds/config` |

### Parameters

- `net_latency` `(string, "")` - average network latency between Vault's location and Mashery API. Defaults to 147
  millisecond for an empty value.
- `oaep_label` `(string, "")` - custom label for data exchange operations.
- `proxy_server` `(string, "")` - proxy server, via which the connection needs to be made
- `proxy_server_auth` `(string, "")` - proxy server authentication type, e.g. `Basic`
- `proxy_server_creds` `(string, "")` - proxy server authentication credential
- `enable_cli_v3_write` `(bool, false)` - whether to enable CLI write operations
- `tls_pinning` `(string, "default" | "system" | "custom")` - desired TLS pinning

### Sample payload

```json
{
  "enable_cli_v3_write": true,
  "net_latency": "12ms",
  "oaep_label": "dddd",
  "proxy_server": "proxy",
  "proxy_server_auth": "Basic",
  "proxy_server_creds": "user:password",
  "tls_pinning": "default"
}
```

### Sample Request

**cURL**
```shell
curl \
  --header 'X-Vault-Token: ...' \
  --header 'Content-Type: application/json' \
  --request PUT \
  --data @payload.json \
  'http://127.0.0.1:8200/v1/mash-creds/config' 
```
**Vault CLI**
```shell
cat payload.json | vault write mash-creds/config
```
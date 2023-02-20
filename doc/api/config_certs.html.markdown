---
layout: api 
page_title: /config/certs/:type - HTTP API 
description: The `/config/certs/:type` endpoint is used to adjust configuration
---

# `/config/certs`

The `/config/certs` endpoint is used to configure the certificate pinning configuration for leaf, issuer,
and root certificate



## Update Pinning Requirements for Certificate

| Method | Path                            |
|:-------|:--------------------------------|
| PUT    | `/mash-auth/config/certs/:type` |


### Parameters
- `type` `(string, <required>)` - type of the certificate to pin: `leaf`, `issuer`, or `root` corresponding to
  leaf, issuer, and root
- `cn` `(string, "")` common name of the certificate to pin
- `sn` `(string, "")` serial number of the certificate to pin
- `fp` `(string, "")` certificate fingerprint to pin

### Sample Payload
```json
{
    "cn": "la",
    "sn": "0a",
    "fp": "a0:ab:ac"
}
```

### Sample Request

**cURL**:

```shell
curl \
  --header 'X-Vault-Token: ...' \
  --header 'Content-Type: application/json' \
  --request POST 'http://127.0.0.1:8200/v1/mash-creds/config/certs/leaf' \
  --data=@payload.json
```

**Vault CLI:**

```shell
cat payload.json | vault write mash-creds/config/certs/leaf
```

## Clear Certificate Pinning Requirement

| Method | Path                             |
|:-------|:---------------------------------|
| PUT    | `/mash-creds/config/certs/:type` |

### Parameters
- `type` `(string, <required>)` - type of the certificate to pin: `leaf`, `issuer`, or `root` corresponding to
  leaf, issuer, and root

### Sample Request

**cURL**
```shell
curl \
  --header 'X-Vault-Token: ...' \
  --request DELETE \ 
  'http://127.0.0.1:8200/v1/mash-creds/config/certs/leaf' 
```
**Vault CLI**
```shell
vault delete mash-creds/config/certs/leaf
```
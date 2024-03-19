---
layout: docs
page_title: TIBCO Mashery V2/V3 Secrets Engine Grant
description: |-
  The Mashery V2/V3 Access Credentials secrets engine generates md5 signatures (for V2 Mashery API) and access/refresh
  tokens (for V3 Mashery API).
---

# Granting V2/V3 Access Credentials

The secret engine can generate V2 signatures and V3 access tokens that could be read and used by the administrators or
applications to authenticate to Mashery V2 and/or V3 API.

> Before using grant method &mdash; which must reveal Mashery API key and Mashery area's numeric identifier &mdash; consider if you 
> can make a use of [proxy mode](proxy_mode.html.markdown). 
> 
> There is **no documented way** to revoke a Mashery V3 access token. Be sure to understand that `lease``-type
> grants are implemented **only** to make this data cacheable in the Vault agent's cache. Revoking this leave 
> **will not** destroy Mashery token.

There are two flavours of accessing V3 access tokens:
- via `/grant` endpoint which will create a *new* access token *always*; and 
- via `/token` endpoint which will *cache* an access token as long as it is valid.

The difference between these two methods is that `/token` endpoint and will store and automatically refresh
the access token upon the expiry of the cache value. The cached token can also be deleted. Granted tokens
(that is, those created with `/grant` endpoint) are not tracked.

The `/token` endpoint can be favoured by a consuming application which is frequently restarted, such as e.g.
Terraform provider. In this case, using `/token` endpoint will minimize the number of access tokens created, 
or applications which want to avoid implementing stateful storage logic while being able to rotate the access
token during the running of the application.


## Usage

Prior to using the role, verify the capabilities of the role you are intending to use and the remaining term:
```shell-session
$ vault read mash-creds/roles/:roleName
```
This would output a brief similar to the following:
```shell-session
Key                  Value
---                  -----
exportable           true
forced_proxy_mode    false
qps                  2
term                 ∞
term_remaining       ∞
use_remaining        ∞
v2_capable           true
v3_capable           true
```

### Grant parameters (`/grant` endpoint)

This output defines which Mashery API this role can support and gives the overview of remaining usage.

> In case Mashery data has exceeded its term or number of time it's used, it will not be possible by design to
> retrieve Mashery credentials. Contact your _seeding administrator_ as [this guide](pull_mode.html.markdown) 
> describes to renew the grant.

The grant option access two optional parameters:
- `api` identifies the API version. Only `2` and `3` are valid options, with 3 being the default
- `lease` identifies whether to return the created secret as Vault [lease](https://www.vaultproject.io/docs/concepts/lease).
  This parameter can be meaningfully set to true only for caching purposes within the Vault's agent
  as [this guide](agent.html.markdown) explains.

### Token reveal parameters (`/token` endpoint)

### Using the CLI

To read the V3 access token using the CLI, execute:
```shell
vault read mash-creds/roles/:roleName/grant
vault read mash-creds/roles/:roleName/token
```
This will obtain Mashery V3 token (if the role is capable) and will produce the following output:
```shell
Key                     Value
---                     -----
access_token            access-token-value
expiry                  yyyy-MM-ddTHH:mm:ss+TZ:00
expiry_epoch            1704191132
qps                     2
token_time_remaining    1817
```

V2 access tokens are retrieved require specifying the `api=2` parameter:

```shell
vault read mash-creds/roles/:roleName/grant api=2
```
This will obtain Mashery V3 token (if the role is capable) and will produce the following output:
```shell
Key         Value
---         -----
api_key     your-mashery-api-key
area_nid    000
qps         2
sig         signature-string
```

### Using the API to retrieve V3 credentials

To retrieve V3 access token, invoke
```shell
curl --location \ 
     --request GET 'http://my-vault-server:8200/v1/mash-creds/roles/:roleName/grant' \
     --header 'X-Vault-Token: <token>' 
```

A successful response body will contain Vault API JSON object, bearing the `data` according to the following schema:
````typescript
class V3AccessTokenData {
    'access_token': string
    expiry: Date
    'expiry_epoch': number
    'token_time_remaining': number
    qps: number
}
````

A full response will be similar to the following:
```json
{
    "request_id": "96e973a8-3bcf-abca-233a-91a6d7e8c421",
    "lease_id": "",
    "renewable": false,
    "lease_duration": 0,
    "data": {
        "access_token": "access-token-value",
        "expiry": "yyyy-MM-ddTHH:mm:ss+TZ:00",
        "expiry_epoch": 1704193422,
        "token_time_remaining": 3600,
        "qps": 2
    },
    "wrap_info": null,
    "warnings": null,
    "auth": null
}
```

A Vault agent-**cacheable** V3 response can be retrieved using `lease=true` query parameter:
```shell
$ curl --location 
       --request GET 'http://localhost:8200/v1/mash-creds/roles/mcc/grant?lease=true' \
        --header 'X-Vault-Token: root'
```
The output will be essentially the same; with the difference that it will carry the lease information.
````json
{
    "request_id": "6d1602ff-9870-881f-1f14-bfac9efc31de",
    "lease_id": "mash-creds/roles/:roleName/grant/25Jg8BaclR2GEZlOaKICooBj",
    "renewable": true,
    "lease_duration": 900,
    "data": {
        "access_token": "access-token-value",
        "expiry": "yyyy-MM-ddTHH:mm:ss+TZ:00",
        "expiry_epoch": 1704193422,
        "token_time_remaining": 3600,
        "qps": 2
    },
    "wrap_info": null,
    "warnings": null,
    "auth": null
}
````
> Note that `lease_duration` is set to 15 minutes and the lease is marked as `renewable=true`. This is needed for
> caching purposes within the Vault agent. Mashery V3 token remains **active** when the lease has been revoked.

### Using the API to retrieve V2 credentials

To retrieve V2 access token, invoke
```shell
curl --location \ 
     --request GET 'http://my-vault-server:8200/v1/mash-creds/roles/:roleName/grant?api=2' \
     --header 'X-Vault-Token: <token>' 
```

A successful response body will contain Vault API JSON object, bearing the `data` according to the following schema:
````typescript
class V2SignatureData {
    'api_key': string
    'area_nid': number
    qps: number
    sig: string
}
````

A full response will be similar to the following:
```json
{
  "request_id": "231614a4-3155-6cd4-1d79-2b41b6fed5ea",
  "lease_id": "",
  "renewable": false,
  "lease_duration": 0,
  "data": {
    "api_key": "api-key",
    "area_nid": 0,
    "qps": 2,
    "sig": "signature-value"
  },
  "wrap_info": null,
  "warnings": null,
  "auth": null
}
```

A Vault agent-**cacheable** V3 response can be retrieved using `lease=true` query parameter:
```shell
$ curl --location 
       --request GET 'http://localhost:8200/v1/mash-creds/roles/mcc/grant?lease=true' \
        --header 'X-Vault-Token: root'
```
The output will be essentially the same; with the difference that it will carry the lease information.
```json
{
    "request_id": "c1609c94-b7b5-65c3-c951-47be08185f09",
    "lease_id": "mash-creds/roles/:roleName/grant/tvR4MNc5cQ3ULzWPcowHSZ3Q",
    "renewable": true,
    "lease_duration": 60,
    "data": {
        "api_key": "api-key",
        "area_nid": 0,
        "qps": 2,
        "sig": "signature-value"
    },
    "wrap_info": null,
    "warnings": null,
    "auth": null
}
```

## Using V2/V3 grants with Postman

In case your administrator team prefers using Postman, it is recommended to setup [Vault agent](https://www.vaultproject.io/docs/agent)
that will cache V2 and V3 tokens. Consult [this guide](agent.html.markdown) for the instructions.
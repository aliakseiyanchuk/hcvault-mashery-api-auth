---
layout: api 
page_title: /role/:roleName/grant - HTTP API 
description: |-
  The `/role/:roleName/grant` endpoint is used to grant a Mashery API access credentials that will be 
  visible to the applications
---

# `/roles/:roleName/token`

The `/role/:roleName/token` endpoint is used to retrieve a currently valid Mashery V3 access token that can be used
by the application. 

### Parameters

- `roleName` `(string, <required>)` - name of the role.
- `lease` `boolean` - whether to return this response as a Vault lease

### Sample Request

**cURL**:
Reading the currently valid access token:
```shell
curl \
  --header 'X-Vault-Token: ...' \
  --request GET 'http://127.0.0.1:8200/v1/mash-creds/roles/sample/token'
```
Deleting he currently valid access token:
```shell
curl \
  --header 'X-Vault-Token: ...' \
  --request DELETE 'http://127.0.0.1:8200/v1/mash-creds/roles/sample/token'
```

**Vault CLI:**
Reading the currently valid access token:
```shell
vault read mash-creds/roles/sample/token
```
Deleting he currently valid access token:
```shell
vault delete mash-creds/roles/sample/token
```

### Sample Response

See [in-depth grant documentation](../grant.html.markdown) for information about response types.
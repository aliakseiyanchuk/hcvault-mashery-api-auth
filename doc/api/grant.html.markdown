---
layout: api 
page_title: /role/:roleName/grant - HTTP API 
description: |-
The `/role/:roleName/grant` endpoint is used to grant a Mashery API access credentials that will be 
visible to the applications
---

# `/roles/:roleName/grant`

The `/role/:roleName/grant` endpoint is used to grant a Mashery API access credentials that will be
visible to the applications

### Parameters

- `roleName` `(string, <required>)` - name of the role.
- `api` `(number, 3 or 2)` - API version to receive grant. If omitted, `api=3` is assumed.

### Sample Request

**cURL**:

```shell
curl \
  --header 'X-Vault-Token: ...' \
  --request GET 'http://127.0.0.1:8200/v1/mash-creds/roles/sample/grant?api=3'
```

**Vault CLI:**

```shell
vault read mash-creds/roles/sample/grant
```

### Sample Response

See [in-depth grant documentation](../grant.html.markdown) for information about response types.
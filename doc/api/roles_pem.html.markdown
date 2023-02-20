---
layout: api 
page_title: /role/:roleName/pem - HTTP API 
description: |-
The `/role/:roleName/pem` endpoint is used to extract role's PEM-encoded certificate for encrypting
data exchanges
---

# `/roles/:roleName/pem`

The `/role/:roleName/pem` endpoint is used to extract role's PEM-encoded certificate for encrypting
data exchanges

### Parameters

- `roleName` `(string, <required>)` - name of the role.
- 'cn' `(string, "")` - administrator's (common) name to print in the output PEM configuration

### Sample Request

**cURL**:

```shell
curl \
  --header 'X-Vault-Token: ...' \
  --request GET 'http://127.0.0.1:8200/v1/mash-creds/roles/sample/pem?cn=MyNickname'
```

**Vault CLI:**

```shell
vault read mash-creds/roles/sample/pem
```

To read the PEM contens with `jq`, execute
```shell
vault read mash-creds/roles/sample/pem | jq -r .data.pem > file_in_file_system.pem
```

### Sample Response

```json
{
  "pem": "-----BEGIN MASHERY ROLE RECIPIENT-----\nCommon-Name: MyNickname\nNotAfter: 2022-01-26 01:03:32.902046 +0100 CET m=+18271.308561101\nRole: empty\n\nMIIFIjCCAwqgAwIBAgI[.....data......]\nRklIesflTu8SvNkjR6BZAxofjQSGtQ==\n-----END MASHERY ROLE RECIPIENT-----\n"
}
```


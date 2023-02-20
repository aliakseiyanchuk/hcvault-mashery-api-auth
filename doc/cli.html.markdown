---
layout: docs
page_title: TIBCO Mashery V2/V3 CLI
description: |-
Mashery secret engine allows interacting with Mashery via CLI using standard Vault commands.
---

# CLI Guide

The secret engine fully supports Vault cli-based operations for configurations and querying Mashery
objects. 

Logically, the CLI paths can be spread into four groups:
- secret engine configuration: concerns  [`/config`](./api/config.html.markdown) and [`/config/certs`](./api/config_certs.html.markdown)
  paths
- entering ([`/roles/:roleName`](./api/roles.html.markdown)) or import/exporting data between Vaults
  (see [pull mode](./pull_mode.html.markdown) data exchange documentation)
- executing V2 and V3 operations, `/roles/:roleName/v2` and `/roles/:roleName/v3` respectively.

> The secret engine supports `/roles/:roleName/proxy/v2` and `/roles/:roleName/proxy/v3` paths which produce
> output that is **not compatible** with the CLI expectations. These proxy URLs are meant to be directly
> consumed by the applications expecting a direct interaction with Mashery V2 and V3 API as [proxy mode](proxy_mode.html.markdown)
> guide explains.
 

## Interacting with V3 objects

Mashery V3 is a REST API that is directly operable with Vault's `read`, `write`, and `delete` commands.
The [V3 Resource](https://developer.mashery.com/docs/read/mashery_api/30/Resource_Hierarchy) is referred to
by the path remained after `/roles/:roleName/v3`. 

For example, assuming that the secret engine is mounted on `mash-creds`, and it has `sandbox` role 
configured, then the following command will retrieve basic information about a service:
```shell
vault read mash-creds/roles/sandbox/v3/services/aServiceId
```

The secret engine will automatically translate `write` command into `POST` or `PUT`, depending on 
whether the object already exists.

A V3 object can be deleted with `vault delete` command.

## Executing V2 Queries

The secret engine includes a basic support for executing V2 queries. `/roles/:roleName/v2` path accepts
two parameters:
- `method` that specifies a valid [V2 query method](https://developer.mashery.com/docs/read/mashery_api/20)
  (e.g. `object.query`), and
- call body that can be supplied either as:
  - a JSON input body, or
  - a single string parameter `query`

For repeated query operations, the method can be encoded as a path remainder after `/roles/:roleName/v2`.
For example to execute the `object.query` repeatedly the following command could be used:
```shell
vault write mash-creds/roles/docExample/v2/object.query query="select * from applications"
```

Most other operations will require a json object to be posted to the server. For example,
to execute `application.update` method via Vault, the administrator may need to prepare payload as follows:
```json
{
  "id": 1234,
  "name": "new_application_name"
}
```
Then the application with id `1234` can be updated:
```shell
cat v2_call.json | vault write mash-creds/roles/docExample/v2 method=application.update
```

To call the same V2 method with the method name encoded as part of the path, execute
```shell
cat v2_call.json | vault write mash-creds/roles/docExample/v2/application.update
```
---
layout: docs
page_title: TIBCO Mashery V2/V3 Secrets Engine Limitations
description: |-
Mashery secret engine has certain limitations regarding the setup and operation that the Vault adminsitrator
needs to be aware of
---

# Secret Engine Limitations

## Intended usage

The secret engine is intended to support administrator and application making infrequent API calls. Examples
of application include, but are not limited to, CI/CD scripts, incidental query tooling, or TIBCO
Cloud Mashery configuration updaters.

> The secret engine is _not_ intended to be used by applications making a lot of V3 calls directly.
> Although it _could_ handle a considerable number of call, each call incurs a considerable write 
> overhead.

Where Vault and this secret engine is a part of the application architecture that should achieve a
considerable use of Mashery V2 API, such applications are advised either to consider
[agent caching](agent.html.markdown) or implement a caching strategy within the application code's logic.

## CLI V3 API write-type operations are disabled by default

It is technically possible to for the secret engine to modify or delete V3 objects. Such operations
involve submitting large JSON objects and could eventually lead to a configuration error. 
The plugin _disables_ CLI write operations by default.

These could be enabled by the administrator as explained in the [setup guide](setup.html.markdown).

As an alternative, consider [Terraform provider for TIBCO Cloud Mashery](https://github.com/aliakseiyanchuk/mashery-terraform-provider)
to modify Mashery resources.

## Header availability in Proxy Mode

V2 and V3 API implementations return important information as part of their headers. These headers are 
not visible by default, following Vault's approach deny-by-default. Where your application relies on
availability of such headers, be sure to follow the corresponding section in the [setup guide](setup.html.markdown).

## `vault list` command limitation

Vault CLI provides the [`vault list`](https://www.vaultproject.io/docs/commands/list) command, which this
secret engine supports for V3 paths responding with object lists. Seeing that such lists could be quite
lengthy at times, the plug-in _does not_ include an option to fetch all objects. Additionally, the
`vault list` command does not allow providing filtering arguments, which limits the applicability of
this command for practical purposes.

The secret engine provides two alternative: reading lists and counting objects in the list.
> These alternatives apply only for paths that return object lists.

### Reading lists with filters
To list objects with filters, append `;list` suffix to the Mashery V3 resource path you want to retrieve. For
example, the following command would list all services with a specific number

```shell
vault read mash-creds/roles/:role/v3/services;list filter=revisionNumber:34
```

### Counting objects
To count the objects that batch the query, append the `;count` suffix to the Mashery V3 resource path
you want to count. For
example, the following command would count number of services having a specific revision number:
```shell
vault read mash-creds/roles/:role/v3/services;count filter=revisionNumber:34
```
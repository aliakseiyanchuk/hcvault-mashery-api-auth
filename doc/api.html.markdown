---
layout: docs
page_title: TIBCO Mashery V2/V3 Secrets Engine API
description: |-
The Mashery V2/V3 secrets engine API provides API for generating access tokens, v2 credentials, or executing
V2 or V3 calls from the Vault itself.
---

# API Structure
The overall structure of the secret engine API paths is:
```shell
mash-creds
├── /config
├   └── /certs
├       ├── /leaf
├       ├── /issuer
├       └── /root
└── /roles
    └── /:roleName
        ├── /pem     
        ├── /export     
        ├── /import          
        ├── /v2
        ├   └── <v2 methods, e.g. object.query, application.fetch>
        ├── /v3
        ├   └── <v3-resources: /services, /members, /applications, etc>
        └── /proxy
            ├── /v2
            └── /v3
```
Paths `/roles/:roleName/v2` and `/roles/:roleName/v3` are mean to support Vault CLI operations
[this guide](cli.html.markdown) explains. Note that the CLI has [limitations](limitations.html.markdown).

Path `/roles/:roleName/proxy/v2` and `/roles/:roleName/proxy/v3` support [proxy mode](proxy_mode.html.markdown)
which fully encapsulates managing Mashery API credentials. With proxy mode, the application needs to 
authentication to Vault using means configured by the administrator.

These endpoints are described in their corresponding pages:
- `/config` [documentation](./api/config.html.markdown)
- `/config/certs`[documentation](./api/config_certs.html.markdown)
- `/roles` [documentation](./api/roles.html.markdown)
- `/roles/pem` [documentation](./api/roles_pem.html.markdown)
- `/roles/export` [documentation](./api/roles_export.html.markdown)
- `/roles/import` [documentation](./api/roles_import.html.markdown)
- `/roles/grant` [documentation](./api/grant.html.markdown)
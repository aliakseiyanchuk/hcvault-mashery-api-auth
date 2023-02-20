---
layout: docs 
page_title: TIBCO Mashery V2/V3 Secrets Engine Proxy Mode 
description: |- The Mashery V2/V3 Access
Credentials secrets engine can proxy requests to the Mashery API, thus removing the needs to distribute the Mashery
secretes outside of Vault.
---

# Proxy Mode

The application interacting with the Mashery V2/V3 API can be designed to function in two modes:

- in a _direct mode_, where an application must source credentials for the V2/V3 API, or this secret engine will be
  used.
- in a _proxy mode_, where an application will be programmed to authenticate against Vault, and this secret engine will
  proxy the requests to the TIBCO Cloud Mashery V2/V3 APIs.

![Direct vs Proxy Modes](proxyMode.png)

## The problem the Proxy Mode solves

The _proxy mode_ solves several security and functional issues:

1. The _proxy mode_ allows encapsulating all access credentials to Mashery APIs, both long-term and short-term. This
   ensures that an application can _only_ access these APIs via Vault, which provides access enforcement and audit trail
   point.
2. The _proxy mode_ manages access token rotation automatically. A long-running application has to include the logic to
   request token renewal shortly before the access token expiry. Practice shows that developers may struggle with making
   the mechanism effective.
3. In _proxy mode_, the requests to the Mashery API will be smoothed to fit within the indicated QPS values. This
   simplifies the application's code that does not need to manage concurrency itself.

> The _proxy mode_ is originally developed to support [Terraform Mashery provider](https://github.com/aliakseiyanchuk/mashery-terraform-provider).
> In order for the Terraform to be able to make any changes to Mashery as part of the plan, an access token
> is needed to be present in the plan. However, the access token is read only when the execution is planned.
>
> This presents a timing problem: a speculative Terraform plans must completely execute within 60 minutes after it
> has been calculated. This timing window is too short for practical applications.
>
> The _proxy mode_ solves this problem for executing the speculative Terraform plans by removing the
> requirement for the Terraform Mashery provider to supply an access token. Instead, as long as the
> provider can authenticate to Vault, the V3 token will be obtained by this secret engine.

Another advantage of switching the application to the _proxy mode_ is to gain a possibility to leverage
the Vault's [policies](https://www.vaultproject.io/docs/concepts/policies) to tighten down the access
controls. For example, an administrator can limit service keys for which an application can create
OAuth access tokens in [createAccessToken](https://support.mashery.com/docs/read/mashery_api/20/oauth_supporting_methods/methods/createAccessToken)
method.

Consult the [API page](api.html.markdown) for the Vault paths this secret engine supports.

## Limitations

The _proxy mode_ is **not** suitable for applications that need to use fetch a complete list of Mashery collections and,
thus, these require `X-Total-Count` header to be present in the list responses. More information can be found in
the [limitations page](limitations.html.markdown).

## Mashery endpoint configuration changes

> Typographical convention
>
> The examples in this guide assume that:
> - the secret engine is mounted on path `mash-creds/`
> - the DNS name of the Vault server is `your-vault-server` 
> - the server listens on port `8200`
> The place where the administrator should specify role name is indicated using `:roleName` token. This __placeholder__
> needs to be replaced with the desired role name when entering the data into the actual Vault.

To make a use of _proxy mode_, the application needs to change the root Mashery URI:

| Mashery API Version | Direct API Endpoint                      | Proxy mode API Endpoint                                                 |
|---------------------|------------------------------------------|-------------------------------------------------------------------------|
| V2 API              | `https://api.mashery.com/v2/json-rpc/%d` | `https://your-vault-server:8200/v1/mash-creds/roles/:roleName/proxy/v2` |
| V3 API              | `https://api.mashery.com/v3/rest`        | `https://your-vault-server:8200/v1/mash-creds/roles/:roleName/proxy/v3` |

> Note that V2 APIs are scoped to the area, requiring the application to specify the area's numeric ID.
 
The following gives an example how to retrieve the [list of defined services](https://support.mashery.com/docs/read/mashery_api/30/resources/services)
within Mashery.
```shell
curl --location --request GET 'https://localhost:8200/v1/mash-creds/roles/docDemoRole/proxy/v3/services' \
      --header 'X-Vault-Token: <token-value>'
```

## Using Vault agent

For applications that cannot (or find it difficult) to implement Vault authentication method, HashiCorp
provides a [Vault agent](https://www.vaultproject.io/docs/agent) solution that can be configured to provide
automatic authentication. Consult [this guide](agent.html.markdown) explaining how to set up the Vault
agent with this secret engine.
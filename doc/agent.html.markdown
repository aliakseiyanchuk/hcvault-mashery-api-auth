---
layout: docs
page_title: Setting up Vault agent
description: |- 
  Mashery secret engine requires a user to authenticate. In some cases, it is unavoidable that authentication
  data will be replicated into a cloud storage
---

# Setting up Vault agent

The [Vault agent](https://www.vaultproject.io/docs/agent) is a client daemon that provide two useful features
for working with Mashery V2/V3 APIs via HasiCorp Vault: [auto-auth](https://www.vaultproject.io/docs/agent/autoauth)
and [caching](https://www.vaultproject.io/docs/agent/caching). An agent will perform Vault authentication
on behalf of the consuming application. 

![Invocation chain](agentConnDiag.png)

These features are useful in context where tools like Postman are used in daily practice.

## The problem agent deployment solves

The following illustrates a threat model how an information disclosure can occur for the Postman user:
![PostmanAttackVector](postmanAttackVector.png)
(For more information about STRIDE thread modelling technique that reveals this weakness, consider
[this excellent article by Martin Forwler](https://martinfowler.com/articles/agile-threat-modelling.html).)

In this scenario, a legitimate user is user a Vault server instance that is deployed in the cloud. However,
unknown to this legitimate user, an attacker was able to gain logon credentials to 
[Postman's Web interface](https://web.postman.co). Since the legitimate user will be saving Vault tokens
e.g. in environment variables &mdash; and given that these tokens would tend to be rather long-lived &mdash;
the attacker is able to impersonate a legitimate user for an extended period of time and extract
information from the TIBCO Cloud Mashery. 

By designing the Postman collections talking to local agent on the `localhost`, you ensure that even in the event
an attacker is able to login to your Postman workspace, the workspace will contain **absolutely no**
usable credentials that an attacker could use.

![NoAttackPossibility](brokenSpoofing.png)

There are two large steps that a developer running Postman against your Mashery V2/V3 API needs to do:
- setup a Vault agent, and
- add pre-request scripts to the collection.

These steps are explained below.

## File directory structure

For the purpose of this guide, the following directory structure is assumed:
````
$ current directory (where vault command will be run)
  \- agent
      \- agent.hcl
  \- policies
      \- agent-guide.policy
  \- secrets
       \-  agent-role-id.txt
       \-  agent-secret-id.txt
````


## Setting up Vault agent using AppRole

> This is short guide to get started with [Vault policies](https://www.vaultproject.io/docs/concepts/policies) 
> and [AppRole](https://www.vaultproject.io/docs/auth/approle) authentication method. It is not a replacement
> for reading the manuals suggested by HashiCorp.

The starting point assumes that there is a Vault role already created where Mashery credentials are stored.

### Create policy
You need to create a policy defining which [API path](api.html.markdown) need to be accessible to the
client. In this guide, a sample policy will assume that:
- access to `application.fetch` method granted to V2 API;
- read-only access needs to be granted to V3 APIs in the proxied mode;
- granting is allowed only for caching in Vault agent 

The policy reflecting this requirement, and _assuming_ that there are roles  `demoRoleV2` and
`demoRoleV3` created, should read:
```hcl
# Only application.fetch method is allowed
path "mash-creds/roles/demoRoleV2/proxy/v2/oauth2.fetchApplication" {
  capabilities = [ "update" ]
}

# Proxy mode allowed for V3 API in read-only mode
path "mash-creds/roles/demoRoleV3/proxy/v3/*" {
  capabilities = [ "list", "read" ]
}

# Grants are allowed only for V3 API, and lease is mandatory.
path "mash-creds/roles/demoRoleV3/grant" {
  capabilities = [ "read" ]
  allowed_parameters = {
    api = ["3"]
    lease=["true"]
  }
}
```
and should be saved in `policy/agent-guide.policy.hcl`

This policy needs to be deployed to the Vault:
```shell
vault write agent-guide-policy policy/agent-guide.policy.hcl
```
> Note: `agent-guide-policy` is a name used in this guide. As a Vault administrator, you can  
> select a role name that is the best suitable for the configuration you are managing

### Enabling AppRole for Agent Auto-Auth

With the policy in place, AppRole authentication method needs to be enabled. This step is done once per
cluster. T
```shell
vault auth enable approle
vault write auth/approle/role/agent-demoRole token_policies=agent-guide-policy
```
> `agnet-demoRole` is a role name chosen in this guide. As a Vault administrator, you can choose
>  a more descriptive name for the configuration you are managing.

### Pull Role Id and Secret Id

A role id and secret needs to be and saved for the use by agent in `.secrets/agent-role-id.txt` and
in `.secrets/agent-secret-id.txt` files.

```shell
vault read -format=json auth/approle/role/agent-demoRole/role-id \
  | jq -r .data.role_id > .secrets/agent-role-id.txt

vault write -format=json -f auth/approle/role/agent-demoRole/secret-id \
  | jq -r .data.secret_id > .secrets/agent-secret-id.txt
```

### Start the Vault agent

To start the agent, the following agent configuration file is _minimally_ required.
```hcl
pid_file = "./pidfile"

vault {
  address = "http://localhost:8200"
  retry {
    num_tries = 5
  }
}

auto_auth {
  method "approle" {
    config = {
      role_id_file_path="./secrets/agent-role-id.txt"
      secret_id_file_path="./secrets/agent-secret-id.txt"
      remove_secret_id_file_after_reading= true
    }
  }
}

cache {
  use_auto_auth_token = true
}

listener "tcp" {
  address = "127.0.0.1:8300"
  tls_disable = true
}
```
> Note: this file shows the example of how to start Vault agent with minimal configuration. Specifically,
> this configuration accepts unsecure connections over HTTP.
>
> The HTTPS configuration is omitted in this guide only because it introduces an extra level of
> complexity with setting up server certificates, which is not really relevant to the main objective
> of this guide. Once you've familiarized yourself with 
> Vault magnet configuration, you should consider addressing this point
> 
> To re-start the agent, you'll need to re-source secret using the following command:
> ```shell
> vault write -format=json -f auth/approle/role/agent-demoRole/secret-id \
>       | jq -r .data.secret_id > .secrets/agent-secret-id.txt
> ```

Now, the agent can be started with the following command:
```shell
vault agent -config=agent/agent.hcl
```
## Using Agent with Proxy Mode

To use V2 or V3 API with agent running, the application should invoke agent URI instead of 
Mashery API. 

| Mashery API Version | Direct API Endpoint                              | Proxy mode API Endpoint                                       |
|---------------------|--------------------------------------------------|---------------------------------------------------------------|
| V2 API              | `https://api.mashery.com/v2/json-rpc/{area-nid}` | `http://localhost:8300/v1/mash-creds/roles/:roleName/proxy/v2` |
| V3 API              | `https://api.mashery.com/v3/rest`                | `http://localhost:8300/v1/mash-creds/roles/:roleName/proxy/v3` |
> You will need to replace the `:roleName` token with the desired role. In this guide, these tokens are
> replaced with `demoRoleV2` and `demoRoleV3` for demonstrating V2 and V3 protocol use respectively.

In the example setup by this guide, the following command will retrieve the list
Mashery Services.

```shell
curl --location --request GET 'http://localhost:8300/v1/mash-creds/roles/demoRoleV3/proxy/v3/services'
```
> Note that with Agent enabled Vault authentication is not required. Mashery authentication is
> not required as well.

And the following command will execute the [object.query](https://support.mashery.com/docs/read/mashery_api/20/object/objectquery) 
method.

```shell
curl --location --request POST 'http://localhost:8300/v1/mash-creds/roles/demoRoleV2/proxy/v2' \
--header 'Content-Type: application/json' \
--data-raw '{
    "jsonrpc": "2.0",
    "method": "object.query",
    "params": [
        "select * from applications"
    ],
    "id": 1
}'
```
    ## Retrieving explicit V3 authentication grant with Postman

The proxy mode has [limitations](limitations.html.markdown). Where an administrator needs to interact with 
V3 API using Postman, an administrator can be granted the use of the V3 credentials and consume
these within Postman pre-process script. The technique to do requires programming a simple pre-request 
script that will retrieve the current credentials and store these in 
the [Postman variables](https://learning.postman.com/docs/writing-scripts/script-references/postman-sandbox-api-reference/#using-variables-in-scripts).
> To ensure that these values *WOULD NOT* be copied to your Postman cloud workspace (where these could
> be potentially attacked), do not declare these in collection or environment.

Mashery V3 authentication is using the Bearer authentication tokens. You need to introduce a 
variable `vaultV3Token` as a value of the Bearer token in the Authentication tab:
![AuthTabWithVariable](postmanTokenVar.png)

To retrieve the actual V3 access token value via the agent, a pre-request script needs to be configured:
```javascript
pm.sendRequest('http://localhost:8300/v1/mash-creds/roles/demoRoleV3/grant?lease=true', 
    function (err, response) {
        pm.variables.set('vaultV3Token', response.json().data.access_token)
    }
);
```
![PreReqScript](postmanTokenPreReqScript.png)
> Notice the `lease=ture` query string. It is *required* to have it set 

That that's all you need to do! You can start executing Mashery V3 calls without thinking too much 
about refreshing tokens. The work will be done for you by the Vault agent and the secret engine.

## Retrieving V2 Authentication with Postman

Mashery V2 API authentication parameters are retrieved with a pre-request script similar to the follwoing one:
```javascript
pm.sendRequest('http://localhost:8300/v1/mash-creds/roles/demoRoleV2/grant?api=2&lease=true', function (err, response) {
    pm.variables.set('vaultV2Key', response.json().data.api_key)
    pm.variables.set('vaultV2AreaNID', response.json().data.area_nid)
    pm.variables.set('vaultV2Sig', response.json().data.sig)
});
```
The URL to which the request should be posted should be set in Postman to
`https://api.mashery.com/v2/json-rpc/{{vaultV2AreaNID}}?apikey={{vaultV2Key}}&sig={{vaultV2Sig}}`

![PostmanGrantV2](postmanV2Grant.png)
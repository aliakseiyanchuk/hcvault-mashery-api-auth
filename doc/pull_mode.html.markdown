---
layout: docs 
page_title: TIBCO Mashery V2/V3 Secrets Engine Pull Mode Configuration 
description: |- 
The Mashery V2/V3 Access Credentials secrets engine facilitates safe exchange of long-term Mashery credentials,
aiming to remove the possibilities to have these exposed during the hand-over and/or exchange between
the team members managing TIBCO Could Mashery.
---

# Sharing Mashery Keys in Pull Mode

The Mashery credentials are obtained from the [TIBCO Cloud Mashery API developer portal](https://developer.mashery.com/)
. An organization may wish to limit the access to this portal for obvious security reasons. For the purpose of this
guide, administrators that can access to this developer portal are termed _seeding administrators_.

Within organization, there will eventually be multiple applications and/or administrators that require access to Mashery
API credentials to perform their legitimate functions. Organizations that have adopted Vault, can trivially ask a 
_seeding administrator_ to enter the Mashery API credentials in Vault and manage the Vault policies granting access to
these credentials.

## The problem this flow solves

As long as all access to Mashery API can be controlled from a single Vault server or cluster, there is no need to share
Mashery credentials as the access to Mashery API can be fully encapsulated by the [proxy mode](proxy_mode.html.markdown). 
As the organization's API programme expands, a situations may arise that require replicating Mashery credentials to
other Vault clusters. For example, where an organization is making a high use
of [TIBCO Cloud Mashery API OAuth2 support methods](https://support.mashery.com/docs/read/mashery_api/20/OAuth_Supporting_Methods)
, creating a satellite Vault instance closer to the application could help improving the response time and manage the
load on the central Vault server or cluster.

The challenge that the organization now needs to solve is how a _seeding administrator_ should transfer the necessary
mashery credentials to another _satellite_ Vault instance in a secure way? If a _seeding administrator_ can connect to 
the _satellite_ Vault directly, then the necessary credentials can be entered e.g. using Vault's CLI. However, this may 
not be possible e.g. for the deployment where the Vault can only receive connection from an application or due to the
requirements to segregation of duties.

In this situation, the _seeding administrator_ needs to hand-over credentials to the administrator responsible for the
_satellite_ Vault server. This administrator is termed _line administrator_ in this guide.

The solution this secret engine offers is to transfer the necessary credentials in the encrypted format, ensuring
confidentiality and integrity of the data received.

## The Flow

The flow depicted by the picture below has seven steps and involves two roles: a _seeding administrator_ and a
_line administrator_ responsible for managing the _satellite_ Vault server.

![Pull Flow](./pull-mode.png "Pull mode flow")

> Typographical convention 
> 
> The examples in this guide assume that the secret engine is mounted on path `mash-creds/`.
> The place where the administrator should specify role name is indicated using `:roleName` token. This __placeholder__
> needs to be replaced with the desired role name when entering the data into the actual Vault.

### Step 1. Enter Mashery API data into the central Vault

> This step needs to be done once, when the credentials are entered into the central vault.

The sharing flow always starts with the _seeding administrator_ entering the credentials obtained from the TIBCO Cloud
Mashery API Developer Portal into the (organization's centralized) Vault server. This can be achieved by entering the
necessary values from the command line.

```shell
vault write mash-creds/roles/:roleName area_id="value" area_nid=9999999 \
  api_key="key" secret="secret" username="userName" password="password" 
```

> Although Vault CLI supports this, **NEVER DO THIS** in practice! These values are logged into your shell history. So
> if your machine is compromised and your shell history is read, the credentials will be exposed.

A more secure way is either to place sensitive values of `secret`, `username`, and `password` into separate files:

```shell
vault write mash-creds/roles/:roleName area_id="value" area_nid=9999999 \
  api_key="key" secret=@secret_file.txt username=@username.txt password=@password.txt 
```

or store all files in a JSON objects and ask Vault to read the parameters from the input.

```json
{
  "area_id": "area-aid",
  "api_key": "api-key",
  "secret": "secret",
  "username": "user-id",
  "password": "password",
  "area_nid": 99999999999999
}
```
```shell
cat mash-creds.json | vault write mash-creds/roles/:roleName -
```

### Step 2: Crete an empty role at _satellite_ Vault

A _line administrator_ will create an empty role in the _satellite_ Vault:
```shell
vault write -f mash-creds/roles/:roleName
```

### Step 3: Obtain recipient role certificate

After the role has been created, a _line administrator_ can extract role's identity certificate that will be used to
encrypt the credentials between the Vaults in transit.

> Note: in this guide, we assume that a direct connection between Vaults is not possible, either technically or
> due to regulatory requirements. The administrators are asked to follow a split-channel delivery.

The certificate is extracted with the following command:
```shell
vault read -format=json mash-creds/roles/:roleName/pem | jq -r .data.pem > role.pem
```
The command takes and optional key `cn` that encodes a common name of the _satellite_ Vault administrator.

The output of this command will produce a **short-lived** certificate, that would be similar to the one shown below. 

```
-----BEGIN MASHERY ROLE RECIPIENT-----
Common-Name: Site Admin
NotAfter: yyyy-MM-dd HH:mm:ss.000000 +0100 CET m=+14440.864005001
Role: <role name>

MIIFIjCCAwqgAwIBAgIEYdroHDANBgkqhkiG9w0BAQsFADBCMSswKQYDVQQKEyJN
[... multiple PEM-encoded lines ...]
20ntouNQ39m5T39oqQw1c6UUKhKmJw==
-----END MASHERY ROLE RECIPIENT-----
```

### Step 4. Transfer the role recipient to _seeding administrator_

The _line administrator_ can transfer this file to the _seeding administrator_ using any channel deemed appropriate. 
This identity file is actually a public key, so this file can travel over unsecure communication lines.

### Step 5. Export role's data from central Vault

Once the public key is received, the _seeding administrator_ can choose which data should be granted for the particular
_satellite_ Vault, and **for how long**. The _seeding administrator_ can:
- choose whether export will allow only V2, or V3 or both Mashery APIs;
- optionally, establish an _explicit term_ for the exported data;
- optionally, establish an _explicit number of uses_  the Mashery API credentials can be accessed.

When both _explicit term_ and _explicit number of uses_ are specified, the _satellite_ Vault will stop any grants
whenever a first condition is achieved.

The export command further allow the _seeding administrator_ to limit the export scope in lines with the
intended business use of the exported data in the _satellite_ Vault:
- `explicit_term` specifying an explicit term for the validity of the export. The following formats are supported:
   - `{NN}d` number of days, where `{NN}` indicates the desired number. For example, `10d` will export the data for the
      recipient to use for 10 calendar days;
   - `{NN}w` number of weeks, where `{NN}` indicates the desired number.
   - `yyyy-MM-dd`, a year-month-date expiration time. For example, `2022-06-07`, indicating that the grant will expire
      on the 0th UCT hour on the 7th of June 2022. 
   - Any valid Go language [duration expression](https://pkg.go.dev/time#ParseDuration)
- `explicit_num_uses` specifies number of times the exported data can be accessed within the receiving Vault;
- `explicit_qps` specifying the QPS to be observed. If omitted, role's own QPS is used in the export
- `v2_only` limits the export only to V2 credentials. The recipient will be unable to use V3 API
- `v3_only` limits the export only to V3 credentials. The recipient will be unable to use V2 API
- `force_proxy_mode` will prohibit the _satellite_ Vault to [grant credentials explicitly](grant.html.markdown), thus
   forcing the applications to talk to Mashery API only via the Vault server in [proxy mode](proxy_mode.html.markdown)

After the seeding administrator has established the desired export option, the following command should be used to 
export the role data:
```shell
vault write -format json math-auth/roles/:roleName/export pem=@identity_file <other options> | jq -r .data.pem
```

An example of full copy for an indefinite period of time will produce a PEM output that will be similar to the following
(note that `∞`  symbol indicates infinity):
```shell
-----BEGIN MASHERY ROLE DATA-----
Date: 2022-01-09 17:18:33.0402062 +0100 CET m=+59.970009501
Forced Proxy Mode: false
Max QPS: 2
Origin Role: <original role>
Recipient: CN=Site Admin,O=Mashery API Authentication Backend
Recipient Role: demoRole
Term: ∞
Uses: ∞
V2 Capable: true
V3 Capable: true

DdZ30ItmNODFaiJCFAwjo1QMbe23mOufnh7vF76/d099WP0+PsjGeBS8mVxCIpHW
[ ... more PEM lines ... ]
VgQeHIvGiLXYvgfC3GHiQ+rT/RWjgGjJsHHQUkFsAiI=
-----END MASHERY ROLE DATA-----
```

An example of time- and use-bound, V2-only in proxy mode export can be achieved with the following command:
```shell
vault write -format json math-auth/roles/:roleName/export pem=@identity_file \
 explicit_term=2w  explicit_num_uses=5000 explicit_qps=5 v2_only=true \
 force_proxy_mode=true | jq -r .data.pem
```
should produce an output that will be similar to the following:
```
-----BEGIN MASHERY ROLE DATA-----
Date: 2022-01-09 17:30:31.8104098 +0100 CET m=+778.740213101
Forced Proxy Mode: true
Max QPS: 5
Origin Role: <original role>
Recipient: CN=Site Admin,O=Mashery API Authentication Backend
Recipient Role: demoRole
Term: 336h0m0s
Uses: max 5000 uses
V2 Capable: true
V3 Capable: false

HaaAaeSQ2SL39GSPzbgYqO/Jzx5aT0lbfHELKFXdhRabihXTljS05V6d/Gt3Y+oa
[ ... more encoded PEM lines ... ]
+bcov68qaP9eAgF1ZHXhS36piDCCeCW/IwSLqkph2C0=
-----END MASHERY ROLE DATA-----
```

### Step 6: Transfer the encrypted file to _line administrator_
The file obtained in the previous step is an RSA-encrypted JSON file. This file is additionally secured with an OAEP
label, which offers an additional level of protection for the data in transit.

The _seeding administrator_ can send this file via any channel deemed sufficiently safe.

### Step 7. Importing data into _satellite_ Vault
Once the export data has been received, the _line administrator_ can import the settings via the CLI command:
```shell
vault write mash-creds/roles/:roleName/import pem=@out\demoRoleImport.pem
```
The effect of importing can be verified by checking success of the data import:
```shell
vault read mash-creds/roles/:roleName
```
which should produce an output similar to this:
```
Key                  Value
---                  -----
exportable           false
forced_proxy_mode    true
qps                  5
term                 23 Jan 22 19:45 CET
term_remaining       335h59m58.8888023s
use_remaining        4999 times remaining (0% used)
v2_capable           true
v3_capable           false
```

## Next steps

Congratulations! Your _satellite_ Vault has been configured and is ready to use. Depending on the desired
usage, you can:
- [setup Vault agent](agent.html.markdown) and configure automatic logon to Vault;
- [read Mashery credentials](grant.html.markdown) and use these directly
- [use the proxy mode](proxy_mode.html.markdown) to invoke Mashery APIs from Vault directly.
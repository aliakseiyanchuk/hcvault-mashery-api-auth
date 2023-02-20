# Mashery V2/V3 Access Credentials Secrets Engine for HashiCorp Vault

[TIBCO Mashery](https://www.tibco.com/products/api-management) is an API management platform. The platform
supports [programmatic access and configuration definition](https://developer.mashery.com/docs/read/mashery_api) that
requires two distinct authentication mechanisms. Mashery V2
uses [timestamp-salted MD5 hashes](https://developer.mashery.com/docs/read/mashery_api/20/Authentication)
while V3 api is using [access tokens](https://developer.mashery.com/docs/read/mashery_api/30/Authentication).

Both authentication schemes use a set of long-term credentials that an organization needs to manage with extreme
care to avoid security incidents.

## The problem this secret engine solves

The logon credentials for Mashery V2 may need be used by the applications continuously. The V3 API credentials 
may be required by the administrators and CI/CD pipelines one a daily basis to adjust Mashery configurations.
> The number of applications that require legitimate access to the Mashery APIs grows with the organization's
> API programme scale

The challenge to distribute these keys to the applications and administrators and _effectively_ rotate these without a
significant downtime scales with the API programme complexity. 

If an organization has adopted the HashiCorp Vault already, then, theoretically, the standard
[key-value](https://www.vaultproject.io/docs/secrets/kv/kv-v2) could be seen as a sufficient solution to distribute 
these secrets to the trusted applications and/or users. This, however, may not be enough in practice. 

If an organization is using cloud-based tools, weakly secured cloud-based logon procedures to the tool could be used
as an attack vector if a logon credentials to the tools' cloud storage is compromised (e.g. due to re-using a  password 
across multiple wed sites).

An application may _inadvertently_ write these into log file; users may enter these into share workspace environment 
variables. Or the users may leave your organization and still have active keys Mashery V2/V3 credentials copied from 
K/V store on their device.

This secret engine improves the security posture with managing the Mashery API by ensuring that an application,
even if granted access, will  _never_ know:
- Mashery key secret;
- Mashery administrator username;
- Mashery administrator password.

Where maximum discretion is sought, this secret engine can invoke the Mashery API on behalf of an authenticated and
authorized Vault user. This is referred to as _[proxy mode](./doc/proxy_mode.html.markdown)_. In _proxy mode_, an application or an administrator will 
have _absolutely no knowledge_ of Mashery credentials. This extra protection degree comes at imposing the
 [several limitations](./doc/limitations.html.markdown) to the proxy mode.

Ultimately, this secret engine solves three problems:
- how to share logon credentials between applications and administrators in a secure way;
- how to ensure need-to-know and need-to-use principles when working with Mashery APIs;
- how to control the period for which the credentials are granted, so that applications and administrators will be
  _forced_ to re-validate the need of their access to the Mashery API.

## How-To Guides

1. [Getting started](./doc/setup.html.markdown) explaining how to install this secret engine on a Vault server 
   and enter the minimum required data.
2. [Pull-mode Mashery credentials distribution](./doc/pull_mode.html.markdown) explains the mechanism how the team 
   responsible for managing Mashery can securely share credentials.
3. [Using Vault CLI](./doc/cli.html.markdown) explains how to use vault native CLI to execute basic queries and
   updates of Mashery objects
4. [Accessing Mashery API in Proxy Mode](./doc/proxy_mode.html.markdown) explains how an application can access Mashery
   objects via Vault.
5. [Configuring Vault agent. Using Postman](./doc/agent.html.markdown) explains how to set up Vault agent for managing 
   the Vault access tokens automatically. Additionally, the guide explains how to configure the proxy caching 
   for use with the Postman application.
6. [Granting access](./doc/grant.html.markdown) to the administrators and applications to read the access tokens
   and signatures that could be consumed by an administrator or an application.

# API Documentation

The complete API documentation of this secret engine can be [found here](./doc/api.html.markdown).

# Build from source code

Building from sources requires go 1.18 or later and make utility installed.
```text
$ make vendor release
```
For Windows-based machines, [Cygwin](https://www.cygwin.com/install.html) provides a working
implementation of make tool. Alternatively, file `compile_win_amd64.bat` provides an option
to build Windows-only executable.

# Testing

Testing the plugin is split into two stages: unit test and acceptance testing on a running
development server

## Running unit tests

Run unit tests using
```shell
$ go test ./mashery
```
from the project directory. The unit testing focuses on

## Running acceptance test

The acceptance tests are based on [Cucumber](https://cucumber.io/) test scenarios located in the
[`features`](./features) directory. The tests are run against the development server that needs
to be launched before the tests are run. 

On systems supporting `make`, this is best achieved by running 
```shell
$ make testacc
```
On Windows, this needs to be two-step process:
```shell
% launch_dev_mode.bat
% go test ./bdd -v
```
Note `-v` flag will allow printing Cucumber scenario steps. 

# Installing and running

Consult [reference manual readme](./doc/index.html.markdown) for information operational limitations and installation
steps.


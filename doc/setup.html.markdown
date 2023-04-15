---
layout: docs
page_title: TIBCO Mashery V2/V3 Secrets Engine Installation Guide
description: |-
The guide describes the Vault-specific installation steps that are required to install this plugin
---

# Installation Guide

To make this secret engine available to Vault runtime, a platform-specific binary needs to be placed into
the Vault's plugins directory. This is a directory on a file system that Vault can read from.

Due to a very versatile nature of Vault (as well as options how to build machine's file systems), there
can be multiple options required to set up the secret engine with the site at hand. This guide given a 
quick overview of the steps and the references to the Vault documentation.

## Testing secret engine in dev server mode

Vault server includes the [dev server mode](https://www.vaultproject.io/docs/concepts/dev-server) that
allows experimenting with Vault features. In this
mode, the storage is in-memory, and no changes are persisted. The consideration of this mode is that
it is started with `HTTP` where a default Vault CLI expects `HTTPS` support. This can be addressed either
by setting `VAULT_ADDR=http://localhost:8200` or specifying `-address=http://localhost:8200/` for Vault
commands.

Assuming your CLI environment defines variable `DEV_PLUGINS_DIR` containing the compiled secrets engine
and `MASH_AUTH_DEV_BINARY` specifying the executable located in this directory, 
the following command sequence will start up a development server and mount the binary under `mash-creds`
Vault path:
```shell
vault server -dev -dev-root-token-id=root -dev-plugin-dir=${DEV_PLUGINS_DIR} -log-level=trace &
# Let vault start-up before attempting mouring	
sleep 5
echo root | vault login -address=http://localhost:8200/ -
vault secrets enable -address=http://localhost:8200/ -path=mash-creds \
              -allowed-response-headers="X-Total-Count" \
              -allowed-response-headers="X-Mashery-Responder" \
              -allowed-response-headers="X-Server-Date" \
              -allowed-response-headers="X-Proxy-Mode" \
              -allowed-response-headers="WWW-Authenticate" \
              -allowed-response-headers="X-Mashery-Error-Code" \
              -allowed-response-headers="X-Mashery-Responder" \
              ${MASH_AUTH_DEV_BINARY}
```
This sequence is also available in [Makefile](../Makefile) as `launch_dev_mode` target.

> Note that in the dev test mode Vault doesn't automatically assign aliases, such as 
> `mashery-api-creds` that is used in the production installation steps below. Unless you will specifically
> call `plugin regsiter` command, you will need to use the name of the executable to enable the
> secret engine for testing.

## Packaging complied secret engine for production use

Whatever the actual Vault VM installation method, the administrator needs to provision an executable of the
secret engine in a format that would be suitable for the operating system of the VM. As a next step, the
administrator needs to supply `plugin_directory` parameter in the [Vault configuration file](https://www.vaultproject.io/docs/configuration).

```hcl
listener "tcp" {
  # Listener configuration
}

storage "<desired type>" {
  # storage configuration
}

default_lease_ttl = "786h" # Default, override as required
max_lease_ttl = "7860h"    # Default, override as required

#  The key that specifies where to load the data from.
plugin_directory = "/opt/vault/plugins"
```

### mlock considerations

Where Vault application starts with MLock enabled, the administrator must ensure that the plugin binaries
are also mlock-enabled by running the following command:

```shell
setcap cap_ipc_lock=+ep /opt/vault/plugins/*  
```
> If you have multiple binaries here, consider using explicit path to the secret engine executable.

The mlcok status is printed by Vault at startup. For example, on Linux systems these would be printed
in the `/var/log/syslog` file. 

In case mlock **is** enabled, but mlock **is not** set for the execution binary, then an `Unrecognized remote plugin message`
will appear when the mount will be attempted. Be sure to enable the mlock and try again.

### API address considerations
If your Vault system is directly responding to the connections, you may need to take extra steps to make
sure the system trusts the Vault TLS certificates. 
> This is especially true if you run your own CA and vault is providing the your own CA's certificates to the
> clients.

This requirement may appear strange, but it has to do with the nature of secret engines. Secret engines
are actually processes that communicate with Vault over https. For production machine, it is strongly 
recommended ensuring that also internal communications run over https. Consider disabling internal https
only where appropriate.

You need to do three steps:
1. From the machine where Vault is deployed, execute
  ```shell
   curl https://your-vault-machine-dns-name:8200/
  ```
  If your machine trusts the certificate, then no further certification configuration is needed.
  > Note that `your-vault-machine-dns-name` **must** match the certificate's common name or one fo the SAN
  > (subject alternative name) listed in the certificate.
  > 
  > Also note that it is _unusual_ for the certificates to specify `localhost` as a common name or SAN , 
  > so `https://localhost:8200`, most likely, won't be a feasible option.
2. If you can't get the `curl` command to trust the certificate, then you need to add the certificate authorities
  to the trusted certificate storage of the operating system. Method vary, depending on the operating system
  at hand.
3. Once you get `curl` command working, you need to set `api_addr` in the vault configuration as follows:
```hcl
api_addr = "https://your-vault-machine-dns-name:8200"
```

### Installation for VM
Where the administrator has installed vault on the VM, the configuration file may need to be edited. The
locations vary depending on the installation specifics. On Linux systems, the configuration file would be typically 
found in `/etc/vault.d/` directory, such as `/etc/vault.d/vault.hcl`

> A hint: `find  / -name '*.hcl'` could be used to locate all HCL files on the machine.

Make sure to include `api_addr` and verify that the processes on Vault machine trust the connection
as the above explains.

### Dockerized installation

Vault provides [docker container](https://hub.docker.com/_/vault/) which contains the pre-compiled 
version of Vault. As Vault administrator, along with making changes necessary packaging this secret engine,
you may want to modify this configuration as required for your environment.

The following gives an overview how to add secret engine to a bare container. This guide will assume the 
following directory structure:
```shell
$ tree
├── Dockerfile
├── config.hcl
├── docker-compose.yaml
└── mashery-api-creds_<version>
```
> Source code for the sample configuration can also be found in [this directory](../docker).

The `Dockerfile` adds a directory where the built plugin will be stored and copies the binary there.
> There are two important permission settings:
> 1. The binary must be world-executable, as vault actually runs under `systemd+` user; and
> 2. The binary must be mlock-enabled; otherwise Vault will not load it.   
```dockerfile
FROM vault:latest

CMD [ "vault", "server", "-config=/vault/config" ]

RUN mkdir -p /vault/plugins
# The executable should be available in the root directory.
COPY ./mashery-api-creds* /vault/plugins
RUN chmod a+x /vault/plugins/* && setcap cap_ipc_lock=+ep /vault/plugins/*

# You may want to budnle additional resources, such as SSL certificates
COPY ./config.hcl /vault/config

```

Similar to VM-based installation, the dockerized installation required supplying the correct `api_addr`
and `plugin_directory` properties in the configuration files. Similar to VM-based installations, these 
values depend on the network topology and considerations.
> Depending on the deployment topology, it may be acceptable to use http protocol in `api_addr`. The added
> value of this configuration is that it removes lots of certificate common name/subject alternative
> name issues.

```hcl
# Traffic listener, that is handling the traffic. Configure as desired.
listener "tcp" {
  # Configure with SSL support as desired.
}

# This is a loop-back listener that will the secret engine process will use to communicate
# with the Vault process.
listener "tcp" {
  address     = "127.0.0.1:8973"
  tls_disable = "true"
}

storage "file" {
  # Configure as desired
}

# Tрe crux of the installation process: make sure you have a correct plugin version
# copied here.
plugin_directory = "/vault/plugins"
api_addr = "http://127.0.0.1:8973"

# Standard lease settings. Modify as required for your deployment
default_lease_ttl = "786h"
max_lease_ttl = "7860h"
```

### Starting dockerized installation

To persist secrets and audit logs, the storage and log files need to be written to Docker volumes, otherwise
there will be lost upon container restart. The production configuration requires mlock support to prevent
swap being written to disk. To make sure the container is repeatedly restarted with correct configuration,
it is recommended to create a docker-compose and start the container with `docker-compose`

```yaml
version: '3.7'

services:
  vault-mash-creds:
    container_name: vault-mash-creds-container
    hostname: vault-mash-creds
    # Add memory lock capability
    cap_add:
      - IPC_LOCK
    # Persist files and (audit) log files between container restarts
    volumes:
      - "vault_file:/vault/file"
      - "vault_logs:/vault/logs"
    
    # Export port 8200 on the host; other ports will not be reachable.
    ports:
      - target: 8200
        protocol: tcp
        published: 8200
        mode: host
    build:
      context: .
      dockerfile: ./Dockerfile

    command: [ "vault", "server", "-config=/vault/config" ]

volumes:
  vault_file:
  vault_logs:

```

## Mounting

> In a non-dev server mode, Vault *will not* automatically trust a binary in the plugins directory.

Mounting the secret engine comprises two steps: adding the secrets' engine to the available plug-in back-ends
list. This needs to be done once. Then, the plugin can be mounted on multiple access paths as desired.

### One-time registration

The secret engine is registered with [`plugin regsiter`]( https://www.vaultproject.io/docs/commands/plugin/register)
command that requires calculating a SHA256 signature of the binary. Below is the example script that
is making a use of `openssl` and `awk` CLI commands:

```shell
REF_BINARY="path-to-binary"
SIG=$(openssl dgst -sha256 $REF_BINARY | awk -F'= ' '{print $2}')
openssl dgst -sha256 ${REF_BINARY} | awk -F'= ' '{print $2}'
vault plugin register \
   -sha256=$SIG \
   -command=$(basename $REF_BINARY) \
   secret mash-creds
```

The installation of the result can be verified by running [`plugin list`](https://www.vaultproject.io/docs/commands/plugin/list)
command, which should indicate `mash-creds`.

### Minimal mounting secret engine on path

After registration is complete, the Vault administrator can mount the desired number of secret engine
instances using [`secrets enable`](https://www.vaultproject.io/docs/commands/secrets/enable) command.

The minimal command version requires only desired path, such as the following:
```shell
vault secrets enable -path=mash-creds mash-creds
```
> This minimal configuration suppresses Mashery API headers in the proxy model. In case applications 
> using this mount need to receive these headers, the mount needs to specify the list of these
> headers explicitly. These are described in the next section.

### Verifying success

To verify that the mount is successful, try reading the default configuration which should yield the
output similar to the following:
```shell
vault read mash-creds/config
Key                        Value
---                        -----
enable_cli_v3_write        false
mashery issuer cert        n/a
mashery leaf cert          n/a
mashery root cert          n/a
net_latency (effective)    147ms
oaep_label (effective)     sha256:d7cd1ff4cd116846fb90cc0843490d5fef80c2f19352849dbb518d36cf080f31
proxy_server               n/a
proxy_server_auth          n/a
proxy_server_creds         n/a
tls_pinning (desired)      default
tls_pinning (effective)    default
```
> If you are unable to retrieve this value, it means that Vault and secret engine plugin cannot 
> communicate. Consider if:
> - CA certificates are installed on the machine and are actually added to the correct root store
> - Installed certificates have common name and/or SAN that matches the `api_addr` value
> - Consider other error messages in the system log files.
>
> Unfortunately, no further information can be given within this guide.

### Allowing Mashery Headers in Vault Responses
V2 adn V3 Mashery API will send headers that a consuming application may find useful. These headers are:
- `WWW-Authenticate`
- `X-Error-Detail-Header` and `X-Mashery-Error-Code`, indicating Mashery-specific error. The presence of
  these headers helps to establish whether the call has actually reached Mashery
- `X-Mashery-Responder` indicating instance
- `X-Total-Count` indicating number of objects in the list response that have matched the query parameter.

The secret engine will add the following headers:
- `X-Proxy-Mode` indicating that the call is being processed in the proxy mode;
- `X-Server-Date` indicating the date of the response according to the Mashery's clock. The value indicated
  by this header should differ several seconds from that indicated by `Date` header. If the difference is
  larger, then sporadic authentication failures may occur as a result of the clock skew.

> These headers are not added by Vault automatically. As a Vault administrator, you need to specifically
> enable the desired headers.

The headers need to be enabled either as part of [`secrets enable`](https://www.vaultproject.io/docs/commands/secrets/enable)
or [`secrets tune`](https://www.vaultproject.io/docs/commands/secrets/tune) command passing
each of the desired headers as `-allowed-response-headers`. The following givens the example of the mounting
command that enables all headers:
```shell
vault secrets enable -path=mash-creds \
    -allowed-response-headers="X-Total-Count" \
    -allowed-response-headers="X-Mashery-Responder" \
    -allowed-response-headers="X-Server-Date" \
    -allowed-response-headers="X-Proxy-Mode" \
    -allowed-response-headers="WWW-Authenticate" 
    -allowed-response-headers="X-Mashery-Error-Code" \
    -allowed-response-headers="X-Mashery-Responder" \
    mashery-api-auth
```

## Define policy
After the engine is mounted, the Vault administrator should define a [Vault policy](https://www.vaultproject.io/docs/concepts/policies)
that controls access to individual paths. The secret engine has the following path structure:
```shell
mash-creds
├── /config 
└── /roles
    └── /:roleName
        ├── /grant     
        ├── /v2
        ├── /v3
            └── <v3-resources: /services, /members, /applications, etc>
        └── /proxy
            ├── /v2
            └── /v3
```
> Typographical convention
> Token `:roleName` refers to a role name that the administrator will assign to a particular credentials
> For example, `ci_cd_test`, `interactive_live`, etc.

[API specification](api.html.markdown) explains this further. As a general guideline:
- the `/config` path should be accessible only to Vault administrators;
- the `/roles/:roleName` is mainly used to export/import data. It is required to limit the access to this 
  to the concerned administrators.
- the `/roles/:roleName/*` should be granted to the concerned applications based on need-to-use basis.
- the administrator may also wish to limit the grant below `/roles/:roleName` following the least-privilege
  principle (i.e. deny functionality that is not required for this application).

An example of the policy that grants interactive access to TIBCO Cloud Mashery services but disables access to the 
application data:
```hcl
path "mash-creds/roles/:roleName/v3/services" {
  capabilities = [ "read", "list" ]
}

path "mash-creds/roles/:roleName/v3/services/*" {
  capabilities = [ "read", "list" ]
}
```

The following policy is suitable for CI/CD tool that works in the [proxy mode](proxy_mode.html.markdown)
and needs to perform TIBCO Cloud Mashery deployments:
```hcl
path "mash-creds/roles/:roleName/proxy/v3/services" {
  capabilities = [  "create", "read", "update", "delete", "list"  ]
}

path "mash-creds/roles/:roleName/v3/services/*" {
  capabilities = [  "create", "read", "update", "delete", "list"  ]
}
```

## Basic configuration

The default configuration can be read by the following command:
```shell
vault read mash-creds/config
```
which will produce the output:
```shell
Key                        Value
---                        -----
enable_cli_v3_write        false
mashery issuer cert        n/a
mashery leaf cert          n/a
mashery root cert          n/a
net_latency (effective)    147ms
oaep_label (effective)     sha256:d7cd1ff4cd116846fb90cc0843490d5fef80c2f19352849dbb518d36cf080f31
proxy_server               n/a
proxy_server_auth          n/a
proxy_server_creds         n/a
tls_pinning (desired)      default
tls_pinning (effective)    default
```
The expectation of this configuration is that:
- no proxy server is required to connect to TIBCO Cloud Mashery;
- default TLS certificate pinning is enabled
- plugin-default OAEP label will be used to additional assert credentials exchanges
- network latency between the site and Mashery, including possible clock skew, is 147 milliseconds.
- CLI-write for Mashery V3 APIs is disabled.

## Enabling CLI write operations for V3 API

The CLI V3 write is controlled by `enable_cli_v3_write` property of boolean type. As Vault administrator,
you set the desired value by executing:
```shell
vault write mash-creds/config enable_cli_v3_write=<desired value>
```
where `<desired value` can be either `true` or `false`

## Adjusting latency
If your site is deployed on a high-speed Internet connection that has a tiny ping to `mashery.com` website,
it is advised to set `net_latency` key to the average ping value, such as to set latency to 3 milliseconds:
```shell
vault write mash-creds/config net_latency=3ms
```

## Enabling Proxy Server

Where the Vault needs to connect to TIBCO Cloud Mashery via a proxy server, this is set by:
```shell
vault write mash-creds/config proxy_server=<url>
```
Where the proxy server requires the authentication, two more keys need to be added `proxy_server_auth`
and `proxy_server_creds` that, together, construct `Proxy-Authentication`.

# Hardening

## Assigning specific OAEP label

For additional security, it is recommended to change the default OAEP label. An OAEP label is a 
set of random bytes that is meant to introduce additional assertions when encrypting/decrypting data. The
main use of the customized OAEP label is to strictly enforce environmental separation, such as e.g. 
credentials stored in production servers cannot be copied into Vault servers supporting test deployments.

As an example, an OAEP can be obtained with the following command:
```shell
cat /dev/urandom | tr -cd '[a-zA-Z0-9]' | fold -w 30 | head -n 1
```
which will produce a string of 30 random characters, such as `TNDphLGJx7LJwRcXOtM9O6oYBajBiD`
> Do not use this label in your deployments!

The OAEP label is stored in the configuration using 
```shell
vault write mash-creds/config oaep_label="<value>"
``` 

## Mashery certificate pinning

The plugin supports three certificate pinning modes:
- `default`, which requires a certificate with `*.mashery.com` common name issued by `DigiCert TLS RSA SHA256 2020 CA1`
  with serial number `0A:35:08:D5:5C:29:2B:01:7D:F8:AD:65:C0:0F:F7:E4`.
- `system`, which will delegate the trust to the operating system
- `custom` which allows the administrator to set the pinning
> Note: certificate pinning is an essential element to prevent man-in-the-middle attacks. If your site
> will be attacked, the certificate pinning will reject connections. Establish that your site is not
> being attacked before changing the certificate pinning!

The system configuration is set by the following command:
```shell
vault write mash-creds/config tls_pinning=system
```
Custom pinning can only be enabled where either leaf, issuer, or root certificate are specified. The
pinning is configured with the command:
```shell
vault write mash-creds/config/certs/:type cn="common name" sn="serial number" fp="fingerprint"
```
where:
- `:type` specified the certificate type to pin, either `leaf`, or `issuer`, or `root`
- `cn` specifies the common name that should appear on certificate
- `sn` specifies the certificate's serial number, and
- `fp` specifies the certificate's fingerprint

The pinning is cleared by executing
```shell
vault delete mash-creds/config/certs/:type
```
After at least custom certificate pinning configurations is specified, the custom pinning can be 
activated by 
```shell
vault write mash-creds/config tls_pinning=custom
```

To reset TLS pinning to default, execute
```shell
vault write mash-creds/config tls_pinning=default
```
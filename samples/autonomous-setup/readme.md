# Creating autonomous Vault installation

This quick guide is for Mashery administrator and/or operators looking to get started with Mashery V2/V3 Access
Credentials Secrets Engine using a Docker container setup. The main benefit of Docker is that is provides a low-effort,
just-in-time access to the Mashery sensitive keys on the operator's device.

To set up Vault in the docker container, you would need:

- Basic knowledge of running shell scripts.
- HashiCorp vault [installed on your machine](https://developer.hashicorp.com/vault/docs/install) (so that `vault` command is accessible in the terminal)
- a running Docker;
- an TLS certificate for your own Vault setup (highly advised; but can be skipped if you are in hurry)
- a strong pass phrase you can easily remember to secure sensitive vault keys

After following this guide, you will have:
- initialized Vault with Mashery secret engine enabled
- encrypted unseal script ensuring that only the operator can access the Mashery configuration
- obtained Vault-specific certificates to login

## Scripts location on your machine

You can copy the contents of this directory to the convenient location on your machine. It contains scripts and useful
templates that will be used during the setup. You may want to change the owner and change execution permissions on the
shell scripts to match the user that will be running these.
 

## Replacing bundled TLS certificate

The traffic to your Vault needs to be encrypted in transit. The default container is provided with a temporary
self-signed TLS certificate that supports host names `localhost` and `myvault.local`. Your system will not immediately
trust these certificates, and this is **expected**.

You are strongly encouraged to replace the bundled certificates with the ones that are specific to the machine where
you want to install. Method to create TLS certificates vary greatly and depend on many considerations. A good starting
point would be to use the services of major cloud providers, such as Azure KeyVault or AWS Certificate Manager.

To replace the TLS certificates, replace `vault-container.pem` and `vault-container.key` files containing the certificate
and decrypted private key respectively e.g. using the following Dockerfile (see [`step_1`](./step_1) directory:
```dockerfile
FROM lspwd2/hcvault-mashery-api-auth:latest

COPY ./vault-container* /vault/tls
```
This Dockerfile can be built and run e.g. with the following command
```shell
$ docker build . -t my-mash-vault
```

Alternatively, if you are fine with using the bundled certificates, you could configure your system to trust it.
```shell
-----BEGIN CERTIFICATE-----
MIIFXTCCA0WgAwIBAgIQYvjKUQz4QRuiZlS8YYvERTANBgkqhkiG9w0BAQsFADAYMRYwFAYDVQQD
Ew1teXZhdWx0LmxvY2FsMB4XDTIzMDQxMDExMTU0NloXDTI0MDQxMDExMjU0NlowGDEWMBQGA1UE
AxMNbXl2YXVsdC5sb2NhbDCCAiIwDQYJKoZIhvcNAQEBBQADggIPADCCAgoCggIBANTE33D+tZ2Z
cceSshoO1HXK29cfEoTHvm69Mgddbc2LizEi+YNOksgAdEtdqBUx75zYRy4uyFbvS5bRsEt2im28
iKFRuuscoxHwY78ii56jDsHavh1xBXnWdGtFwN4ymGTdREfY/2bePNodqbZkbQzIFhNgA53NOmqa
2lJxQGuu2B1kOLu1JRjLgYmnvQvBrQq2FRbSDIPKr6fn74JlYfdRBHcr/4YZpfDZZbn45eNVcas4
y0Ph8RViUBN69cr8ZnhXZqstRfa2Mu4063+1F4sxXX4SJBthKYIGkBB6DHv43Bgt9MQdQ1XqAchS
fmlBAik8GDeiXVAJVxUVw5/0NzyGBOaKwlhJZFFL1pb5lyIylwFfE0WfvBalIL+K0mI618iuX9sn
bMryN6F+/ztRtH+iKYrTKymDMkXcIudTBL94W8vmhR4EcS15bMu4q8+yIwcg/XoKP+Z9pwS073nl
blS2ph7iKLt5gmMa+0A3eEo3KsujnvDHCD6NurP2aV3vgjnv6/zajiqXVZCWBFmId/cc43HPiOdN
dboYLCECs1WXF+cv3dhHevUz6dN0/RAXaEA5Rg0uLkZ5cPxfqDKAnwK5kVVNmhjtFMM6pBpvm1HS
LrG84/XfvrjKQrnyD9lAOS/i5Xnhh/LdGTCaUucQvZE7XaW92hb8xA2iHxux3EN5AgMBAAGjgaIw
gZ8wDgYDVR0PAQH/BAQDAgWgMAkGA1UdEwQCMAAwHQYDVR0lBBYwFAYIKwYBBQUHAwEGCCsGAQUF
BwMCMCMGA1UdEQQcMBqCDW15dmF1bHQubG9jYWyCCWxvY2FsaG9zdDAfBgNVHSMEGDAWgBQSICDD
3yX1dEiUNhcS26pb3eTmLjAdBgNVHQ4EFgQUEiAgw98l9XRIlDYXEtuqW93k5i4wDQYJKoZIhvcN
AQELBQADggIBAIKqashqcnpjfpqX3+F2TNcjbZhfxPxbXYvYtOQDzHjRLAbMC7efW5WkY8zXw22o
8uzVuKxeM1JFFZH2nFqa/J9FzhAEg8JA7ecxm3nmfjNVEQE+AE65P/YPiPcwsuEs7BuPOrivj5k0
rKt6vQf42QWrn/xod0W3ofFpfCbJnM4V/ax8FY8YeF2PIOzpf610d/5rvKpEXhdQ+gBO4RZo9Yk6
AbKxqUQtw9FRW3YPQGkUwr6mEtuHNDoJSQJlPNwQxTV/ex2wXirjfDDm/+XTBEyzocJrZmOchDrm
5G/4n1Y4Zf3sOMRB8GYOa/N82OwpIOIqMUa4OH3iWssmXw5tOBvdaMj8Ib2VQZTvwhpJBMwJtjVa
mt2Q3yC95cSks4wZdz43llCxVtoz/e7LGJ0hwe4B9hhZD2Eo75rKJos4QnCIbs2cedzMelto/EDa
Wr8W0m9GXR3keMj8URHdFDUW3MuBi6t5wPHAIB68C2O04iZXlayyjzBs0HJaTbb77AvkMowp4ymV
1MiUQByTaRDoxcfaK20Pg9VFPfkiCyW8aVk5gMzqAZWl1ZvUO8HNtpEE0Qic4K/62uOx8WN69nFK
lZIxWbWGIJ/F8S8XyV1ZprSSz+jk/nYliLsA8Pf2JqGAfbQCORn81B/z0wVSLX6N6fkg0QQA04fY
qDlC+KxJP/QN
-----END CERTIFICATE-----
```
## Prepare the terminal session

The setup and operations in this guide are performed using CLI scripts. It is important to make sure that your terminal
session is prepared.

### Setting `VAULT_ADDR` variable
Ensure that the `VAULT_ADDR` is pointing to the correct host name. A sensible default is `https://localhost:8200/`, which
can be set as
```shell
$ export VAULT_ADDR=https://localhost:8200/
```
### Specifying seal file pass phrase
> **Security consideration**
>
> The reason for this step is to ensure that only owner of the device is able to start the operation of the Vault. To
> ensure that this condition is met, do not store this pass phrase in the clear on your device. Also bear in mind
> that there is no way to recover this pass phrase if you forget it! Write the pass phrase e.g. on a piece of paper and
> put it in a secure location.

The seal file pass phrase protects keys that can be used to get root access to Vault. Each time you start a terminal 
session that performs Vault sealing or unsealing, you will need to add `HCV_SEALFILE_PASS` environment variable by running:
```shell
export HCV_SEALFILE_PASS="<pass phrase you can easily remember>"
```

## Starting and testing connection
The 

Finally, check that `vault` command is able to talk to your vault
```shell
$ vault status
```
which should output the following:
```text
Key                Value
---                -----
Seal Type          shamir
Initialized        false
Sealed             true
Total Shares       0
Threshold          0
Unseal Progress    0/0
Unseal Nonce       n/a
Version            1.13.1
Build Date         2023-03-23T12:51:35Z
Storage Type       file
HA Enabled         false
```
This indicates that you have a Vault that hasn't been initialized.



## Initializing Vault
Next step is to initialize [Vault seal](https://developer.hashicorp.com/vault/docs/concepts/seal#seal-unseal). This
mechanism is suitable for production-grade installation. Since the objective is to provide a single-user vault, 
this guide suggests a simplified alternative: store unsealed keys in an encrypted file on the operator's device.



# Quick Instructions

You can override bundled TLS certificates that will be used by the standalone Vault container. To achieve it:
1. Copy `vault-container.pem` and `vault-container.key` into this directory
2. Run `docker build . -t my-mash-vault`
> Note: `my-mash-vault` is an image used in this guide. You can invent your own image as you see fit.

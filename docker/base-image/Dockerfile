FROM hashicorp/vault:latest

RUN mkdir -p /vault/plugins
COPY ./hcvault-mashery-api-auth_v* /vault/plugins
RUN mv /vault/plugins/hcvault-mashery-api-auth_v*.sha256 ~ && chown -R vault:vault /vault/plugins && chmod a+x /vault/plugins/* && setcap cap_ipc_lock=+ep /vault/plugins/*
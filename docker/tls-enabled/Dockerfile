FROM --platform=$TARGETPLATFORM hashicorp/vault:latest
ARG TARGETPLATFORM

COPY config.hcl /vault/config
RUN mkdir -p /vault/plugins /vault/tls

COPY ./dist/$TARGETPLATFORM/hcvault-mashery-api-auth_v* /vault/plugins
RUN mv /vault/plugins/hcvault-mashery-api-auth_v*.sha256 /home/vault/ && chown -R vault:vault /vault/plugins && chmod a+x /vault/plugins/* && setcap cap_ipc_lock=+ep /vault/plugins/*

COPY ./vault-container* /vault/tls

CMD [ "vault", "server", "-config=/vault/config" ]

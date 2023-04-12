FROM vault:latest

COPY config.hcl /vault/config
RUN mkdir -p /vault/plugins /vault/tls

COPY ./hcvault-mashery-api-auth_v* /vault/plugins
RUN chown -R vault:vault /vault/plugins && chmod a+x /vault/plugins/* && setcap cap_ipc_lock=+ep /vault/plugins/*

COPY ./vault-container* /vault/tls

CMD [ "vault", "server", "-config=/vault/config" ]

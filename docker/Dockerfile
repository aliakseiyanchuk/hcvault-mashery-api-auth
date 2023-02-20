FROM vault:latest

CMD [ "vault", "server", "-config=/vault/config" ]

RUN mkdir -p /vault/plugins
# The executable should be available in the root directory.
COPY ./mashery-api-creds* /vault/plugins
RUN chmod a+x /vault/plugins/* && setcap cap_ipc_lock=+ep /vault/plugins/*

# You may want to budnle additional resources, such as SSL certificates
COPY ./config.hcl /vault/config

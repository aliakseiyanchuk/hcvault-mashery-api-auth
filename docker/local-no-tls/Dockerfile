FROM vault:latest

# You may want to budnle additional resources, such as SSL certificates
COPY config.hcl /vault/config
RUN mkdir -p /vault/plugins
# The executable should be available in the root directory.
COPY ./hcvault-mashery-api-auth_v* /vault/plugins
RUN /vault/plugins/hcvault-mashery-api-auth_v*.sha256 ~ && chmod a+x /vault/plugins/* && setcap cap_ipc_lock=+ep /vault/plugins/*

CMD [ "vault", "server", "-config=/vault/config" ]

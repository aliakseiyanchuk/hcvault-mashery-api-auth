FROM ghcr.io/aliakseiyanchuk/hcvault-mashery-api-auth:sha256-55e787a63a32ca62bcca7f3e646a90643e10f055449700e3df7293f9726a5ba4 AS hcvault-distro

FROM hashicorp/vault:latest
ARG PLATFORM

# You may want to budnle additional resources, such as SSL certificates
COPY config.hcl /vault/config
RUN mkdir -p /vault/plugins

# The executable should be available in the root directory.
COPY --from=hcvault-distro /home/distro/$PLATFORM/hcvault-mashery-api-auth* /vault/plugins
RUN chown vault:vault /vault/plugins/* && \
      chmod a+r /vault/plugins/hcvault-mashery-api-auth.sha256 && \
      chmod a+rx /vault/plugins/hcvault-mashery-api-auth && \
      setcap cap_ipc_lock=+ep /vault/plugins/hcvault-mashery-api-auth

CMD [ "vault", "server", "-config=/vault/config" ]

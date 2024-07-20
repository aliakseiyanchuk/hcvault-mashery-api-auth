FROM ghcr.io/aliakseiyanchuk/hcvault-mashery-api-auth:latest AS hcvault-distro

FROM docker.io/alpine/openssl:latest as cert-builder

WORKDIR /home/cert

# The subject in the expample should be modified to suit your particular needs
RUN openssl req -x509 -newkey rsa:4096 -keyout vault-container.key -out vault-container.pem -sha256 -days 365 -nodes -subj "/C=NL/CN=vault"


FROM hashicorp/vault:latest
ARG PLATFORM

COPY config.hcl /vault/config
RUN mkdir -p /vault/plugins /vault/tls

COPY --from=hcvault-distro /home/distro/$PLATFORM/hcvault-mashery-api-auth /vault/plugins
COPY --from=cert-builder /home/cert/vault-container* /vault/tls/

RUN chown vault:vault /vault/plugins/* && \
      chmod a+rx /vault/plugins/hcvault-mashery-api-auth && \
      setcap cap_ipc_lock=+ep /vault/plugins/hcvault-mashery-api-auth

CMD [ "vault", "server", "-config=/vault/config" ]

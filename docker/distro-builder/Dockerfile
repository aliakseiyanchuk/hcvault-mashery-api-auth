FROM alpine:latest

LABEL org.opencontainers.image.title="HashiCorp Vault secret engine for Mashery Distribution Container"
LABEL org.opencontainers.image.description="Contains pre-built HashiCorp secrets engine for use with deriving custom containers"
LABEL org.opencontainers.image.authors="Alaiksei Yanchuk"

USER distro
WORKDIR /home/distro

COPY ./dist /home/distro/

ENTRYPOINT ["/bin/sh"]
CMD [ "-c", "echo 'Use this container to copy files' && exit 1" ]
FROM alpine:latest

LABEL org.opencontainers.image.title="HashiCorp Vault secret engine for Mashery Distribution Container"
LABEL org.opencontainers.image.description="Contains pre-built HashiCorp secrets engine for use with deriving custom containers"
LABEL org.opencontainers.image.authors="Alaiksei Yanchuk"

WORKDIR /home/distro

COPY ./sig.sh ./dist/ /home/distro/
RUN chmod u+x sig.sh


RUN adduser -D distro
USER distro

ENTRYPOINT ["/bin/sh", "/home/distro/sig.sh"]

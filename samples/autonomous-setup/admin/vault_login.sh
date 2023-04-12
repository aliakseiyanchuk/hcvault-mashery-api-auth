#!/bin/sh

DIR_PREFIX=$(dirname "$0")
SCRIPT_DIR=$(realpath "$DIR_PREFIX")
ROOT=$(dirname "$SCRIPT_DIR")
SECRETS_DIR=${ROOT}/.secret
CERT_PEM=$SECRETS_DIR/cert.pem
CERT_KEY=$SECRETS_DIR/cert.key

vault login -method=cert -client-cert=${CERT_PEM} -client-key=${CERT_KEY} name=mashery-admin
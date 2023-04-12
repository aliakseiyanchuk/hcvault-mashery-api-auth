#!/bin/sh

DIR_PREFIX=$(dirname "$0")
source $DIR_PREFIX/functions.sh || exit 1

initLocations "$DIR_PREFIX"

vault login -method=cert -client-cert=${CERT_PEM} -client-key=${CERT_KEY} name=mashery-admin
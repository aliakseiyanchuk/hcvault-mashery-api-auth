#!/bin/sh

DIR_PREFIX=$(dirname "$0")
. $DIR_PREFIX/functions.sh || exit 1

initLocations "$DIR_PREFIX"

checkUserCertExpiry

if [ "$USER_CERT_EXPIRED" != "" ] ; then
  echo "Admin certificate has expired. You should rotate the certificate to continue,"
  echo "To rotate, you need to know the unseal password of this Vault"
  exit 1
fi

vault login -method=cert -client-cert=${CERT_PEM} -client-key=${CERT_KEY} name=mashery-admin
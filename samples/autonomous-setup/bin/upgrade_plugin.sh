#!/bin/sh

DIR_PREFIX=$(dirname "$0")
. $DIR_PREFIX/functions.sh || exit 1

initLocations "$DIR_PREFIX"
loadIdentity

deriveBinaryVersionFromContainer

if [ "" = "$MASH_AUTH_BINARY" ] || [ "" = "$MASH_AUTH_BINARY_SHA" ] ; then
  echo "Cannot establish plugin binary name and sha256 signature. Is the container running?"
  exit 1
fi

verifyEncryptionPassword
generateRootAndLogin
trap 'vault token revoke -self > /dev/null' EXIT

echo "Reloading Mashery secrets engine..."
vault plugin register \
  -sha256=${MASH_AUTH_BINARY_SHA} \
  secret ${MASH_AUTH_BINARY}

vault plugin reload -mounts mash-auth/

#!/bin/sh

DIR_PREFIX=$(dirname "$0")
. $DIR_PREFIX/functions.sh || exit 1

initLocations "$DIR_PREFIX"

verifyEncryptionPassword

generateRootToken
echo $ROOT_TOKEN
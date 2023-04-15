#!/bin/sh

DIR_PREFIX=$(dirname "$0")
. $DIR_PREFIX/functions.sh || exit 1

initLocations "$DIR_PREFIX"
loadIdentity

verifyEncryptionPassword

generateRootAndLogin
trap 'vault token revoke -self > /dev/null' EXIT

issueUserCertificate

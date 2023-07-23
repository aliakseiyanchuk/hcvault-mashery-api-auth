#!/bin/sh

DIR_PREFIX=$(dirname "$0")
. $DIR_PREFIX/functions.sh || exit 1

initLocations "$DIR_PREFIX"
loadIdentity

verifyEncryptionPassword
generateRootAndLogin
trap 'vault token revoke -self > /dev/null' EXIT


ROLE_ID=$(vault read  auth/approle/role/mashery-vault-agent/role-id -format=json | jq -r .data.role_id)
SECRET_ID=$(vault write auth/approle/role/mashery-vault-agent/secret-id secret_id_num_uses=1 -format=json | jq -r .data.secret_id)

vault write auth/approle/login \
    role_id=$ROLE_ID \
    secret_id=$SECRET_ID \
    -format=json | jq -r .auth.client_token | vault login -

#!/bin/sh

DIR_PREFIX=$(dirname "$0")
. $DIR_PREFIX/functions.sh || exit 1

initLocations "$DIR_PREFIX"
loadIdentity

verifyEncryptionPassword
generateRootAndLogin
trap 'vault token revoke -self > /dev/null' EXIT

# Launching the agent
set -x
vault read  auth/approle/role/mashery-vault-agent/role-id -format=json | jq -r .data.role_id > "$ROLE_ID"
vault write auth/approle/role/mashery-vault-agent/secret-id secret_id_num_uses=1 -format=json | jq -r .data.secret_id > "$ROLE_SECRET_ID"

cd $AGENT_DIR || exit 1
rm nohup.out

cat "$ROLE_SECRET_ID"

nohup vault agent -config=./agent.hcl &
echo "Agent started"
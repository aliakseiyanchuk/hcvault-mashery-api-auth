#!/bin/sh

DIR_PREFIX=$(dirname "$0")
. $DIR_PREFIX/functions.sh || exit 1

verifyPreconditions

initLocations "$DIR_PREFIX"
loadIdentity

deriveBinaryVersionFromContainer

if [ "" = "$MASH_AUTH_BINARY" ] || [ "" = "$MASH_AUTH_BINARY_SHA" ] ; then
  echo "Cannot establish plugin binary name and sha256 signature. Is the container running?"
  exit 1
fi

# Make sure the script will stop when the first error will occur
set -e

echo "Checking Vault status..."
VAULT_INIT=$(vault status -format=json | jq -r .initialized )
if [ "$VAULT_INIT" != "false" ] ; then
  echo "This Vault has already been initialized. Unseal it instead (if it is sealed)"
  exit 1
fi

echo "Verifying unseal keys encryption password..."
verifyEncryptionPassword

echo "Initializing and unsealing vault..."
vault operator init -format=json | openssl enc -a -aes-128-cbc -pass env:HCV_SEALFILE_PASS -out "$UNSEAL_FILE"
passUnsealToken 0
passUnsealToken 1
passUnsealToken 2


echo "Logging in as root user..."
rootLogin
trap 'vault token revoke -self > /dev/null' EXIT

echo "Enabling Mashery secrets engine"
vault plugin register \
  -sha256=${MASH_AUTH_BINARY_SHA} \
  secret ${MASH_AUTH_BINARY}
vault secrets enable -path=mash-auth \
              -allowed-response-headers="X-Total-Count" \
              -allowed-response-headers="X-Mashery-Responder" \
              -allowed-response-headers="X-Server-Date" \
              -allowed-response-headers="X-Proxy-Mode" \
              -allowed-response-headers="WWW-Authenticate" \
              -allowed-response-headers="X-Mashery-Error-Code" \
              -allowed-response-headers="X-Mashery-Responder" \
              ${MASH_AUTH_BINARY}

echo "Setting up user certificate login"
vault secrets enable pki
vault secrets tune -max-lease-ttl=87600h pki
vault write pki/root/generate/internal common_name="$LOCAL_CA_CN" ttl=87600h

# Default certificate TTL is 1 week, after which it need to be rotated.
# Only accept email from the specified domain
# common name must be an email
# Allow email addresses directly at the specified email domain
# Sub-domains are not required for the single operator
 # Wildcards have no practical applications
vault write pki/roles/mashery-admin \
  ttl=178h max_ttl=336h \
  allowed_domains=$OPERATOR_EMAIL_DOMAIN \
  cn_validations=email \
  allow_bare_domains=true \
  allow_subdomains=false \
  allow_wildcard_certificates=false ;

issueUserCertificate

echo "Enabling operator TLS-based authentication"
vault auth enable cert
vault write auth/cert/certs/mashery-admin certificate=@${CA_PEM} \
  token_ttl=8h token_max_ttl=24h

echo Creating operator entity and policy
< "$POLICIES_DIR/operator_policy.hcl" vault policy write mashery-admin-policy -

ENTITY_JSON=/tmp/.entity.json
trap 'rm -rf ${ENTITY_JSON}' EXIT
vault write /identity/entity -format=json name="$OPERATOR_ENTITY_NAME" policies=default policies=mashery-admin-policy > "${ENTITY_JSON}"
cat $ENTITY_JSON
ENTITY_ID=$(jq -r .data.id "${ENTITY_JSON}")
MOUNT_ACCESSOR_ID=$(vault auth list -format=json | jq -r '."cert/".accessor')
vault write /identity/entity-alias name="$OPERATOR_EMAIL" canonical_id="$ENTITY_ID" mount_accessor="$MOUNT_ACCESSOR_ID"

echo Enabling Agent login...
vault auth enable approle
vault write auth/approle/role/mashery-vault-agent \
  token_max_ttl=8h token_policies=mashery-admin-policy
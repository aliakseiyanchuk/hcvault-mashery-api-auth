#!/bin/sh

DIR_PREFIX=$(dirname "$0")
SCRIPT_DIR=$(realpath "$DIR_PREFIX")
ROOT=$(dirname "$SCRIPT_DIR")
SECRETS_DIR=${ROOT}/.secret
UNSEAL_FILE=$SECRETS_DIR/unseal.json.enc
CERTS_FILE=$SECRETS_DIR/certs.json
CA_PEM=$SECRETS_DIR/ca.pem
CERT_PEM=$SECRETS_DIR/cert.pem
CERT_KEY=$SECRETS_DIR/cert.key
ROLE_ID=$SECRETS_DIR/agent_role_id

MASH_AUTH_BINARY=hcvault-mashery-api-auth_v0.3
MASH_AUTH_BINARY_SHA=8557d544cc8134680ff3a492002ab432bc20c2fe889085acee1c341121ea4236

# You may want to change this to suit your needs.
OPERATOR_EMAIL_DOMAIN=operations.local
OPERATOR_EMAIL=local.admin@$OPERATOR_EMAIL_DOMAIN

passUnsealToken() {
  UNSEAL_TOKEN=$(openssl enc -a -d -aes-128-cbc -pass env:HCV_SEALFILE_PASS -in "$UNSEAL_FILE" |\
   jq -r .unseal_keys_b64\[$1\])

  vault operator unseal "$UNSEAL_TOKEN"
}

rootLogin() {
  ROOT_TOKEN=$(openssl enc -a -d -aes-128-cbc -pass env:HCV_SEALFILE_PASS -in "$UNSEAL_FILE" |\
     jq -r .root_token)

    vault login token="$ROOT_TOKEN" > /dev/null
}

# Make sure the script will stop when the first error will occur
set -ex

echo "Initializing and unsealing vault..."
vault operator init -format=json | openssl enc -a -aes-128-cbc -pass env:HCV_SEALFILE_PASS -out "$UNSEAL_FILE"
passUnsealToken 0
passUnsealToken 1
passUnsealToken 2


echo "Logging in as root user..."
rootLogin

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
vault write pki/root/generate/internal common_name="Mashery Vault Users" ttl=876h
vault write pki/roles/mashery-admin \
  allowed_domains=$OPERATOR_EMAIL_DOMAIN \
  cn_validations=email \
  allow_bare_domains=true allow_subdomains=false allow_wildcard_certificates=false

vault write pki/issue/mashery-admin common_name=$OPERATOR_EMAIL -format=json > "$CERTS_FILE"
jq -r .data.private_key   "$CERTS_FILE" > "$CERT_KEY"
jq -r .data.certificate   "$CERTS_FILE" > "$CERT_PEM"
jq -r .data.issuing_ca    "$CERTS_FILE" > "$CA_PEM"

echo "Enabling operator TLS-based authentication"
vault auth enable cert
vault write auth/cert/certs/mashery-admin certificate=@${CA_PEM} \
  token_ttl=8h token_max_ttl=24h

echo Creating operator entity and policy
< "$DIR_PREFIX/operator_policy.hcl" vault policy write mashery-admin-policy -

ENTITY_JSON=/tmp/.entity.json
trap 'rm -rf ${ENTITY_JSON}' EXIT
vault write /identity/entity name="Mashery Local Operator" policies=default policies=mashery-admin-policy -format=json> "${ENTITY_JSON}"
cat $ENTITY_JSON
ENTITY_ID=$(jq -r .data.id "${ENTITY_JSON}")
MOUNT_ACCESSOR_ID=$(vault auth list -format=json | jq -r '."cert/".accessor')
vault write /identity/entity-alias name=$OPERATOR_EMAIL canonical_id="$ENTITY_ID" mount_accessor="$MOUNT_ACCESSOR_ID"

echo Enabling Agent login...
vault auth enable approle
vault write auth/approle/role/mashery-vault-agent \
  token_max_ttl=8h token_policies=mashery-admin-policy
vault read auth/approle/role/mashery-vault-agent/role-id -format=json | jq -r .data.role_id > "$ROLE_ID"
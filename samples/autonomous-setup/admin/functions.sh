#!/bin/sh

initLocations() {
  SCRIPT_DIR=$(realpath "$1")
  ROOT=$(dirname "$SCRIPT_DIR")
  ROOT=$(realpath --relative-to $(pwd) $ROOT)

  SECRETS_DIR=${ROOT}/.secret
  ENC_CHECK=${SECRETS_DIR}/encryption_check.enc

  UNSEAL_FILE=$SECRETS_DIR/unseal.json.enc
  CERTS_FILE=$SECRETS_DIR/certs.json
  CA_PEM=$SECRETS_DIR/ca.pem
  CERT_PEM=$SECRETS_DIR/cert.pem
  CERT_KEY=$SECRETS_DIR/cert.key
  ROLE_ID=$SECRETS_DIR/agent_role_id
}

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

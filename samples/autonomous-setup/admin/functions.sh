#!/bin/sh

initLocations() {
  SCRIPT_DIR=$(realpath "$1")
  ROOT=$(dirname "$SCRIPT_DIR")
  ROOT=$(realpath --relative-to "$(pwd)" $ROOT)

  SECRETS_DIR=${ROOT}/.secret
  ENC_CHECK=${SECRETS_DIR}/encryption_check.enc

  UNSEAL_FILE=$SECRETS_DIR/unseal.json.enc
  CERTS_FILE=$SECRETS_DIR/certs.json
  CA_PEM=$SECRETS_DIR/ca.pem
  CERT_PEM=$SECRETS_DIR/cert.pem
  CERT_KEY=$SECRETS_DIR/cert.key
  ROLE_ID=$SECRETS_DIR/agent_role_id
}

loadIdentity() {
  . $SCRIPT_DIR/default_identity.sh

  OVERRIDE=$ROOT/me.sh
  if [ -f "$OVERRIDE" ] ; then
    . "$OVERRIDE"
  fi
}

verifyPreconditions() {
  if [ "$HCV_SEALFILE_PASS" = "" ] ; then
    echo "Please set $HCV_SEALFILE_PASS environment variable to contain unseal key"
    echo "This can be done by running the following command:"
    echo "read HCV_SEALFILE_PASS; export HCV_SEALFILE_PASS"
    exit 1
  fi

  if ! which vault ; then
    echo "vault command is not found on your path";
    exit 1
  fi

  if ! which jq ; then
    echo "jq command is not found on your path";
    exit 1
  fi
}

passUnsealToken() {
  UNSEAL_TOKEN=$(openssl enc -a -d -aes-128-cbc -pass env:HCV_SEALFILE_PASS -in "$UNSEAL_FILE" |
    jq -r .unseal_keys_b64\[$1\])

  vault operator unseal "$UNSEAL_TOKEN"
}

passUnsealTokenToRootGeneration() {
  UNSEAL_TOKEN=$(openssl enc -a -d -aes-128-cbc -pass env:HCV_SEALFILE_PASS -in "$UNSEAL_FILE" |
    jq -r .unseal_keys_b64\[$1\])

  ENC_ROOT_TOKEN=$(vault operator generate-root -otp="$2" -nonce="$3" -format=json "$UNSEAL_TOKEN" | jq -r .encoded_root_token)
}

rootLogin() {
  ROOT_TOKEN=$(openssl enc -a -d -aes-128-cbc -pass env:HCV_SEALFILE_PASS -in "$UNSEAL_FILE" |
    jq -r .root_token)

  vault login token="$ROOT_TOKEN" >/dev/null
}

generateRootAndLogin() {
  OTP=$(vault operator generate-root -generate-otp -format=json | jq -r .otp)
  NONCE=$(vault operator generate-root -init -otp="$OTP" -format=json | jq -r .nonce)
  passUnsealTokenToRootGeneration 0 $OTP $NONCE
  passUnsealTokenToRootGeneration 1 $OTP $NONCE
  passUnsealTokenToRootGeneration 2 $OTP $NONCE

  ROOT_TOKEN=$(vault operator generate-root -nonce="$NONCE" -otp="$OTP" -decode="$ENC_ROOT_TOKEN" -format=json | jq -r .token)
  vault login token="$ROOT_TOKEN" >/dev/null
}


issueUserCertificate() {
  vault write pki/issue/mashery-admin common_name=$OPERATOR_EMAIL -format=json > "$CERTS_FILE"
  jq -r .data.private_key   "$CERTS_FILE" > "$CERT_KEY"
  jq -r .data.certificate   "$CERTS_FILE" > "$CERT_PEM"
  jq -r .data.issuing_ca    "$CERTS_FILE" > "$CA_PEM"
}

checkUserCertExpiry() {
  if [ -f $CERT_PEM ]; then
    CERT_EXPIRY=$(openssl x509 -in "$CERT_PEM" -enddate -noout | awk -F= '{print $2}')
    CERT_DT=$(date -d "$CERT_EXPIRY" "+%s")
    NOW_DT=$(date "+%s")

    if [ $CERT_DT -le $NOW_DT ]; then
      USER_CERT_EXPIRED=true
    fi
  fi
}

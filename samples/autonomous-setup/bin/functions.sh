#!/bin/sh

initLocations() {
  SCRIPT_DIR=$(realpath "$1")
  ROOT=$(dirname "$SCRIPT_DIR")

  # For
  case "$(uname -s)" in
  CYGWIN*)
    ROOT=$(realpath --relative-to "$(pwd)" $ROOT)
    ;;
  esac

  SECRETS_DIR=${ROOT}/.secret
  POLICIES_DIR=${ROOT}/policies
  AGENT_DIR=${ROOT}/agent
  ENC_CHECK=${SECRETS_DIR}/encryption_check.enc

  UNSEAL_FILE=$SECRETS_DIR/unseal.json.enc
  CERTS_FILE=$SECRETS_DIR/certs.json
  CA_PEM=$SECRETS_DIR/ca.pem
  CERT_PEM=$SECRETS_DIR/cert.pem
  CERT_KEY=$SECRETS_DIR/cert.key


  ROLE_ID=$AGENT_DIR/agent_role_id
  ROLE_SECRET_ID=$AGENT_DIR/agent_secret_id
  AGENT_CFG=$AGENT_DIR/agent.hcl
  AGENT_PID=$AGENT_DIR/agent.pid
}

loadIdentity() {
  . $SCRIPT_DIR/default_identity.sh

  OVERRIDE=$ROOT/me.sh
  if [ -f "$OVERRIDE" ]; then
    . "$OVERRIDE"
  fi
}

verifyPreconditions() {
  #  if [ "$HCV_SEALFILE_PASS" = "" ] ; then
  #    echo "Please set $HCV_SEALFILE_PASS environment variable to contain unseal key"
  #    echo "This can be done by running the following command:"
  #    echo "read -s HCV_SEALFILE_PASS; export HCV_SEALFILE_PASS"
  #    exit 1
  #  fi

  if ! which vault >/dev/null; then
    echo "vault command is not found on your path"
    exit 1
  fi

  if ! which jq >/dev/null; then
    echo "jq command is not found on your path"
    exit 1
  fi
}

verifyEncryptionPassword() {
  if [ "$HCV_SEALFILE_PASS" = "" ]; then
    echo "Please enter the password protecting the unseal keys of your vault:"
    read -s HCV_SEALFILE_PASS
    export HCV_SEALFILE_PASS
  fi

  if [ -f "$ENC_CHECK" ]; then
    openssl enc -a -d -aes-128-cbc -pass env:HCV_SEALFILE_PASS -in "$ENC_CHECK" >/dev/null 2>&1
    if [ $? -ne 0 ]; then
      echo "The encryption password you have entered is not correct."
      unset HCV_SEALFILE_PASS
      exit 1
    fi
  else
    echo "password check" | openssl enc -a -e -aes-128-cbc -pass env:HCV_SEALFILE_PASS -out "$ENC_CHECK"
  fi
}

passUnsealToken() {
  UNSEAL_TOKEN=$(openssl enc -a -d -aes-128-cbc -pass env:HCV_SEALFILE_PASS -in "$UNSEAL_FILE" |
    jq -r .unseal_keys_b64\[$1\] | td -r '\r')

  vault operator unseal "$UNSEAL_TOKEN"
}

passUnsealTokenToRootGeneration() {
  UNSEAL_TOKEN=$(openssl enc -a -d -aes-128-cbc -pass env:HCV_SEALFILE_PASS -in "$UNSEAL_FILE" |
    jq -r .unseal_keys_b64\[$1\] | td -r '\r')

  ENC_ROOT_TOKEN=$(vault operator generate-root -otp="$2" -nonce="$3" -format=json "$UNSEAL_TOKEN" | jq -r .encoded_root_token | td -r '\r')
}

rootLogin() {
  ROOT_TOKEN=$(openssl enc -a -d -aes-128-cbc -pass env:HCV_SEALFILE_PASS -in "$UNSEAL_FILE" |
    jq -r .root_token | td -r '\r')

  vault login token="$ROOT_TOKEN" >/dev/null
}

generateRootToken() {
  OTP=$(vault operator generate-root -generate-otp -format=json | jq -r .otp | td -r '\r')
  NONCE=$(vault operator generate-root -init -otp="$OTP" -format=json | jq -r .nonce | td -r '\r')

  if [ "$NONCE" = "" ]; then
    echo "No nonce returned to start the root generation."
    echo "Try running this command: vault operator generate-root -cancel"
    exit 1
  fi

  passUnsealTokenToRootGeneration 0 $OTP $NONCE
  passUnsealTokenToRootGeneration 1 $OTP $NONCE
  passUnsealTokenToRootGeneration 2 $OTP $NONCE

  ROOT_TOKEN=$(vault operator generate-root -nonce="$NONCE" -otp="$OTP" -decode="$ENC_ROOT_TOKEN" -format=json | jq -r .token | td -r '\r')
}

generateRootAndLogin() {
  generateRootToken
  vault login token="$ROOT_TOKEN" >/dev/null
}

issueUserCertificate() {
  vault write pki/issue/mashery-admin common_name=$OPERATOR_EMAIL -format=json >"$CERTS_FILE"
  jq -r .data.private_key "$CERTS_FILE" >"$CERT_KEY"
  jq -r .data.certificate "$CERTS_FILE" >"$CERT_PEM"
  jq -r .data.issuing_ca "$CERTS_FILE" >"$CA_PEM"
}

checkUserCertExpiry() {
  if [ -f $CERT_PEM ]; then
    CERT_EXPIRY=$(openssl x509 -in "$CERT_PEM" -enddate -noout | awk -F= '{print $2}')

    case "$(uname -s)" in
    Darwin*)
      CERT_EXPIRY=$(printf "$CERT_EXPIRY" | sed s/\ GMT/\ +0000/)
      CERT_DT=$(date -j -f "%b %d %H:%M:%S %Y %z" "$CERT_EXPIRY" "+%s")
      ;;
    *)
      CERT_DT=$(date -d "$CERT_EXPIRY" "+%s")
      ;;
    esac

    NOW_DT=$(date "+%s")

    if [ $CERT_DT -le $NOW_DT ]; then
      USER_CERT_EXPIRED=true
    fi
  fi
}

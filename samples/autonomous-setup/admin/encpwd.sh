#!/bin/sh

DIR_PREFIX=$(dirname "$0")
. $DIR_PREFIX/functions.sh || exit 1

initLocations "$DIR_PREFIX"

if [ "$HCV_SEALFILE_PASS" = "" ] ; then
  echo "Please specify password protecting unseal keys by running: read HCV_SEALFILE_PASS"
  exit 1
fi

if [ -f "$ENC_CHECK" ]; then
  openssl enc -a -d -aes-128-cbc -pass env:HCV_SEALFILE_PASS -in "$ENC_CHECK" > /dev/null 2>&1
  if [ $? -ne 0 ] ; then
    echo "Incorrect password"
    unset HCV_SEALFILE_PASS
  fi
else
  echo "password check" | openssl enc -a -e -aes-128-cbc -pass env:HCV_SEALFILE_PASS -out "$ENC_CHECK"
fi
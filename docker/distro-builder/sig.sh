#!/bin/sh

if [ "$1" = "" ] ; then
  echo "Please supply an architecture"
  exit 1
fi
FILE=$(find /home/distro/$1 -name 'hcvault-mashery-api-auth_v*.sha256' )

if [ "$FILE" = "" ] ; then
  echo "No signature found"
  exit 1
else
  CMD=$(basename $FILE .sha256)
  SHA=$(cat $FILE | awk -F'= ' '{print $2}')
  VERSION=$(echo $FILE | sed 's/^.*_v\([0-9\.]*\).sha256$/\1/')

  echo $CMD
  echo $VERSION
  echo $SHA
  echo vault plugin register -sha256=${SHA} -command=${CMD} -version=${VERSION} secret hcvault-mashery-api-auth
fi
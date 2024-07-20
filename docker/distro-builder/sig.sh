#!/bin/sh

if [ "$1" = "" ] ; then
  echo "Please supply an architecture"
  exit 1
fi

FILE=$(find /home/distro/$1 -name 'hcvault-mashery-api-auth_v*.sha256' )

if [ "$FILE" = "" ] ; then
  echo "No signature for architecture $1"
  exit 1
else
  basename $FILE .sha256
  cat $FILE | awk -F'= ' '{print $2}'
fi
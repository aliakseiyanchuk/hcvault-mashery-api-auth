#!/bin/sh

if [ "$1" = "" ] ; then
  echo "Please supply an architecture"
  exit 1
elif [ ! -f /home/distro/$1/hcvault-mashery-api-auth.sha256 ] ; then
  echo "No signature for architecture $1"
  exit 1
else
  cat /home/distro/$1/hcvault-mashery-api-auth.sha256 | awk -F'= ' '{print $2}'
fi
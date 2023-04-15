#!/bin/sh

MASH_AUTH_BINARY=hcvault-mashery-api-auth_v0.3
MASH_AUTH_BINARY_SHA=2992674a8c3ed61a7bb974ea89ebf7ddba2cdb2e4bcf3f93bc70f28e4dc1c275

if [ "$(uname)" = "Darwin" ] && [ "$(uname -p)" = "arm" ] ; then
 MASH_AUTH_BINARY_SHA=8557d544cc8134680ff3a492002ab432bc20c2fe889085acee1c341121ea4236
fi
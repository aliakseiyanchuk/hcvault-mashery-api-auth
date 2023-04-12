#!/bin/bash

DIR_PREFIX=$(dirname "$0")
source $DIR_PREFIX/functions.sh || exit 1

initLocations "$DIR_PREFIX"

passUnsealToken 0
passUnsealToken 1
passUnsealToken 2

#!/bin/sh

DIR_PREFIX=$(dirname "$0")
. $DIR_PREFIX/functions.sh || exit 1

initLocations "$DIR_PREFIX"

if [ -f "$AGENT_PID" ] ; then
  PID=$(cat "$AGENT_PID")
  kill -3 $PID
fi

sleep 2

NUM_REMAINING=$(ps aux | grep vault | grep agent | wc -l)
if [ $NUM_REMAINING -gt 0 ] ; then
   echo: "Warning: Vault agent didn't exit gracefully. You need to shut it down manually"
fi
#!/bin/sh

DIR_PREFIX=$(dirname "$0")
. $DIR_PREFIX/functions.sh || exit 1

initLocations "$DIR_PREFIX"

sighupAgent() {
  if [ -f "$AGENT_PID" ]; then
    PID=$(cat "$AGENT_PID")

    PROC_CNT=$(ps aux | grep $PID | wc -l)

    if [ $PROC_CNT != 0 ] ; then
      kill -3 $PID
      sleep 2
    else
      echo "Agent isn't currently running"
    fi
  fi
}

sighupAgentCygwin() {
  if [ -f "$AGENT_PID" ]; then
    PID=$(cat "$AGENT_PID")
    CYGWIN_PID=$(ps aux | grep $PID | awk '{print $1}')

    if [ "$CYGWIN_PID" != "" ]; then
      kill -3 $CYGWIN_PID
      sleep 2
    else
      echo "Agent isn't currently running"
    fi
  fi
}

case "$(uname)" in
CYGWIN*)
  sighupAgentCygwin
  ;;

*)
  sighupAgent
  ;;
esac



NUM_REMAINING=$(ps aux | grep vault | grep agent | wc -l)
if [ $NUM_REMAINING -gt 0 ]; then
  echo "Warning: Vault agent didn't exit gracefully. You need to shut it down manually"
fi

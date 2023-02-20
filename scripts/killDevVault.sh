#!/bin/sh

PID=$(ps aux | grep -v 'grep vault' | grep 'vault server -dev' | awk '{print $2}')
if [ "$PID" != "" ]; then
  kill $PID
  sleep 5
else
  echo "No test vault server running at this time"
fi

#!/bin/sh

PID=$(ps aux | grep vault | awk '{print $1}')
if [ "$PID" != "" ]; then
  kill $PID
fi
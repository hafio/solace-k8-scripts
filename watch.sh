#!/bin/bash

EXDIR=`dirname $0`
if [[ "$1" == "--env" ]] && [[ -n "$2" ]]; then
  watch -n 1 "${EXDIR}/get-op-status.sh $@; echo -e '\n'; ${EXDIR}/get-broker-status.sh $@"
else
  echo "Need to specify '--env [env]"
  exit 1
fi

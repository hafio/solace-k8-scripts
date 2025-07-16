#!/bin/bash

SELECT_ENV_FILE="000-env.sh"
if [[ -f "`dirname $0`/${SELECT_ENV_FILE}" ]]; then
	source "`dirname $0`/${SELECT_ENV_FILE}"
else 
	echo "Environment file '${SELECT_ENV_FILE}' not found"
	exit 1
fi

if [[ "$1" =~ (p|b|m) ]]; then 
  pod=$1
elif [[ -n "$1" ]]; then
  echo "Invalid node: $1"
  exit 1
else
  pod=p
fi


${KUBE} logs -n ${SOLOP_DERIVED_NS} pod/${SOLOP_DERIVED_POD} $@

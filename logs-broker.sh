#!/bin/bash

SELECT_ENV_FILE="000-env.sh"
if [[ -f "`dirname $0`/${SELECT_ENV_FILE}" ]]; then
	source "`dirname $0`/${SELECT_ENV_FILE}"
else 
	echo "Environment file '${SELECT_ENV_FILE}' not found"
	exit 1
fi

[[ "$1" =~ (p|b|m) ]] && (pod=$1; shift) || pod=p

${KUBE} logs -n ${SOLBK_NS} pod/${SOLBK_NAME}-pubsubplus-${pod}-0 $@

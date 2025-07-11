#!/bin/bash

SELECT_ENV_FILE="000-env.sh"
if [[ -f "`dirname $0`/${SELECT_ENV_FILE}" ]]; then
	source "`dirname $0`/${SELECT_ENV_FILE}"
else 
	echo "Environment file '${SELECT_ENV_FILE}' not found"
	exit 1
fi

[[ -z "$1" ]] && node=p || node=$1

${KUBE} exec -it -n ${SOLBK_NS} ${SOLBK_NAME}-pubsubplus-${node}-0 -- bash

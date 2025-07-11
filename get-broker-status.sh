#!/bin/bash

SELECT_ENV_FILE="000-env.sh"
if [[ -f "`dirname $0`/${SELECT_ENV_FILE}" ]]; then
	source "`dirname $0`/${SELECT_ENV_FILE}"
else 
	echo "Environment file '${SELECT_ENV_FILE}' not found"
	exit 1
fi

${KUBE} get pods -n ${SOLBK_NS} -o wide
echo
${KUBE} get svc -n ${SOLBK_NS}
echo
${KUBE} get statefulset -n ${SOLBK_NS}

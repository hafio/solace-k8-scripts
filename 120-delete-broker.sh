#!/bin/bash

SELECT_ENV_FILE="000-env.sh"
if [[ -f "`dirname $0`/${SELECT_ENV_FILE}" ]]; then
	source "`dirname $0`/${SELECT_ENV_FILE}"
else 
	echo "Environment file '${SELECT_ENV_FILE}' not found"
	exit 1
fi

TMPFILE=.tmp-${BASHPID}

echo "apiVersion: pubsubplus.solace.com/v1beta1
kind: PubSubPlusEventBroker
metadata:
  namespace: ${SOLBK_NS:-solace-namespace}
  name: ${SOLBK_NAME:-solace-event-broker}
" > ${TMPFILE}

${KUBE} delete -f ${TMPFILE}

echo "Deleting PVCs..."
${KUBE} delete pvc -n ${SOLBK_NS} data-${SOLBK_NAME}-pubsubplus-p-0 2> /dev/null
[[ "${SOLBK_REDUNDANCY:-false}" == true ]] && (
	${KUBE} delete pvc -n ${SOLBK_NS} data-${SOLBK_NAME}-pubsubplus-b-0 2> /dev/null
	${KUBE} delete pvc -n ${SOLBK_NS} data-${SOLBK_NAME}-pubsubplus-m-0 2> /dev/null
)
rm ${TMPFILE}

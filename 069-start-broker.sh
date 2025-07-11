#!/bin/bash

SELECT_ENV_FILE="000-env.sh"
if [[ -f "`dirname $0`/${SELECT_ENV_FILE}" ]]; then
	source "`dirname $0`/${SELECT_ENV_FILE}"
else 
	echo "Environment file '${SELECT_ENV_FILE}' not found"
	exit 1
fi

${KUBE} scale statefulset ${SOLBK_NAME}-pubsubplus-p -n ${SOLBK_NS} --replicas=1
echo "Waiting for primary node to come up..."
while [[ `${KUBE} get pods -n ${SOLBK_NS} 2> /dev/null | grep "${SOLBK_NAME}-pubsubplus-p" | grep Running | wc -l` -eq 0 ]]; do
  sleep 5
done

if [[ "${SOLBK_REDUNDANCY}" == "true" ]]; then
  ${KUBE} scale statefulset ${SOLBK_NAME}-pubsubplus-b -n ${SOLBK_NS} --replicas=1
  echo "Waiting for backup node to come up..."
  while [[ `${KUBE} get pods -n ${SOLBK_NS} 2> /dev/null | grep "${SOLBK_NAME}-pubsubplus-b" | grep Running | wc -l` -eq 0 ]]; do
    sleep 5
  done
  ${KUBE} scale statefulset ${SOLBK_NAME}-pubsubplus-m -n ${SOLBK_NS} --replicas=1
fi
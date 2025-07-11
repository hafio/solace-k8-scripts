#!/bin/bash

SELECT_ENV_FILE="000-env.sh"
if [[ -f "`dirname $0`/${SELECT_ENV_FILE}" ]]; then
	source "`dirname $0`/${SELECT_ENV_FILE}"
else 
	echo "Environment file '${SELECT_ENV_FILE}' not found"
	exit 1
fi

echo
echo "[WARNING] This script will scale each broker node's statefulset to 0"
echo -n "Confirm to proceed? [y/n] "
read -n 1 SCALEDOWN
echo 
if [[ ${SCALEDOWN} == "y" ]]; then
  ${KUBE} scale statefulset ${SOLBK_NAME}-pubsubplus-p -n ${SOLBK_NS} --replicas=0
  if [[ "${SOLBK_REDUNDANCY}" == "true" ]]; then
    ${KUBE} scale statefulset ${SOLBK_NAME}-pubsubplus-b -n ${SOLBK_NS} --replicas=0
    ${KUBE} scale statefulset ${SOLBK_NAME}-pubsubplus-m -n ${SOLBK_NS} --replicas=0
  fi
fi
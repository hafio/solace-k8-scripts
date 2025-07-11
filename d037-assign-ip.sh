#!/bin/bash

SELECT_ENV_FILE="000-env.sh"
if [[ -f "`dirname $0`/${SELECT_ENV_FILE}" ]]; then
	source "`dirname $0`/${SELECT_ENV_FILE}"
else 
	echo "Environment file '${SELECT_ENV_FILE}' not found"
	exit 1
fi

${KUBE} patch -n ${SOLBK_NS} service/${SOLBK_NAME}-pubsubplus -p '{"spec": { "loadBalancerIP": "'${SOLBK_LOADBALANCER_IP}'" }}'

#!/bin/bash

SELECT_ENV_FILE="000-env.sh"
if [[ -f "`dirname $0`/${SELECT_ENV_FILE}" ]]; then
	source "`dirname $0`/${SELECT_ENV_FILE}"
else 
	echo "Environment file '${SELECT_ENV_FILE}' not found"
	exit 1
fi

${KUBE} delete secret ${SOLBK_ADM_SECRET} -n ${SOLBK_NS}
[[ -n "${SOLBK_SVR_SECRET}" ]] && ${KUBE} delete secret ${SOLBK_SVR_SECRET} -n ${SOLBK_NS}
[[ -n "${IMAGEREPO_SECRET}" ]] && ${KUBE} delete secret ${IMAGEREPO_SECRET} -n ${SOLBK_NS}
${KUBE} get secret -n ${SOLBK_NS}

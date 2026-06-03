#!/bin/bash

echoUsage() {
  echo "Usage: $0
  Delete the broker secrets (admin credentials, TLS server cert, image-pull secret) from
  \$SOLBK_NS. Takes no positional arguments."
}

SELECT_ENV_FILE="000-env.sh"
if [[ -f "$(dirname "$0")/${SELECT_ENV_FILE}" ]]; then
	source "$(dirname "$0")/${SELECT_ENV_FILE}"
else 
	echo "Environment file '${SELECT_ENV_FILE}' not found"
	exit 1
fi

${KUBE} delete secret ${SOLBK_USR_SECRET} -n ${SOLBK_NS} --ignore-not-found
[[ -n "${SOLBK_SVR_SECRET}" ]] && ${KUBE} delete secret ${SOLBK_SVR_SECRET} -n ${SOLBK_NS} --ignore-not-found
[[ -n "${IMAGEREPO_SECRET}" ]] && ${KUBE} delete secret ${IMAGEREPO_SECRET} -n ${SOLBK_NS} --ignore-not-found
${KUBE} get secret -n ${SOLBK_NS}

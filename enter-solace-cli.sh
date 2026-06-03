#!/bin/bash

echoUsage() {
  echo "Usage: $0 [p|b|m]
  Open an interactive Solace CLI session on the chosen broker pod (p=primary, b=backup, m=monitor). Default: p."
}

SELECT_ENV_FILE="000-env.sh"
if [[ -f "$(dirname "$0")/${SELECT_ENV_FILE}" ]]; then
	source "$(dirname "$0")/${SELECT_ENV_FILE}"
else 
	echo "Environment file '${SELECT_ENV_FILE}' not found"
	exit 1
fi

pod=$(pick_pod "$1") || exit 1

${KUBE} exec -it -n ${SOLBK_NS} ${SOLBK_NAME}-pubsubplus-${pod}-0 -- cli -A

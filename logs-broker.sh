#!/bin/bash

echoUsage() {
  echo "Usage: $0 [p|b|m] [kubectl-logs-args...]
  Show logs for the chosen broker pod (p=primary, b=backup, m=monitor; default p).
  Any extra arguments are passed through to 'kubectl logs' (e.g. -f, --tail=100)."
}

SELECT_ENV_FILE="000-env.sh"
if [[ -f "$(dirname "$0")/${SELECT_ENV_FILE}" ]]; then
	source "$(dirname "$0")/${SELECT_ENV_FILE}"
else 
	echo "Environment file '${SELECT_ENV_FILE}' not found"
	exit 1
fi

pod=$(pick_pod "$1") || exit 1
[[ -n "$1" ]] && shift

${KUBE} logs -n ${SOLBK_NS} pod/${SOLBK_NAME}-pubsubplus-${pod}-0 $@

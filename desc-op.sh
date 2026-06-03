#!/bin/bash

echoUsage() {
  echo "Usage: $0
  Describe the Solace operator pod(s) in the operator namespace. Takes no positional arguments."
}

SELECT_ENV_FILE="000-env.sh"
if [[ -f "$(dirname "$0")/${SELECT_ENV_FILE}" ]]; then
	source "$(dirname "$0")/${SELECT_ENV_FILE}"
else 
	echo "Environment file '${SELECT_ENV_FILE}' not found"
	exit 1
fi

${KUBE} describe pods -n ${SOLOP_DERIVED_NS}

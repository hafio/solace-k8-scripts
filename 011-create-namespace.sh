#!/bin/bash

echoUsage() {
  echo "Usage: $0 [OPTIONS]
  Create the broker Kubernetes namespace (\$SOLBK_NS).
  OPTIONS:
    --only-gen-yaml : print the generated manifest to stdout instead of applying it"
}

SELECT_ENV_FILE="000-env.sh"
if [[ -f "$(dirname "$0")/${SELECT_ENV_FILE}" ]]; then
	source "$(dirname "$0")/${SELECT_ENV_FILE}"
else 
	echo "Environment file '${SELECT_ENV_FILE}' not found"
	exit 1
fi

gen_yaml() {
  ${KUBE} create ns ${SOLBK_NS} --dry-run=client -o yaml
}

if [[ "${GENONLY}" == "true" ]]; then
  gen_yaml
  exit 0
fi

gen_yaml | ${KUBE} apply -f -

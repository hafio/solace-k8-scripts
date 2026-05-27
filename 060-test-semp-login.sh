#!/bin/bash

SELECT_ENV_FILE="000-env.sh"
if [[ -f "$(dirname "$0")/${SELECT_ENV_FILE}" ]]; then
	source "$(dirname "$0")/${SELECT_ENV_FILE}"
else
	echo "Environment file '${SELECT_ENV_FILE}' not found"
	exit 1
fi

node=$(pick_pod "$1") || exit 1

echo -n "Enter SEMP username: "
read SEMP_USER
echo -n "Enter SEMP password: "
read -s SEMP_PASS
echo

SEMP_USER=${SEMP_USER:-admin}
SEMP_PASS=${SEMP_PASS:-${SOLBK_ADM_PASS}}

# Run curl inside the broker pod via kubectl exec; capture response locally.
# No file is written to the pod, so credentials never persist on its filesystem.
HTTP_OUT=$(${KUBE} exec -n ${SOLBK_NS} ${SOLBK_NAME}-pubsubplus-${node}-0 -- \
  curl -is -u "${SEMP_USER}:${SEMP_PASS}" http://localhost:8080/SEMP/v2/monitor 2>/dev/null)
HTTP_STATUS=$(echo "${HTTP_OUT}" | grep -i '^HTTP/' | head -1 | tr -d '\r')

if [[ "${HTTP_STATUS}" =~ \ 2[0-9][0-9]( |$) ]]; then
  echo "Login OK"
else
  echo "Login failed. Reason: ${HTTP_STATUS:-<no HTTP response from broker>}"
  echo -n "Do you want to display the username & password? [y/n] "
  read -n 1 display
  echo
  if [[ "${display,,}" == "y" ]]; then
    echo "Username: ${SEMP_USER}"
    echo "Password: ${SEMP_PASS}"
    echo "cURL Command: curl -i -s -u ${SEMP_USER}:${SEMP_PASS} http://localhost:8080/SEMP/v2/monitor"
  fi
fi

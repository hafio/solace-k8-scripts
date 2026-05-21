#!/bin/bash

SELECT_ENV_FILE="000-env.sh"
if [[ -f "$(dirname "$0")/${SELECT_ENV_FILE}" ]]; then
	source "$(dirname "$0")/${SELECT_ENV_FILE}"
else 
	echo "Environment file '${SELECT_ENV_FILE}' not found"
	exit 1
fi

[[ -n "${SOLBK_SVR_SECRET}" ]] && [[ ! -f "${SOLBK_TLS_CERT}" ]] && errmsg+="\n\t${SOLBK_TLS_CERT} not found"
[[ -n "${SOLBK_SVR_SECRET}" ]] && [[ ! -f "${SOLBK_TLS_CERTKEY}" ]] && errmsg+="\n\t${SOLBK_TLS_CERTKEY} not found"
if [[ -n "${errmsg}" ]]; then
	echo -e "Error:${errmsg}"
	exit 1
fi

CLI_USR_PASS=(--from-literal="username_admin_password=${SOLBK_ADM_PASS}")
[[ -n "${SOLBK_MON_PASS}" ]] && CLI_USR_PASS+=(--from-literal="username_monitor_password=${SOLBK_MON_PASS}")
for up in "${SOLBK_USR_PASS[@]}"; do
	user="${up%%=*}"
	pass="${up#*=}"
	CLI_USR_PASS+=(--from-literal="username_${user}_password=${pass}")
done
${KUBE} create secret generic "${SOLBK_USR_SECRET}" -n "${SOLBK_NS}" "${CLI_USR_PASS[@]}"
if [[ $? -eq 0 ]]; then
	echo "Solace Admin Secret '${SOLBK_USR_SECRET}' created successfully."
else
	echo "Error creating Solace Admin Secret '${SOLBK_USR_SECRET}'."
	exit 1
fi

if [[ -n "${SOLBK_SVR_SECRET}" ]]; then
  ${KUBE} create secret tls ${SOLBK_SVR_SECRET} -n ${SOLBK_NS} --cert <(cat ${SOLBK_TLS_CERT} ${SOLBK_TLS_CERTCAS[@]}) --key=${SOLBK_TLS_CERTKEY}
  if [[ $? -eq 0 ]]; then
    echo "Solace TLS Certificate Secret '${SOLBK_SVR_SECRET}' created successfully."
  else
    echo "Error creating Solace TLS Certificate Secret '${SOLBK_SVR_SECRET}'."
    exit 1
  fi
fi

if [[ -n "${IMAGEREPO_SECRET}" ]]; then
  ${KUBE} create secret docker-registry ${IMAGEREPO_SECRET} -n ${SOLBK_NS} --docker-server="${IMAGEREPO_HOST}" --docker-username="${IMAGEREPO_USER}" --docker-password="${IMAGEREPO_PASS}"
  if [[ $? -eq 0 ]]; then
    echo "Image Repository Secret '${IMAGEREPO_SECRET}' created successfully."
  else
    echo "Error creating Image Repository Secret '${IMAGEREPO_SECRET}'."
    exit 1
  fi
fi

${KUBE} get secret -n ${SOLBK_NS}

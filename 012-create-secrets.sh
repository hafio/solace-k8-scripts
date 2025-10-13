#!/bin/bash

SELECT_ENV_FILE="000-env.sh"
if [[ -f "`dirname $0`/${SELECT_ENV_FILE}" ]]; then
	source "`dirname $0`/${SELECT_ENV_FILE}"
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

${KUBE} create secret generic ${SOLBK_ADM_SECRET} -n ${SOLBK_NS} --from-literal="username_admin_password=${SOLBK_ADM_PASS}"
if [[ $? -eq 0 ]]; then
	echo "Solace Admin Secret '${SOLBK_ADM_SECRET}' created successfully."
else
	echo "Error creating Solace Admin Secret '${SOLBK_ADM_SECRET}'."
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

KEYVALS="--from-literal=\"k=v\""
[[ -n "${SOLBK_PRODUCTKEY}" ]] && KEYVALS+=" --from-literal=\"productkey=\${SOLBK_PRODUCTKEY} "

${KUBE} create secret generic ${SOLBK_CONFIGMAP} -n ${SOLBK_NS} ${KEYVALS}
if [[ $? -eq 0 ]]; then
  echo "Secret Config Map '${SOLBK_CONFIGMAP}' created successfully."
else
  echo "Error creating Secret Config Map '${SOLBK_CONFIGMAP}'."
  exit 1
fi

${KUBE} get secret -n ${SOLBK_NS}

#!/bin/bash

SELECT_ENV_FILE="000-env.sh"
if [[ -f "`dirname $0`/${SELECT_ENV_FILE}" ]]; then
	source "`dirname $0`/${SELECT_ENV_FILE}"
else 
	echo "Environment file '${SELECT_ENV_FILE}' not found"
	exit 1
fi

DT=`date +%Y-%m-%d`

if [[ -z "${SOLBK_TLS_CERT}" ]] || [[ -z "${SOLBK_TLS_CERTKEY}" ]]; then
  echo '[Error] $SOLBK_TLS_CERT & $SOLBK_TLS_CERTKEY must be specified!'
  exit 1
elif [[ ! -f "${SOLBK_TLS_CERT}" ]] || [[ ! -f "${SOLBK_TLS_CERTKEY}" ]]; then
  echo '[Error] Both $SOLBK_TLS_CERT & $SOLBK_TLS_CERT must be valid files!'
  exit 1
else
    echo "enable
configure
ssl server-certificate tls-${DT}.crt.key
show ssl server-certificate detail" > .cert
  cat "${SOLBK_TLS_CERTKEY}" "${SOLBK_TLS_CERT}" "${SOLBK_TLS_CERTCAS[@]}" > .tls

  nodes=("p")
  [[ "${SOLBK_REDUNDANCY}" == "true" ]] && nodes=("p" "b" "m")
  
  for node in "${nodes[@]}"; do
    echo "Copying files into '${node}' node..."
    ${KUBE} cp -n ${SOLBK_NS} .tls ${SOLBK_NAME}-pubsubplus-${node}-0:/usr/sw/jail/certs/tls-${DT}.crt.key
    if [[ $? -eq 0 ]]; then
      ${KUBE} cp -n ${SOLBK_NS} .cert ${SOLBK_NAME}-pubsubplus-${node}-0:/usr/sw/jail/cliscripts/.apply-server-certs.cli
      if [[ $? -eq 0 ]]; then
        echo "Applying server certificate in '${node}' node..."
        ${KUBE} exec -n ${SOLBK_NS} ${SOLBK_NAME}-pubsubplus-${node}-0 -- /usr/sw/loads/currentload/bin/cli -Apes .apply-server-certs.cli | head -50 | tail -30        
      else
        echo "[Error] Unable to copy the cliscript into the broker."
        exit 1
      fi
    else
      echo "[Error] Unable to copy the certificates into the broker."
      exit 1
    fi
  done
fi

rm -f .tls .cert
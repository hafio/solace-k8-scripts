#!/bin/bash

SELECT_ENV_FILE="000-env.sh"
if [[ -f "`dirname $0`/${SELECT_ENV_FILE}" ]]; then
	source "`dirname $0`/${SELECT_ENV_FILE}"
else 
	echo "Environment file '${SELECT_ENV_FILE}' not found"
	exit 1
fi

TMPFILE=.tmp-${BASHPID}

if [[ -z "${SOLBK_PRODUCTKEYS[@]}" ]]; then
  echo "No Product Key(s) specified in \$SOLBK_PRODUCTKEYS"
  exit 1
else 
  echo "enable
admin" > ${TMPFILE}
  for PK in "${SOLBK_PRODUCTKEYS[@]}"; do
    echo "product-key ${PK}" >> ${TMPFILE}
  done
  echo "show product-key" >> ${TMPFILE}
  
  echo "Copying CLI script(s)..."
  ${KUBE} cp -n ${SOLBK_NS} ${TMPFILE} ${SOLBK_NAME}-pubsubplus-p-0:/usr/sw/jail/cliscripts/.product-keys.cli > cli.out
  [[ "${SOLBK_REDUNDANCY}" == "true" ]] && ${KUBE} cp -n ${SOLBK_NS} ${TMPFILE} ${SOLBK_NAME}-pubsubplus-b-0:/usr/sw/jail/cliscripts/.product-keys.cli >> cli.out
  
  echo "Executing CLI script(s)..."
  ${KUBE} exec -it -n ${SOLBK_NS} ${SOLBK_NAME}-pubsubplus-p-0 -- /usr/sw/loads/currentload/bin/cli -Apes .product-keys.cli >> cli.out
  [[ "${SOLBK_REDUNDANCY}" == "true" ]] && ${KUBE} exec -it -n ${SOLBK_NS} ${SOLBK_NAME}-pubsubplus-b-0 -- /usr/sw/loads/currentload/bin/cli -Apes .product-keys.cli >> cli.out
  
  echo "Cleaning up CLI script(s)..."
  ${KUBE} exec -it -n ${SOLBK_NS} ${SOLBK_NAME}-pubsubplus-p-0 -- rm -f /usr/sw/jail/cliscripts/.product-keys.cli >> cli.out
  [[ "${SOLBK_REDUNDANCY}" == "true" ]] && ${KUBE} exec -it -n ${SOLBK_NS} ${SOLBK_NAME}-pubsubplus-b-0 -- rm -f /usr/sw/jail/cliscripts/.product-keys.cli >> cli.out
  rm -f ${TMPFILE}
  
  if [[ `egrep -i 'error|fail' cli.out | wc -l` -ge 1 ]]; then
    echo "[Error] Error detected. Please check cli.out for output logs."
  else
    echo "Output is captured in cli.out."
  fi
fi
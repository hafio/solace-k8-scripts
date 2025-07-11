#!/bin/bash

SELECT_ENV_FILE="000-env.sh"
if [[ -f "`dirname $0`/${SELECT_ENV_FILE}" ]]; then
	source "`dirname $0`/${SELECT_ENV_FILE}"
else 
	echo "Environment file '${SELECT_ENV_FILE}' not found"
	exit 1
fi

if [[ ! -r "${SOLBK_CLISCRIPTS_FOLDER}" ]]; then
  echo  "[Error] ${SOLBK_CLISCRIPTS_FOLDER} is not accessible or valid."
  exit 1
fi

if [[ -z "$1" ]]; then
  CLI_FILES=(`ls ${SOLBK_CLISCRIPTS_FOLDER}`)
  while [[ ! "${CLI_IND}" =~ [0-9]+ ]]; do
    for i in "${!CLI_FILES[@]}"; do
      echo "[${i}] ${CLI_FILES[$i]}"
    done
    echo -n "Which cli script to execute: "
    read CLI_IND
    echo ""
    CLI="${CLI_FILES[${CLI_IND}]}"
  done
else
  CLI="$1"
fi

if [[ ! -f "${SOLBK_CLISCRIPTS_FOLDER}/${CLI}" ]]; then
	echo "Usage: $0 <script filename in '"'$SOLBK_CLISCRIPTS_FOLDER'"' folder>"
	exit 1
else
	${KUBE} cp -n ${SOLBK_NS} "${SOLBK_CLISCRIPTS_FOLDER}/${CLI}" "${SOLBK_NAME}-pubsubplus-p-0:/usr/sw/jail/cliscripts/.${CLI}"
  if [[ $? -eq 0 ]]; then
	  ${KUBE} exec -n ${SOLBK_NS} ${SOLBK_NAME}-pubsubplus-p-0 -- /usr/sw/loads/currentload/bin/cli -Apes ".${CLI}"
	  ${KUBE} exec -n ${SOLBK_NS} ${SOLBK_NAME}-pubsubplus-p-0 -- rm -f "/usr/sw/jail/cliscripts/.${CLI}"
  else
    echo "[Error] Unable to copy script"
  fi
fi
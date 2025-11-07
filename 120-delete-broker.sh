#!/bin/bash

SELECT_ENV_FILE="000-env.sh"
if [[ -f "`dirname $0`/${SELECT_ENV_FILE}" ]]; then
	source "`dirname $0`/${SELECT_ENV_FILE}"
else 
	echo "Environment file '${SELECT_ENV_FILE}' not found"
	exit 1
fi

if [[ "${1:0:1}" == "?" ]] || [[ "${1:0:2}" == "-h" ]] || [[ "${1:0:3}" == "--h" ]]; then
	echo "Usage: $0 [OPTIONS]"
	echo "	OPTIONS:"
  echo "    --no-prompt: does not prompt the user for confirmation. Removes Solace Broker deployment and PVCs."
  echo "    --keep-pvc: does not remove Persistent Volume Claims of the broker."
	exit
fi

TMPFILE=.tmp-${BASHPID}

echo "apiVersion: pubsubplus.solace.com/v1beta1
kind: PubSubPlusEventBroker
metadata:
  namespace: ${SOLBK_NS:-solace-namespace}
  name: ${SOLBK_NAME:-solace-event-broker}
" > ${TMPFILE}

while [[ $# -gt 0 ]]; do
	case "$1" in
		--keep-pvc)
			DEL_PVC=n
			shift 1
			;;
    --no-prompt)
			DEL_PVC=y
      CONF=y
      ;;
		*)
			shift
			;;
	esac
done

while ! [[ "${CONF,,}" =~ [yn] ]]; do
  echo "Solace deployment: \"${SOLBK_NAME}\""
  echo " Solace namespace: \"${SOLBK_NS}\""
  echo -n "Proceed to delete? [y/n] "
  read -n 1 CONF
  echo
done

if [[ "${CONF}" == "y" ]]; then
  ${KUBE} delete -f ${TMPFILE}
  
  while ! [[ "${DEL_PVC,,}" =~ [yn] ]]; do
    echo -n "Delete PVCs? [y/n] "
    read -n 1 DEL_PVC
    echo
  done
  
  if [[ "${DEL_PVC,,}" == "y" ]]; then
    echo "Deleting PVCs..."
    ${KUBE} delete pvc -n ${SOLBK_NS} data-${SOLBK_NAME}-pubsubplus-p-0 2> /dev/null
    [[ "${SOLBK_REDUNDANCY:-false}" == true ]] && (
      ${KUBE} delete pvc -n ${SOLBK_NS} data-${SOLBK_NAME}-pubsubplus-b-0 2> /dev/null
      ${KUBE} delete pvc -n ${SOLBK_NS} data-${SOLBK_NAME}-pubsubplus-m-0 2> /dev/null
    )
  fi  
fi

rm ${TMPFILE}
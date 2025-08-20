#!/bin/bash

SELECT_ENV_FILE="000-env.sh"
if [[ -f "`dirname $0`/${SELECT_ENV_FILE}" ]]; then
	source "`dirname $0`/${SELECT_ENV_FILE}"
else 
	echo "Environment file '${SELECT_ENV_FILE}' not found"
	exit 1
fi

TMPFILE=.tmp-${BASHPID}

if [[ "${SOLBK_REDUNDANCY}" == "true" ]]; then

  # revert activity cli
  echo 'home
no paging
enable
admin
redundancy revert-activity' > ${TMPFILE}
  
  # copy revert activity cli and execute
  ${KUBE} cp -n ${SOLBK_NS} ${TMPFILE} ${SOLBK_NAME}-pubsubplus-b-0:/usr/sw/jail/cliscripts/.revert-activity.cli
  ${KUBE} exec -it -n ${SOLBK_NS} ${SOLBK_NAME}-pubsubplus-b-0 -- /usr/sw/loads/currentload/bin/cli -Apes .revert-activity.cli > /dev/null

  # create show redundancy cli and copy
  echo 'show redundancy' > ${TMPFILE}
  ${KUBE} cp -n ${SOLBK_NS} ${TMPFILE} ${SOLBK_NAME}-pubsubplus-p-0:/usr/sw/jail/cliscripts/.show-redundancy.cli

  # create assert leader redundancy cli and copy
  echo 'home
no paging
enable
admin
config-sync assert-leader router
config-sync assert-leader message-vpn *
show config-sync database
' > ${TMPFILE}
  ${KUBE} cp -n ${SOLBK_NS} ${TMPFILE} ${SOLBK_NAME}-pubsubplus-p-0:/usr/sw/jail/cliscripts/.assert-leader.cli
  
  
  echo "Waiting for redundancy state to be restored fully..."
  for i in {0..10}; do
    echo -ne `date`'\r'
	  ${KUBE} exec -n ${SOLBK_NS} ${SOLBK_NAME}-pubsubplus-p-0 -- /usr/sw/loads/currentload/bin/cli -Apes .show-redundancy.cli > ${TMPFILE}
    CFG_ST=`grep "Configuration Status" ${TMPFILE}`
    RDC_ST=`grep "Redundancy Status" ${TMPFILE}`
    RDC_RL=`grep "Active-Standby Role" ${TMPFILE}`
    ADB_LK=`grep "ADB Link To Mate" ${TMPFILE}`
    ADB_MT=`grep "ADB Hello To Mate" ${TMPFILE}`
    if [[ "${CFG_ST#*: }" == "Enabled" ]] && [[ "${RDC_ST#*: }" == "Up" ]] && [[ "${RDC_RL#*: }" == "Primary" ]] && [[ "${ADB_LK#*: }" == "Up" ]] && [[ "${ADB_MT#*: }" == "Up" ]]; then
      echo ""
      ${KUBE} exec -n ${SOLBK_NS} ${SOLBK_NAME}-pubsubplus-p-0 -- /usr/sw/loads/currentload/bin/cli -Apes .assert-leader.cli | tail -12
      rm ${TMPFILE}
      exit 0
    fi
    sleep 2
  done
  echo "[Error] Timeout!"
  echo 'no paging
show redundancy detail' > ${TMPFILE}
  ${KUBE} cp -n ${SOLBK_NS} ${TMPFILE} ${SOLBK_NAME}-pubsubplus-p-0:/usr/sw/jail/cliscripts/.show-redundancy-detail.cli
  ${KUBE} exec -n ${SOLBK_NS} ${SOLBK_NAME}-pubsubplus-p-0 -- /usr/sw/loads/currentload/bin/cli -Apes .show-redundancy-detail.cli
  rm ${TMPFILE}
else
  echo "Standalone Broker Deployment Mode detected!"
fi
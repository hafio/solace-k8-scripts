#!/bin/bash

SELECT_ENV_FILE="000-env.sh"
if [[ -f "`dirname $0`/${SELECT_ENV_FILE}" ]]; then
	source "`dirname $0`/${SELECT_ENV_FILE}"
else 
	echo "Environment file '${SELECT_ENV_FILE}' not found"
	exit 1
fi

if [[ "${SOLBK_REDUNDANCY}" == "true" ]]; then

  # revert activity cli
  echo 'home
no paging
enable
admin
redundancy revert-activity' > .tmp
  
  # copy revert activity cli and execute
  ${KUBE} cp -n ${SOLBK_NS} .tmp ${SOLBK_NAME}-pubsubplus-b-0:/usr/sw/jail/cliscripts/.revert-activity.cli
  ${KUBE} exec -it -n ${SOLBK_NS} ${SOLBK_NAME}-pubsubplus-b-0 -- /usr/sw/loads/currentload/bin/cli -Apes .revert-activity.cli > /dev/null

  # create show redundancy cli and copy
  echo 'show redundancy' > .tmp
  ${KUBE} cp -n ${SOLBK_NS} .tmp ${SOLBK_NAME}-pubsubplus-p-0:/usr/sw/jail/cliscripts/.show-redundancy.cli

  # create assert leader redundancy cli and copy
  echo 'home
no paging
enable
admin
config-sync assert-leader router
config-sync assert-leader message-vpn *
show config-sync database
' > .tmp
  ${KUBE} cp -n ${SOLBK_NS} .tmp ${SOLBK_NAME}-pubsubplus-p-0:/usr/sw/jail/cliscripts/.assert-leader.cli
  
  
  echo "Waiting for redundancy state to be restored fully..."
  for i in {0..10}; do
    echo -ne `date`'\r'
	  ${KUBE} exec -n ${SOLBK_NS} ${SOLBK_NAME}-pubsubplus-p-0 -- /usr/sw/loads/currentload/bin/cli -Apes .show-redundancy.cli > .tmp
    CFG_ST=`grep "Configuration Status" .tmp`
    RDC_ST=`grep "Redundancy Status" .tmp`
    RDC_RL=`grep "Active-Standby Role" .tmp`
    ADB_LK=`grep "ADB Link To Mate" .tmp`
    ADB_MT=`grep "ADB Hello To Mate" .tmp`
    if [[ "${CFG_ST#*: }" == "Enabled" ]] && [[ "${RDC_ST#*: }" == "Up" ]] && [[ "${RDC_RL#*: }" == "Primary" ]] && [[ "${ADB_LK#*: }" == "Up" ]] && [[ "${ADB_MT#*: }" == "Up" ]]; then
      echo ""
      ${KUBE} exec -n ${SOLBK_NS} ${SOLBK_NAME}-pubsubplus-p-0 -- /usr/sw/loads/currentload/bin/cli -Apes .assert-leader.cli | tail -12
      rm .tmp
      exit 0
    fi
    sleep 2
  done
  echo "[Error] Timeout!"
  echo 'no paging
show redundancy detail' > .tmp
  ${KUBE} cp -n ${SOLBK_NS} .tmp ${SOLBK_NAME}-pubsubplus-p-0:/usr/sw/jail/cliscripts/.show-redundancy-detail.cli
  ${KUBE} exec -n ${SOLBK_NS} ${SOLBK_NAME}-pubsubplus-p-0 -- /usr/sw/loads/currentload/bin/cli -Apes .show-redundancy-detail.cli
  rm .tmp
else
  echo "Standalone Broker Deployment Mode detected!"
fi
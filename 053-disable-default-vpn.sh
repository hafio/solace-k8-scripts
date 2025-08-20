#!/bin/bash

SELECT_ENV_FILE="000-env.sh"
if [[ -f "`dirname $0`/${SELECT_ENV_FILE}" ]]; then
	source "`dirname $0`/${SELECT_ENV_FILE}"
else 
	echo "Environment file '${SELECT_ENV_FILE}' not found"
	exit 1
fi

TMPFILE=.tmp-${BASHPID}

echo 'home
enable
configure

message-vpn "default"
  authentication
    basic shutdown
    client-certificate
      shutdown
      exit
    exit
  semp-over-msgbus shutdown
  semp-over-msgbus show-cmds shutdown
  semp-over-msgbus admin-cmds shutdown
  semp-over-msgbus admin-cmds distributed-cache-cmds shutdown
  semp-over-msgbus admin-cmds client-cmds shutdown
  service smf plain-text shutdown
  service smf ssl shutdown
  service web-transport plain-text shutdown
  service web-transport ssl shutdown
  service rest incoming plain-text shutdown
  no service rest incoming listen-port
  service rest incoming ssl shutdown
  no service rest incoming listen-port ssl
  service mqtt plain-text shutdown
  no service mqtt listen-port
  service mqtt ssl shutdown
  no service mqtt listen-port ssl
  service mqtt websocket shutdown
  no service mqtt listen-port web
  service mqtt websocket-secure shutdown
  no service mqtt listen-port ssl web
  service amqp plain-text shutdown
  no service amqp listen-port
  service amqp ssl shutdown
  no service amqp listen-port ssl
  no ssl allow-downgrade-to-plain-text
  exit

client-username "default" message-vpn "default"
  shutdown
  exit

message-vpn "default"
  shutdown
  exit
' > ${TMPFILE}
  
  # copy revert activity cli and execute
  echo ${KUBE} cp -n ${SOLBK_NS} ${TMPFILE} ${SOLBK_NAME}-pubsubplus-b-0:/usr/sw/jail/cliscripts/.revert-activity.cli
  echo ${KUBE} exec -it -n ${SOLBK_NS} ${SOLBK_NAME}-pubsubplus-b-0 -- /usr/sw/loads/currentload/bin/cli -Apes .revert-activity.cli > /dev/null

  # create show redundancy cli and copy
  echo 'show redundancy'  ${TMPFILE}
  echo ${KUBE} cp -n ${SOLBK_NS} ${TMPFILE} ${SOLBK_NAME}-pubsubplus-p-0:/usr/sw/jail/cliscripts/.show-redundancy.cli

  echo ${KUBE} cp -n ${SOLBK_NS} ${TMPFILE} ${SOLBK_NAME}-pubsubplus-p-0:/usr/sw/jail/cliscripts/.assert-leader.cli
  
  
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
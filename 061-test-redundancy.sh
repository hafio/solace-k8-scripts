#!/bin/bash

SELECT_ENV_FILE="000-env.sh"
if [[ -f "`dirname $0`/${SELECT_ENV_FILE}" ]]; then
	source "`dirname $0`/${SELECT_ENV_FILE}"
else 
	echo "Environment file '${SELECT_ENV_FILE}' not found"
	exit 1
fi

set -e

if [[ "${SOLBK_REDUNDANCY}" == "true" ]]; then

  # create show redundancy cli file
  echo "show redundancy" > .tmp-${BASHPID}
  ${KUBE} cp -n ${SOLBK_NS} .tmp-${BASHPID} ${SOLBK_NAME}-pubsubplus-p-0:/usr/sw/jail/cliscripts/.show-rd.cli
  ${KUBE} cp -n ${SOLBK_NS} .tmp-${BASHPID} ${SOLBK_NAME}-pubsubplus-b-0:/usr/sw/jail/cliscripts/.show-rd.cli

  # create release activity cli file
  echo "home
no paging
enable
configure
redundancy release-activity" > .tmp-${BASHPID}
  ${KUBE} cp -n ${SOLBK_NS} .tmp-${BASHPID} ${SOLBK_NAME}-pubsubplus-p-0:/usr/sw/jail/cliscripts/.release.cli

  # create release activity cli file
  echo "home
no paging
enable
configure
no redundancy release-activity" > .tmp-${BASHPID}
  ${KUBE} cp -n ${SOLBK_NS} .tmp-${BASHPID} ${SOLBK_NAME}-pubsubplus-p-0:/usr/sw/jail/cliscripts/.no-release.cli

  # create revert activity cli file
  echo "home
no paging
enable
admin
redundancy revert-activity " > .tmp-${BASHPID}
  ${KUBE} cp -n ${SOLBK_NS} .tmp-${BASHPID} ${SOLBK_NAME}-pubsubplus-b-0:/usr/sw/jail/cliscripts/.revert-activity.cli

  # ensure redundancy is up and running properly
  ${KUBE} exec -n ${SOLBK_NS} ${SOLBK_NAME}-pubsubplus-p-0 -- /usr/sw/loads/currentload/bin/cli -Apes .show-rd.cli > .tmp-${BASHPID}
  CF_STS=`grep "Configuration Status" .tmp-${BASHPID}`
  RD_STS=`grep "Redundancy Status" .tmp-${BASHPID}`
  RD_ROL=`grep "Active-Standby Role" .tmp-${BASHPID}`
  ADB_LK=`grep "ADB Link To Mate" .tmp-${BASHPID}`
  ADB_HL=`grep "ADB Hello To Mate" .tmp-${BASHPID}`
  
  if [[ "${CF_STS#*: }" == "Enabled" ]] && [[ "${RD_STS#*: }" == "Up" ]] && [[ "${RD_ROL#*: }" == "Primary" ]] && [[ "${ADB_LK#*: }" == "Up" ]] && [[ "${ADB_HL#*: }" == "Up" ]]; then
    IS_ACT=`grep "Activity Status" .tmp-${BASHPID} | grep "Local Active" | wc -l` 
    if [[ ${IS_ACT} -eq 1 ]]; then
      echo "[Info] Detected Primary Node is active."
      
      # release primary
      ${KUBE} exec -n ${SOLBK_NS} ${SOLBK_NAME}-pubsubplus-p-0 -- /usr/sw/loads/currentload/bin/cli -Apes .release.cli > /dev/null
      while [[ "${CF_STS#*: }" != "Enabled-Released" ]] || [[ "${RD_STS#*: }" != "Down" ]] || [[ ${MT_ACT} -ne 1 ]]; do
        echo -ne [`date`] Waiting for Primary Node to be released...'\r'
        ${KUBE} exec -n ${SOLBK_NS} ${SOLBK_NAME}-pubsubplus-p-0 -- /usr/sw/loads/currentload/bin/cli -Apes .show-rd.cli > .tmp-${BASHPID}
        CF_STS=`grep "Configuration Status" .tmp-${BASHPID}`
        RD_STS=`grep "Redundancy Status" .tmp-${BASHPID}`
        MT_ACT=`grep "Activity Status" .tmp-${BASHPID} | grep "Mate Active" | wc -l`
        sleep 3
      done
      echo "[Info] Primary node is released. Backup node is active.                                    "
      
      # no release primary
      ${KUBE} exec -n ${SOLBK_NS} ${SOLBK_NAME}-pubsubplus-p-0 -- /usr/sw/loads/currentload/bin/cli -Apes .no-release.cli > /dev/null
      while [[ "${CF_STS#*: }" != "Enabled" ]] || [[ "${RD_STS#*: }" != "Up" ]] || [[ ${MT_ACT} -ne 1 ]] || [[ ${BK_ACT} -ne 1 ]]; do
        echo -ne [`date`] Waiting for Primary Node to be un-released...'\r'
        ${KUBE} exec -n ${SOLBK_NS} ${SOLBK_NAME}-pubsubplus-p-0 -- /usr/sw/loads/currentload/bin/cli -Apes .show-rd.cli > .tmp-${BASHPID}
        CF_STS=`grep "Configuration Status" .tmp-${BASHPID}`
        RD_STS=`grep "Redundancy Status" .tmp-${BASHPID}`
        MT_ACT=`grep "Activity Status" .tmp-${BASHPID} | grep "Mate Active" | wc -l`
        
        ${KUBE} exec -n ${SOLBK_NS} ${SOLBK_NAME}-pubsubplus-b-0 -- /usr/sw/loads/currentload/bin/cli -Apes .show-rd.cli > .tmp-${BASHPID}
        BK_ACT=`grep "Activity Status" .tmp-${BASHPID} | grep "Local Active" | wc -l`
        sleep 3
      done
      echo "[Info] Primary node is 'un-released'. Backup node is active.                               "
    fi
    ${KUBE} exec -n ${SOLBK_NS} ${SOLBK_NAME}-pubsubplus-b-0 -- /usr/sw/loads/currentload/bin/cli -Apes .show-rd.cli > .tmp-${BASHPID}
    BK_ACT=`grep "Activity Status" .tmp-${BASHPID} | grep "Local Active" | wc -l`
    if [[ ${BK_ACT} -eq 1 ]]; then
      echo "[Info] Detected Backup node is active."
      ${KUBE} exec -n ${SOLBK_NS} ${SOLBK_NAME}-pubsubplus-b-0 -- /usr/sw/loads/currentload/bin/cli -Apes .revert-activity.cli > /dev/null
      while [[ "${CF_STS#*: }" != "Enabled" ]] || [[ "${RD_STS#*: }" != "Up" ]] || [[ ${MT_ACT} -ne 0 ]] || [[ ${IS_ACT} -ne 1 ]] || [[ ${BK_ACT} -ne 0 ]]; do
        echo -ne [`date`] Waiting for Primary Node to become active...'\r'
        ${KUBE} exec -n ${SOLBK_NS} ${SOLBK_NAME}-pubsubplus-p-0 -- /usr/sw/loads/currentload/bin/cli -Apes .show-rd.cli > .tmp-${BASHPID}
        CF_STS=`grep "Configuration Status" .tmp-${BASHPID}`
        RD_STS=`grep "Redundancy Status" .tmp-${BASHPID}`
        IS_ACT=`grep "Activity Status" .tmp-${BASHPID} | grep "Local Active" | wc -l`
        MT_ACT=`grep "Activity Status" .tmp-${BASHPID} | grep "Mate Active" | wc -l`

        ${KUBE} exec -n ${SOLBK_NS} ${SOLBK_NAME}-pubsubplus-b-0 -- /usr/sw/loads/currentload/bin/cli -Apes .show-rd.cli > .tmp-${BASHPID}
        BK_ACT=`grep "Activity Status" .tmp-${BASHPID} | grep "Local Active" | wc -l`
        #echo line ${CF_STS#*: } ${RD_STS#*: } ${IS_ACT} ${MT_ACT} ${BK_ACT}
        sleep 3
      done
      echo "[Info] Reverted back to Primary node successfully.                                               "
    else
      echo "[Error] Both Primary and Backup doesn't seem to be active."
      cat .tmp-${BASHPID}
      exit 1
    fi
  else
    echo "[Error] Something is wrong with the redundancy configuration and/or status:"
    echo
    cat .tmp-${BASHPID} | tail -27
    rm -f .tmp-${BASHPID}
    exit 1
  fi
rm -f .tmp-${BASHPID}
else
  echo "Standalone Deployment Mode detected!"
  exit 1
fi
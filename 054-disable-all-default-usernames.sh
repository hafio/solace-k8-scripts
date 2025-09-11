#!/bin/bash

SELECT_ENV_FILE="000-env.sh"
if [[ -f "`dirname $0`/${SELECT_ENV_FILE}" ]]; then
  source "`dirname $0`/${SELECT_ENV_FILE}"
else
  echo "Environment file '${SELECT_ENV_FILE}' not found"
  exit 1
fi

TMPFILE=.tmp-${BASHPID}

echo 'show message-vpn *' > ${TMPFILE}
${KUBE} cp -n ${SOLBK_NS} ${TMPFILE} ${SOLBK_NAME}-pubsubplus-p-0:/usr/sw/jail/cliscripts/.show-vpn.cli
${KUBE} exec -n ${SOLBK_NS} ${SOLBK_NAME}-pubsubplus-p-0 -- /usr/sw/loads/currentload/bin/cli -Apes .show-vpn.cli > ${TMPFILE}

VPNS=()
while IFS= read -r line; do
  if [[ "${line:0:30}" == "------------------------------" ]]; then
    PARSE_VPN=yes
  elif [[ "${PARSE_VPN}" == "yes" ]] && [[ "${line:0:1}" != "#" ]]; then
    VPNS+=`echo ${line:0:32}`
  fi
done < ${TMPFILE}

echo "home
enable
configure" > ${TMPFILE}
for vpn in "${VPNS[@]}"; do
  echo "client-username default message-vpn \"${vpn}\"
shutdown
exit" >> ${TMPFILE}
echo "show client-username default message-vpn *" >> ${TMPFILE}
done

${KUBE} cp -n ${SOLBK_NS} ${TMPFILE} ${SOLBK_NAME}-pubsubplus-p-0:/usr/sw/jail/cliscripts/.disable-default-usernames.cli
${KUBE} exec -n ${SOLBK_NS} ${SOLBK_NAME}-pubsubplus-p-0 -- /usr/sw/loads/currentload/bin/cli -Apes .disable-default-usernames.cli

rm -f ${TMPFILE}

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
  service smf plain-text shutdown
  service smf ssl shutdown
  service web-transport plain-text shutdown
  service web-transport ssl shutdown
  service rest incoming plain-text shutdown
  service rest incoming ssl shutdown
  service mqtt plain-text shutdown
  service mqtt ssl shutdown
  service mqtt websocket shutdown
  service mqtt websocket-secure shutdown
  service amqp plain-text shutdown
  service amqp ssl shutdown
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
  ${KUBE} cp -n ${SOLBK_NS} ${TMPFILE} ${SOLBK_NAME}-pubsubplus-p-0:/usr/sw/jail/cliscripts/.disable-default-vpn.cli
  ${KUBE} exec -it -n ${SOLBK_NS} ${SOLBK_NAME}-pubsubplus-p-0 -- /usr/sw/loads/currentload/bin/cli -Apes .disable-default-vpn.cli > /dev/null

echo "home
enable
configure
show message-vpn *" > ${TMPFILE}

# create show redundancy cli and copy
${KUBE} cp -n ${SOLBK_NS} ${TMPFILE} ${SOLBK_NAME}-pubsubplus-p-0:/usr/sw/jail/cliscripts/.show-vpn.cli
${KUBE} exec -it -n ${SOLBK_NS} ${SOLBK_NAME}-pubsubplus-p-0 -- /usr/sw/loads/currentload/bin/cli -Apes .show-vpn.cli

${KUBE} exec -it -n ${SOLBK_NS} ${SOLBK_NAME}-pubsubplus-p-0 -- rm -f /usr/sw/jail/cliscripts/.disable-default-vpn.cli /usr/sw/jail/cliscripts/.show-vpn.cli
  
rm -f ${TMPFILE}
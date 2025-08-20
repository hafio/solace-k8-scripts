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
  ${KUBE} cp -n ${SOLBK_NS} ${TMPFILE} ${SOLBK_NAME}-pubsubplus-p-0:/usr/sw/jail/cliscripts/.disable-default-vpn.cli
  ${KUBE} exec -it -n ${SOLBK_NS} ${SOLBK_NAME}-pubsubplus-p-0 -- /usr/sw/loads/currentload/bin/cli -Apes .disable-default-vpn.cli > /dev/null

echo "home
enable
configure

show message-vpn *" > ${TMPFILE}

# create show redundancy cli and copy
${KUBE} cp -n ${SOLBK_NS} ${TMPFILE} ${SOLBK_NAME}-pubsubplus-p-0:/usr/sw/jail/cliscripts/.show-vpn.cli
${KUBE} exec -it -n ${SOLBK_NS} ${SOLBK_NAME}-pubsubplus-p-0 -- /usr/sw/loads/currentload/bin/cli -Apes .show-vpn.cli | head -6
${KUBE} exec -it -n ${SOLBK_NS} ${SOLBK_NAME}-pubsubplus-p-0 -- rm -f /usr/sw/jail/cliscripts/.disable-default-vpn.cli /usr/sw/jail/cliscripts/.show-vpn.cli
  
rm -f ${TMPFILE}
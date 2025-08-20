#!/bin/bash

SELECT_ENV_FILE="000-env.sh"
if [[ -f "`dirname $0`/${SELECT_ENV_FILE}" ]]; then
	source "`dirname $0`/${SELECT_ENV_FILE}"
else 
	echo "Environment file '${SELECT_ENV_FILE}' not found"
	exit 1
fi

REPL_MODES=("" "async" "sync")

echo "[IMPORTANT] This script assumes you have configured Data Replication at the Solace Event Broker Level."
echo -n "Press any key to start "
read -n 1 tmp

echo -n "Enter Message VPN Name: "
read VPN

echo -n "Enter Basic Client-Username: "
read BASICUSER

echo -n "Enter Basic Password: "
read -s BASICPASS

echo 

TMPFILE=.tmp-${BASHPID}

gen_yaml() {
    echo 'home
enable
configure
message-vpn '${VPN}'
  replication
    no-ackp-propagation shutdown
    transaction-replication-mode async
    bridge authentication auth-scheme Basic
    bridge authentication basic client-username '${BASICUSER}' password '${BASICPASS}'
    bridge ssl
    queue reject-msg-to-sender-on-discard'
}

gen_yaml > ${TMPFILE}

# TODO ACTIVE STATE + SHUTDOWN REPLCATION
REPL_TOPIC="<topic>"
while [[ "${#REPL_TOPIC}" -gt 0 ]]; do
    unset REPL_TOPIC
    echo -n "Enter replicated topic (leave blank to exit script): "
    read REPL_TOPIC
    if [[ -n "${REPL_TOPIC}" ]]; then
        echo "    create replicated-topic ${REPL_TOPIC}" >> ${TMPFILE}
        echo -n "[ 1 ] Asynchronous
[ 2 ] Synchronous
Please choose the replication mode of the topic: "
        read REPL_TOPIC_MODE
        echo ${REPL_TOPIC_MODE}
        echo ${REPL_MODES[@]}
        echo "      replication-mode "${REPL_MODES[$REPL_TOPIC_MODE]}  >> ${TMPFILE}
        echo "      exit" >> ${TMPFILE}
    fi
done

# copy files
echo " - Copy Files..."
${KUBE} cp -n ${SOLBK_NS} ${TMPFILE} ${SOLBK_NAME}-pubsubplus-p-0:/usr/sw/jail/cliscripts/.replication.cli

# execute cli
echo " - Running CLI scripts..."
${KUBE} exec -n ${SOLBK_NS} ${SOLBK_NAME}-pubsubplus-${node}-0 -- /usr/sw/loads/currentload/bin/cli -Apes .replication.cli

rm -f ${TMPFILE}
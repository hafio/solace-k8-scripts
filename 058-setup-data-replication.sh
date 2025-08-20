#!/bin/bash

SELECT_ENV_FILE="000-env.sh"
if [[ -f "`dirname $0`/${SELECT_ENV_FILE}" ]]; then
	source "`dirname $0`/${SELECT_ENV_FILE}"
else 
	echo "Environment file '${SELECT_ENV_FILE}' not found"
	exit 1
fi

if [[ -n "${REPL_MATE}" ]] || [[ -n "${REPL_CONN_SSL[@]}" ]]; then
    echo "[Error] Missing Replication Mate virtual-router-name and/or connect-via."
    exit 1
fi

[[ "${REPL_MATE:0:2}" != "v:" ]] && REPL_MATE="v:${REPL_MATE}"

if [[ -z "${REPL_PSK}" ]]; then
    echo "Replication Pre-shared Key (PSK) is empty."
    echo -n "Enter a key or leave blank to generate one: "
    read REPL_PSK
    if [[ -z "${REPL_PSK}" ]]; then
        REPL_PSK=`openssl rand -base64 256 | tr -d \\n`
        echo
        echo "Generated PSK: ${REPL_PSK}"
        echo 
        echo "Please store it for Replication Mate!"
        echo -n "Press any key to continue..."
        read -n 1 tmp
    fi
fi

TMPFILE=.tmp-${BASHPID}

gen_yaml() {
    echo 'home
enable
configure

no replication interface'
for conn in "${REPL_CONN_SSL}"; do
    echo 'replication mate connect-via "'${conn}'" "ssl"'
done
echo 'replication mate virtual-router-name "'${REPL_MATE}'"
replication config-sync bridge shutdown
replication config-sync bridge authentication pre-shared-key key '${REPL_PSK}'
no replication config-sync bridge authentication insecure-upgrade-mode
no replication config-sync bridge shutdown
show replication 
'
}

gen_yaml > ${TMPFILE}

# copy files
echo " - Copy Files..."
${KUBE} cp -n ${SOLBK_NS} ${TMPFILE} ${SOLBK_NAME}-pubsubplus-p-0:/usr/sw/jail/cliscripts/.replication.cli

# execute cli
echo " - Running CLI scripts..."
${KUBE} exec -n ${SOLBK_NS} ${SOLBK_NAME}-pubsubplus-${node}-0 -- /usr/sw/loads/currentload/bin/cli -Apes .replication.cli

rm -f ${TMPFILE}

#!/bin/bash

SELECT_ENV_FILE="000-env.sh"
if [[ -f "`dirname $0`/${SELECT_ENV_FILE}" ]]; then
	source "`dirname $0`/${SELECT_ENV_FILE}"
else 
	echo "Environment file '${SELECT_ENV_FILE}' not found"
	exit 1
fi

ERRMSG=()
[[ -z "${REP_NAME}" ]] && ERRMSG+=('$REP_NAME is empty')
[[ ${#REP_CONN[@]} -eq 0 ]] && ERRMSG+=('$REP_CONN is empty')
[[ -z "${REP_PSK}" ]] && ERRMSG+=('$REP_PSK is empty. Run "openssl rand -base64 256" to generate one.')

if [[ ${#ERRMSG[@]} -ne 0 ]]; then
  for err in "${ERRMSG[@]}"; do
    echo ${err}
  done
  exit 1
fi

echo 'no paging
no paging
enable
configure

no replication interface' > .tmp
for conn in "${REP_CONN[@]}"; do
  echo "replication mate connect-via \"${conn}\" \"ssl\"" >> .tmp
done
echo "replication mate virtual-router-name \"v:${REP_NAME}\"
replication config-sync bridge shutdown
replication config-sync bridge authentication pre-shared-key key ${REP_PSK}
no replication config-sync bridge authentication insecure-upgrade-mode
no replication config-sync bridge shutdown" >> .tmp

${KUBE} cp -n ${SOLBK_NS} .tmp ${SOLBK_NAME}-pubsubplus-p-0:/usr/sw/jail/cliscripts/.setup-rep.cli
rm .tmp
${KUBE} exec -n ${SOLBK_NS} ${SOLBK_NAME}-pubsubplus-p-0 -- /usr/sw/loads/currentload/bin/cli -Apes .setup-rep.cli
#!/bin/bash

SELECT_ENV_FILE="000-env.sh"
if [[ -f "`dirname $0`/${SELECT_ENV_FILE}" ]]; then
	source "`dirname $0`/${SELECT_ENV_FILE}"
else 
	echo "Environment file '${SELECT_ENV_FILE}' not found"
	exit 1
fi

TMPFILE=.tmp-${BASHPID}

echo 'no paging
enable
configure' > ${TMPFILE}

for CA in "${!SOLBK_DOMAINCERT_FILES[@]}"; do
	echo "no ssl domain-certificate-authority ${CA}" >> ${TMPFILE}
done

echo 'home
show domain-certificate-authority ca-name *' >> ${TMPFILE}
cat ${TMPFILE}

${KUBE} cp -n ${SOLBK_NS} ${TMPFILE} ${SOLBK_NAME}-pubsubplus-p-0:/usr/sw/jail/cliscripts/.remove-domain-certs.cli
rm ${TMPFILE}
${KUBE} exec -n ${SOLBK_NS} ${SOLBK_NAME}-pubsubplus-p-0 -- /usr/sw/loads/currentload/bin/cli -Apes .remove-domain-certs.cli

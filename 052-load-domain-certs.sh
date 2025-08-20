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
configure
ssl' > ${TMPFILE}

for CA in "${!SOLBK_DOMAINCERT_FILES[@]}"; do
	echo "create domain-certificate-authority ${CA}
certificate file ${SOLBK_DOMAINCERT_FILES[${CA}]}
exit" >> ${TMPFILE}
  ${KUBE} cp "${SOLBK_DOMAINCERT_FOLDER}/${SOLBK_DOMAINCERT_FILES[${CA}]}" -n ${SOLBK_NS} ${SOLBK_NAME}-pubsubplus-p-0:/usr/sw/jail/certs/. > /dev/null
  [[ $? -ne 0 ]] && echo "[Error] Unable to copy ${SOLBK_DOMAINCERT_FILES[${CA}]}"  
done

echo 'end
show domain-certificate-authority ca-name *' >> ${TMPFILE}

${KUBE} cp -n ${SOLBK_NS} ${TMPFILE} ${SOLBK_NAME}-pubsubplus-p-0:/usr/sw/jail/cliscripts/.load-domain-certs.cli
rm ${TMPFILE}
${KUBE} exec -n ${SOLBK_NS} ${SOLBK_NAME}-pubsubplus-p-0 -- /usr/sw/loads/currentload/bin/cli -Apes .load-domain-certs.cli

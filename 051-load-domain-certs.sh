#!/bin/bash

SELECT_ENV_FILE="000-env.sh"
if [[ -f "`dirname $0`/${SELECT_ENV_FILE}" ]]; then
	source "`dirname $0`/${SELECT_ENV_FILE}"
else 
	echo "Environment file '${SELECT_ENV_FILE}' not found"
	exit 1
fi

echo 'no paging
enable
configure
ssl' > .tmp

for CA in "${!SOLBK_DOMAINCERT_FILES[@]}"; do
	echo "create domain-certificate-authority ${CA}
certificate file ${SOLBK_DOMAINCERT_FILES[${CA}]}
exit" >> .tmp
  ${KUBE} cp "${SOLBK_DOMAINCERT_FOLDER}/${SOLBK_DOMAINCERT_FILES[${CA}]}" -n ${SOLBK_NS} ${SOLBK_NAME}-pubsubplus-p-0:/usr/sw/jail/certs/. > /dev/null
  [[ $? -ne 0 ]] && echo "[Error] Unable to copy ${SOLBK_DOMAINCERT_FILES[${CA}]}"  
done

echo 'end
show domain-certificate-authority ca-name *' >> .tmp

${KUBE} cp -n ${SOLBK_NS} .tmp ${SOLBK_NAME}-pubsubplus-p-0:/usr/sw/jail/cliscripts/.load-domain-certs.cli
rm .tmp
${KUBE} exec -n ${SOLBK_NS} ${SOLBK_NAME}-pubsubplus-p-0 -- /usr/sw/loads/currentload/bin/cli -Apes .load-domain-certs.cli

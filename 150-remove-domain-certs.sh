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
configure' > .tmp

for CA in "${!SOLBK_DOMAINCERT_FILES[@]}"; do
	echo "no ssl domain-certificate-authority ${CA}" >> .tmp
done

echo 'home
show domain-certificate-authority ca-name *' >> .tmp
cat .tmp

${KUBE} cp -n ${SOLBK_NS} .tmp ${SOLBK_NAME}-pubsubplus-p-0:/usr/sw/jail/cliscripts/.remove-domain-certs.cli
rm .tmp
${KUBE} exec -n ${SOLBK_NS} ${SOLBK_NAME}-pubsubplus-p-0 -- /usr/sw/loads/currentload/bin/cli -Apes .remove-domain-certs.cli

#!/bin/bash

SELECT_ENV_FILE="000-env.sh"
if [[ -f "`dirname $0`/${SELECT_ENV_FILE}" ]]; then
	source "`dirname $0`/${SELECT_ENV_FILE}"
else 
	echo "Environment file '${SELECT_ENV_FILE}' not found"
	exit 1
fi

echo "Copying certificates..."
for FILE in "${SOLBK_DOMAINCERT_FILES[@]}"; do
	${KUBE} cp "${SOLBK_DOMAINCERT_FOLDER}/${FILE}" -n ${SOLBK_NS} ${SOLBK_NAME}-pubsubplus-p-0:/usr/sw/jail/certs/.
done

echo "Listing /usr/sw/jail/certs directory in broker..."
${KUBE} exec -n ${SOLBK_NS} ${SOLBK_NAME}-pubsubplus-p-0 -- ls /usr/sw/jail/certs/.
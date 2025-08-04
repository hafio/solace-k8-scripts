#!/bin/bash

SELECT_ENV_FILE="000-env.sh"
if [[ -f "`dirname $0`/${SELECT_ENV_FILE}" ]]; then
	source "`dirname $0`/${SELECT_ENV_FILE}"
else 
	echo "Environment file '${SELECT_ENV_FILE}' not found"
	exit 1
fi

[[ -z "$1" ]] && node=p || node=$1

HEADLINES=16

echo -n "Enter SEMP username: "
read SEMP_USER
echo -n "Enter SEMP password: "
read -s SEMP_PASS
echo 

SEMP_USER=${SEMP_USER:-admin}
SEMP_PASS=${SEMP_PASS:-${SOLBK_ADM_SECRET}}

echo '#!/bin/bash

curl -if -u '${SEMP_USER}:${SEMP_PASS}' -s http://localhost:8080/SEMP/v2/monitor -o .http.out
if [[ $? -eq 0 ]]; then
	echo "Login OK"
else
	echo -n "Login failed. Reason: "
	grep HTTP .http.out
	rm .http.out
	exit 1
fi' > .tmp-${BASHPID}
chmod +x .tmp-${BASHPID}

  ${KUBE} cp -n ${SOLBK_NS} .tmp-${BASHPID} ${SOLBK_NAME}-pubsubplus-${node}-0:/usr/sw/jail/test-semp.sh
if [[ $? -eq 0 ]]; then
	${KUBE} exec -n ${SOLBK_NS} ${SOLBK_NAME}-pubsubplus-${node}-0 -- /usr/sw/jail/test-semp.sh | tee .semp.out-${BASHPID}
	if [[ `grep failed .semp.out | wc -l` -gt 0 ]]; then
		echo -n "Do you want to display the username & password? [y/n] "
		read -n 1 display
		echo
		if [[ "${display,,}" == "y" ]]; then
			echo "Username: ${semp_user}"
			echo "Password: ${semp_pass}"
			echo "cURL Command: curl -i -f -s http://localhost:8080/SEMP/v2/monitor -u ${semp_user}:${semp_pass}"
		fi
	fi
fi

${KUBE} exec -n ${SOLBK_NS} ${SOLBK_NAME}-pubsubplus-${node}-0 -- rm /usr/sw/jail/test-semp.sh
rm -f .tmp-${BASHPID} .semp.out-${BASHPID}
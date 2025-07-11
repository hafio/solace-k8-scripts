#!/bin/bash

SELECT_ENV_FILE="000-env.sh"
if [[ -f "`dirname $0`/${SELECT_ENV_FILE}" ]]; then
	source "`dirname $0`/${SELECT_ENV_FILE}"
else 
	echo "Environment file '${SELECT_ENV_FILE}' not found"
	exit 1
fi

echoUsage() {
	echo "$0 [OPTIONS] <filename(s)>
	
	<filename(s)> : Specify the file(s) to be copied into the broker. Does not support wildcard filenames.
	
	OPTIONS:
		--pod (p|b|m): specifies the Solace pod to copy files into. Default is 'p'."
	exit 1
}

while [[ $# -gt 0 ]]; do
	case "$1" in
		?)
			echoUsage
			;;
		--pod)
			if [[ "$2" =~ ^[pbm]$ ]]; then
				pod="$2"
			else
				echoUsage
			fi
			shift 2
			;;
		*)
			FILE+=("$1")
			shift
			;;
	esac
done

pod=${pod:-p}

if [[ ${#FILE[@]} -eq 0 ]]; then
	echoUsage
fi

for f in "${FILE[@]}"; do
	${KUBE} cp -n ${SOLBK_NS} ${SOLBK_NAME}-pubsubplus-${pod}-0:"${f}" "`basename ${f}`"
	if [[ $? -eq 0 ]]; then
		echo "OK: Copied ${f}"
	else
		echo "Error: ${f} copy failed/error!"
	fi
done

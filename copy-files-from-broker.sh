#!/bin/bash

echoUsage() {
  echo "Usage: $0 [OPTIONS] <filename(s)>
  Copy file(s) FROM a broker pod into the current local directory. Wildcards are not supported.
  OPTIONS:
    --pod (p|b|m) : broker pod to copy files from. Default: p."
}

SELECT_ENV_FILE="000-env.sh"
if [[ -f "$(dirname "$0")/${SELECT_ENV_FILE}" ]]; then
	source "$(dirname "$0")/${SELECT_ENV_FILE}"
else 
	echo "Environment file '${SELECT_ENV_FILE}' not found"
	exit 1
fi

while [[ $# -gt 0 ]]; do
	case "$1" in
		--pod)
			if [[ "$2" =~ ^[pbm]$ ]]; then
				pod="$2"
			else
				echoUsage; exit 1
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
	echoUsage; exit 1
fi

for f in "${FILE[@]}"; do
	${KUBE} cp -n ${SOLBK_NS} ${SOLBK_NAME}-pubsubplus-${pod}-0:"${f}" "`basename ${f}`"
	if [[ $? -eq 0 ]]; then
		echo "OK: Copied ${f}"
	else
		echo "Error: ${f} copy failed/error!"
	fi
done

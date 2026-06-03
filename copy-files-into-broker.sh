#!/bin/bash

echoUsage() {
  echo "Usage: $0 [OPTIONS] <filename(s)>
  Copy file(s) INTO a broker pod. Wildcards are not supported.
  OPTIONS:
    --pod (p|b|m)  : broker pod to copy files into. Default: p.
    --dir <dir>    : target directory inside the pod. Default: '.' (the shell login directory)."
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
		--dir)
			dir="$2"
			shift 2
			;;
		--pod)
			if [[ "$2" =~ ^[pbm]$ ]]; then
				pod="$2"
			else
				echoUsage; exit 1
			fi
			shift 2
			;;
		*)
			if [[ -f "$1" ]]; then
				FILE+=("$1")
				shift
			else
				echo "Error: $1 not found."
				exit 1
			fi
			;;
	esac
done

pod=${pod:-p}
dir=${dir:-.}

if [[ ${#FILE[@]} -eq 0 ]]; then
	echoUsage; exit 1
fi

for f in "${FILE[@]}"; do
	${KUBE} cp "${f}" -n ${SOLBK_NS} ${SOLBK_NAME}-pubsubplus-${pod}-0:${dir}
	if [[ $? -eq 0 ]]; then
		echo "OK: Copied ${f}"
	else
		echo "Error: ${f} copy failed/error!"
	fi
done

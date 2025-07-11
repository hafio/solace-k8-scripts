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
		--pod (p|b|m): specifies the Solace pod to copy files into. Default is 'p'.
		--dir directory: specifies the target directory for files to be copied into. Default is '.' (default directory when logging into shell)"
	exit 1
}

while [[ $# -gt 0 ]]; do
	case "$1" in
		?)
			echoUsage
			;;
		--dir)
			dir="$2"
			shift 2
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
	echoUsage
fi

for f in "${FILE[@]}"; do
	${KUBE} cp "${f}" -n ${SOLBK_NS} ${SOLBK_NAME}-pubsubplus-${pod}-0:${dir}
	if [[ $? -eq 0 ]]; then
		echo "OK: Copied ${f}"
	else
		echo "Error: ${f} copy failed/error!"
	fi
done

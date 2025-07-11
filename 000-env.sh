#!/bin/bash

while [[ $# -gt 0 ]]; do
	case "$1" in
		--env)
			ENV_FILE="$2"
			shift 2
			;;
		*)
			PARAMS="${PARAMS} $1"
			shift
			;;
	esac
done

EXDIR=`dirname $0`
ENV_FILE=${ENV_FILE:-default}
EXDIR=`dirname $0`
if [[ -f "${EXDIR}/env/${ENV_FILE}" ]]; then

  source "${EXDIR}/env/${ENV_FILE}"
    
  ######################################################################
  # DERIVED VALUES                                                     #
  ######################################################################
  
  SOLOP_DERIVED_NS=`${KUBE} get deployment --all-namespaces -o custom-columns=NS:.metadata.namespace,NAME:.metadata.name | grep pubsubplus-eventbroker-operator | cut -d" " -f1`
  [[ -n "${SOLOP_DERIVED_NS}" ]] && SOLOP_DERIVED_POD=`${KUBE} get pod -n ${SOLOP_DERIVED_NS} -o custom-columns=NAME:.metadata.name --no-headers`
  
  ######################################################################
  # DEFAULT VALUES                                                     #
  # DO NOT CHANGE THE VARIABLES AND VALUES BELOW HERE!                 #
  # ANY VARIABLES THAT CAN BE BLANK SHOULD NOT BE MENTIONED BELOW.     #
  ######################################################################

  KUBE=${KUBE:-kubectl}

  SOLOP_IMAGE=${SOLOP_IMAGE:-pubsubplus-eventbroker-operator:1.2.0}
  SOLOP_WATCH_NS=${SOLOP_WATCH_NS:-solace-namespace}

  SOLOP_CPU=${SOLOP_CPU:-500m}
  SOLOP_MEM=${SOLOP_MEM:-512Mi}

  SOLBK_REDUNDANCY=${SOLBK_REDUNDANCY:-false}

  SOLBK_ADM_PASS=${SOLBK_ADM_PASS:-adm1nPA@55w0rD}
  SOLBK_ADM_SECRET=${SOLBK_ADM_SECRET:-solace-admin-secret}

  if [[ -n "${SOLBK_SVR_SECRET}" ]]; then
    SOLBK_TLS_CERT=${SOLBK_TLS_CERT:-certs/tls.crt}
    SOLBK_TLS_CERTKEY=${SOLBK_TLS_CERTKEY:-certs/tls.key}
  fi

  SOLBK_SCALING_MAXCONN=${SOLBK_SCALING_MAXCONN:-100}
  SOLBK_SCALING_MAXPOOL=${SOLBK_SCALING_MAXPOOL:-10000}
  SOLBK_SCALING_MAXQMSG=${SOLBK_SCALING_MAXQMSG:-100}

  SOLBK_MSGNODE_CPU=${SOLBK_MSGNODE_CPU:-2}
  SOLBK_MSGNODE_MEM=${SOLBK_MSGNODE_MEM:-3410Mi}

  SOLBK_STORAGE_MONNODE=${SOLBK_STORAGE_MONNODE:-5Gi}

  if [[ -n "${SOLBK_ANTIAFFINITY_NS[@]}" ]]; then
    SOLBK_ANTIAFFINITY_WT=${SOLBK_ANTIAFFINITY_WT:-100}
  fi

  SOLBK_DIAG_DIR=${SOLBK_DIAG_DIR:-diag-configs}
  SOLBK_CLISCRIPTS_FOLDER=${SOLBK_CLISCRIPTS_FOLDER:-cli}

  ######################################################################
  # MANDATORY VALUES CHECK                                             #
  ######################################################################

  PARM=("SOLBK_NAME" "SOLBK_NS" "SOLBK_IMAGE" "SOLBK_IMG_TAG" "SOLBK_STORAGE_MSGNODE")
  EMPTY_VAR=()

  for var in "${PARM[@]}"; do
    [[ -z "${!var}" ]] && EMPTY_VAR+=("${var}")
  done
  if [[ ${#EMPTY_VAR[@]} -gt 0 ]]; then
    echo "These variables cannot be empty: ${EMPTY_VAR[@]}"
    exit 1
  fi
  
else
	echo "Error: ${ENV_FILE} not found in '${EXDIR}/env' folder.

Usage: $0 --env <file>
	where <file> is a shell script that is used to capture the necessary variables used in deployment inside the 'env' directory.
	Default value is 'default'"
	exit 1
fi
set -- ${PARAMS}

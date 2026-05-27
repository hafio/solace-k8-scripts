#!/bin/bash

# Normalize a pod-role argument to its single-letter form (p|b|m).
# Accepts: p|primary, b|backup, m|monitor. Empty defaults to p.
# Usage: pod=$(pick_pod "$1") || exit 1
pick_pod() {
  case "$1" in
    p|primary) echo p ;;
    b|backup)  echo b ;;
    m|monitor) echo m ;;
    "")        echo p ;;
    *)
      echo "[Error] Invalid node: $1 (expected p|b|m or primary|backup|monitor)" >&2
      return 1
      ;;
  esac
}

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

EXDIR=$(dirname "$0")
ENV_FILE=${ENV_FILE:-default}
if [[ -f "${EXDIR}/env/${ENV_FILE}" ]]; then

  source "${EXDIR}/env/${ENV_FILE}"
    
  ######################################################################
  # DERIVED VALUES                                                     #
  ######################################################################
  
  ######################################################################
  # DEFAULT VALUES                                                     #
  # DO NOT CHANGE THE VARIABLES AND VALUES BELOW HERE!                 #
  # ANY VARIABLES THAT CAN BE BLANK SHOULD NOT BE MENTIONED BELOW.     #
  ######################################################################

  KUBE=${KUBE:-kubectl}

  SOLOP_IMAGE=${SOLOP_IMAGE:-docker.io/solace/pubsubplus-eventbroker-operator:1.4.0}
  
  if [[ -n "${SOLOP_NS}" ]]; then
    SOLOP_DERIVED_NS=${SOLOP_NS}
  else
    SOLOP_DERIVED_NS=$(${KUBE} get deployment --all-namespaces -o custom-columns=NS:.metadata.namespace,NAME:.metadata.name 2> /dev/null | grep pubsubplus-eventbroker-operator | cut -d" " -f1)
  fi
  
  [[ -n "${SOLOP_DERIVED_NS}" ]] && SOLOP_DERIVED_POD=$(${KUBE} get pod -n "${SOLOP_DERIVED_NS}" -o custom-columns=NAME:.metadata.name --no-headers)
  
  # IF OPERATOR IS NOT RUNNING AND ${SOLOP_NS} IS NOT SPECIFIED, DEFAULT TO 'pubsubplus-operator-system'
  # THIS LINE NEEDS TO BE AFTER SOLOP_DERIVED_POD SO THAT POD NAME WILL NOT BE RETRIEVED IF OPERATOR IS NOT RUNNING
  SOLOP_DERIVED_NS=${SOLOP_DERIVED_NS:-pubsubplus-operator-system}
  
  SOLOP_WATCH_SOLBK_NS=${SOLOP_WATCH_SOLBK_NS:-true}
  if [[ "${SOLOP_WATCH_SOLBK_NS}" == "true" ]]; then
    [[ -n "${SOLOP_WATCH_NS}" ]] && SOLOP_WATCH_NS+=","
    SOLOP_WATCH_NS+=${SOLBK_NS}
  fi 

  SOLOP_CPU=${SOLOP_CPU:-500m}
  SOLOP_MEM=${SOLOP_MEM:-512Mi}

  SOLBK_REDUNDANCY=${SOLBK_REDUNDANCY:-false}

  SOLBK_ADM_PASS=${SOLBK_ADM_PASS:-adm1nPA@55w0rD}
  SOLBK_USR_SECRET=${SOLBK_USR_SECRET:-solace-admin-secret}

  if [[ -n "${SOLBK_SVR_SECRET}" ]]; then
    SOLBK_TLS_CERT=${SOLBK_TLS_CERT:-certs/tls.crt}
    SOLBK_TLS_CERTKEY=${SOLBK_TLS_CERTKEY:-certs/tls.key}
  fi

  SOLBK_SCALING_MAXCONN=${SOLBK_SCALING_MAXCONN:-100}
  SOLBK_SCALING_MAXPOOL=${SOLBK_SCALING_MAXPOOL:-10000}
  SOLBK_SCALING_MAXQMSG=${SOLBK_SCALING_MAXQMSG:-100}
  SOLBK_SCALING_MAXKAFKABRIDGE=${SOLBK_SCALING_MAXKAFKABRIDGE:-0}
  SOLBK_SCALING_MAXKAFKACONN=${SOLBK_SCALING_MAXKAFKACONN:-0}
  SOLBK_SCALING_MAXBRIDGE=${SOLBK_SCALING_MAXBRIDGE:-25}
  SOLBK_SCALING_MAXSUB=${SOLBK_SCALING_MAXSUB:-50000}
  SOLBK_SCALING_MAXGMSSIZE=${SOLBK_SCALING_MAXGMSSIZE:-10}

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

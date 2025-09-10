#!/bin/bash

SELECT_ENV_FILE="000-env.sh"
if [[ -f "`dirname $0`/${SELECT_ENV_FILE}" ]]; then
  source "`dirname $0`/${SELECT_ENV_FILE}"
else 
  echo "Environment file '${SELECT_ENV_FILE}' not found"
  exit 1
fi

echo
echo "IMPORTANT: This script only checks if value exists. It does not verify if the values are correct."
echo "Press any key to start..."
read -s 
echo
echo "##### Kubernetes #####"
[[ -n ${KUBE} ]] && echo "[ OK  ] kubectl command: ${KUBE}" || echo "[ERROR] empty kubectl comand"

echo
echo "##### IMAGE REPOSITORY #####"
[[ -n "${IMAGEREPO_HOST}" ]] && echo "[ OK  ] Image Repository URL: ${IMAGEREPO_HOST}" || echo "[ INFO] Images are loaded locally or via default URLs"
if [[ -n "${IMAGEREPO_SECRET}" ]]; then
  echo "[ INFO] Image Repository requires credential"
  [[ -n "${IMAGEREPO_USER}" ]] && echo " - [ OK  ] Image repo username specified" || echo "[ INFO] Empty image repo username"
  [[ -n "${IMAGEREPO_PASS}" ]] && echo " - [ OK  ] Image repo password specified" || echo "[ INFO] Empty image repo password"
else
  echo "[ INFO] Image repository does not require any login"
fi

echo
echo "##### SOLACE OPERATOR #####"
[[ -n "${SOLOP_IMAGE}" ]] && echo "[ INFO] Solace Operator Image Name: ${SOLOP_IMAGE}" || echo "[ERROR] Empty Solace Operator image"
echo "[ INFO] Solace Operator Namespace: ${SOLOP_DERIVED_NS}"
[[ -n "${SOLOP_WATCH_NS}" ]] && echo "[ INFO] Namespaces watched by Operator: ${SOLOP_WATCH_NS}" || echo "[ INFO] All namespaces are watched by Operator"
[[ -n "${SOLOP_CPU}" ]] && echo "[ INFO] Operator CPU: ${SOLOP_CPU}" || echo "[ERROR] Operator CPU not specified"
[[ -n "${SOLOP_MEM}" ]] && echo "[ INFO] Operator Memory: ${SOLOP_MEM}" || echo "[ERROR] Operator Memory not specified"

echo
echo "##### SOLACE BROKER #####"
[[ -n "${SOLBK_NAME}" ]] && echo "[ INFO] Solace Broker Name: ${SOLBK_NAME}" || echo "[ERROR] Solace Broker Name not specified"
[[ -n "${SOLBK_NS}" ]] && echo "[ INFO] Solace Broker Namespace: ${SOLBK_NS}" || echo "[ERROR] Solace Broker Namespace not specified"
[[ -n "${SOLBK_IMAGE}" ]] && echo "[ INFO] Solace Broker Image: ${SOLBK_IMAGE}" || echo "[ERROR] Solace Broker Image not specified"
[[ -n "${SOLBK_IMG_TAG}" ]] && echo "[ INFO] Solace Broker Image Tag: ${SOLBK_IMG_TAG}" || echo "[ERROR] Solace Broker Image Tag not specified"
if [[ -n "${SOLBK_REDUNDANCY}" ]]; then
  if [[ "${SOLBK_REDUNDANCY}" == "true" ]]; then
    echo "[ INFO] High Availability Deployment"
  elif [[ "${SOLBK_REDUNDANCY}" == "false" ]]; then
    echo "[ INFO] Standalone Deployment"
  else
    echo "[ERROR] Unknown deployment redundancy"
  fi
else
  echo "[ INFO] Standalone Deployment"
fi

echo "[ INFO] Broker Max Connections: ${SOLBK_SCALING_MAXCONN:-100 (default)}"
echo "[ INFO] Broker Max Queue Messages (mil): ${SOLBK_SCALING_MAXQMSG:-100 (default)}"
echo "[ INFO] Broker Max Spool Usage (MB): ${SOLBK_SCALING_MAXPOOL:-1000 (default)}"
echo "[ INFO] Broker CPU (Msging Node): ${SOLBK_MSGNODE_CPU:-2 (default)}"
echo "[ INFO] Broker Memory (Msging Node): ${SOLBK_MSGNODE_MEM:-4025Mi (default)}"

[[ -n "${SOLBK_STORAGECLASS}" ]] && echo "[ INFO] Storage Class: ${SOLBK_STORAGECLASS}"
echo "[ INFO] Broker Storage (Msging Node): ${SOLBK_STORAGE_MSGNODE:-30Gi (default)}"
echo "[ INFO] Broker Storage (Monitor Node): ${SOLBK_STORAGE_MONNODE:-3Gi (default)}"


[[ -n "${SOLBK_ADM_PASS}" ]] && echo "[ OK  ] Solace Broker Admin Password specfied" || echo "[ERROR] No Solace Broer Admin Password"
[[ -n "${SOLBK_ADM_SECRET}" ]] && echo "[ OK  ] Solace Broker Admin Secret Name: ${SOLBK_ADM_SECRET}" || echo "[ERROR] Solace Broker Admin Secret Name not specified"

if [[ -n "${SOLBK_SVR_SECRET}" ]]; then
  echo "[ OK  ] SSL Server certificates:"
  [[ -n "${SOLBK_TLS_CERT}" ]] && echo " - [ INFO] Server certificate file: ${SOLBK_TLS_CERT}" || echo " - [ERROR] No server certificates specified"
  [[ -n "${SOLBK_TLS_CERTKEY}" ]] && echo " - [ INFO] Server certificate key file: ${SOLBK_TLS_CERTKEY}" || echo " - [ERROR] No server certificate key specified"
else
  echo "[ WARN] No SSL server certificates configured"
fi

[[ -n "${SOLBK_LOADBALANCER_IP}" ]] && echo "[ INFO] Solace Load Balancer External IP: ${SOLBK_LOADBALANCER_IP}"

[[ -n "${SOLBK_ANTIAFFINITY_NS[@]}" ]] && echo "[ INFO] Anti-affinity Namespaces: ${SOLBK_ANTIAFFINITY_NS[@]}" || echo "[ WARN] Anti-affinity namespace labels should be specified"

[[ -n "${SOLBK_DIAG_DIR}" ]] && echo "[ INFO] Gather Diagnostics and Configs Directory: ${SOLBK_DIAG_DIR}" || echo "[ WARN] Empty Gather Diagnostics and Configs Directory"
#!/bin/bash

echoUsage() {
  echo "Usage: $0 [OPTIONS]
  Create the broker secrets: admin/monitor credentials, optional TLS server certificate, and
  optional image-pull secret.
  OPTIONS:
    --only-gen-yaml : print the generated manifest to stdout instead of applying it"
}

SELECT_ENV_FILE="000-env.sh"
if [[ -f "$(dirname "$0")/${SELECT_ENV_FILE}" ]]; then
	source "$(dirname "$0")/${SELECT_ENV_FILE}"
else 
	echo "Environment file '${SELECT_ENV_FILE}' not found"
	exit 1
fi

[[ -n "${SOLBK_SVR_SECRET}" ]] && [[ ! -f "${SOLBK_TLS_CERT}" ]] && errmsg+="\n\t${SOLBK_TLS_CERT} not found"
[[ -n "${SOLBK_SVR_SECRET}" ]] && [[ ! -f "${SOLBK_TLS_CERTKEY}" ]] && errmsg+="\n\t${SOLBK_TLS_CERTKEY} not found"
if [[ -n "${errmsg}" ]]; then
	echo -e "Error:${errmsg}"
	exit 1
fi

CLI_USR_PASS=(--from-literal="username_admin_password=${SOLBK_ADM_PASS}")
[[ -n "${SOLBK_MON_PASS}" ]] && CLI_USR_PASS+=(--from-literal="username_monitor_password=${SOLBK_MON_PASS}")
for up in "${SOLBK_USR_PASS[@]}"; do
	user="${up%%=*}"
	pass="${up#*=}"
	CLI_USR_PASS+=(--from-literal="username_${user}_password=${pass}")
done
# Emit each applicable secret manifest separated by '---'. Aborts (return 1) if any
# secret fails to generate, so a partial manifest never gets silently applied.
gen_yaml() {
  ${KUBE} create secret generic "${SOLBK_USR_SECRET}" -n "${SOLBK_NS}" "${CLI_USR_PASS[@]}" --dry-run=client -o yaml || return 1
  if [[ -n "${SOLBK_SVR_SECRET}" ]]; then
    echo "---"
    ${KUBE} create secret tls ${SOLBK_SVR_SECRET} -n ${SOLBK_NS} --cert <(cat ${SOLBK_TLS_CERT} ${SOLBK_TLS_CERTCAS[@]}) --key=${SOLBK_TLS_CERTKEY} --dry-run=client -o yaml || return 1
  fi
  if [[ -n "${IMAGEREPO_SECRET}" ]]; then
    echo "---"
    ${KUBE} create secret docker-registry ${IMAGEREPO_SECRET} -n ${SOLBK_NS} --docker-server="${IMAGEREPO_HOST}" --docker-username="${IMAGEREPO_USER}" --docker-password="${IMAGEREPO_PASS}" --dry-run=client -o yaml || return 1
  fi
}

if [[ "${GENONLY}" == "true" ]]; then
  gen_yaml
  exit $?
fi

# Generate first so a generation failure aborts before any apply (the pipe below would
# otherwise mask gen_yaml's exit status behind kubectl apply's).
manifest=$(gen_yaml) || { echo "Error generating secrets."; exit 1; }
echo "${manifest}" | ${KUBE} apply -f -
if [[ $? -ne 0 ]]; then
  echo "Error creating secrets."
  exit 1
fi
echo "Secrets created successfully."

${KUBE} get secret -n ${SOLBK_NS}

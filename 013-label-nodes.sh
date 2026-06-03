#!/bin/bash

echoUsage() {
  echo "Usage: $0
  Interactively label cluster nodes with the custom labels from the SOLBK_NODELABEL_PRI/BKP/MON
  arrays (built-in kubernetes.io labels are skipped). Prompts for a node per role; takes no positional arguments."
}

SELECT_ENV_FILE="000-env.sh"
if [[ -f "$(dirname "$0")/${SELECT_ENV_FILE}" ]]; then
	source "$(dirname "$0")/${SELECT_ENV_FILE}"
else
	echo "Environment file '${SELECT_ENV_FILE}' not found"
	exit 1
fi

# ── RBAC check ──
echo "Checking cluster permissions to label nodes..."
if ! ${KUBE} auth can-i update nodes > /dev/null 2>&1; then
  echo "[ERROR] Current user does not have permission to label nodes."
  echo "        Request 'update' access on 'nodes' resources from your cluster administrator."
  exit 1
fi
echo "[ OK  ] Permissions verified."
echo

# ── Helper: filter custom labels from a NODELABEL array ──
# Built-in K8s label prefixes to exclude
K8S_PREFIXES=("kubernetes.io/" "k8s.io/" "node.kubernetes.io/" "beta.kubernetes.io/")

is_builtin_label() {
  local key="$1"
  for prefix in "${K8S_PREFIXES[@]}"; do
    if [[ "${key}" == ${prefix}* ]]; then
      return 0
    fi
  done
  return 1
}

# Extract custom labels from an array, returns results in CUSTOM_LABELS array
extract_custom_labels() {
  local -n arr=$1
  CUSTOM_LABELS=()
  for entry in "${arr[@]}"; do
    # Format is "key: value" — split on first ": "
    local key="${entry%%:*}"
    key="${key## }"  # trim leading space
    key="${key%% }"  # trim trailing space
    if ! is_builtin_label "${key}"; then
      CUSTOM_LABELS+=("${entry}")
    fi
  done
}

# ── Determine which roles to process ──
ROLES=("PRI")
ROLE_NAMES=("Primary")
if [[ "${SOLBK_REDUNDANCY}" == "true" ]]; then
  ROLES=("PRI" "BKP" "MON")
  ROLE_NAMES=("Primary" "Backup" "Monitor")
fi

# ── Scan all roles for custom labels ──
HAS_CUSTOM=false
for role in "${ROLES[@]}"; do
  varname="SOLBK_NODELABEL_${role}"
  # use indirect reference to check array
  eval 'arr_check=("${'"${varname}"'[@]}")'
  if [[ ${#arr_check[@]} -gt 0 ]]; then
    extract_custom_labels arr_check
    if [[ ${#CUSTOM_LABELS[@]} -gt 0 ]]; then
      HAS_CUSTOM=true
      break
    fi
  fi
done

if [[ "${HAS_CUSTOM}" != "true" ]]; then
  echo "No custom labels detected in SOLBK_NODELABEL arrays."
  echo "Only built-in Kubernetes labels found (e.g. kubernetes.io/hostname)."
  exit 0
fi

# ── Fetch node list once ──
NODE_LIST=$(${KUBE} get nodes -o custom-columns=NAME:.metadata.name --no-headers 2>/dev/null)
if [[ -z "${NODE_LIST}" ]]; then
  echo "[ERROR] Unable to retrieve cluster nodes. Check your kubeconfig and connectivity."
  exit 1
fi
mapfile -t NODES <<< "${NODE_LIST}"

show_node_menu() {
  echo "  Available nodes:"
  for i in "${!NODES[@]}"; do
    printf "    [%d] %s\n" $((i+1)) "${NODES[$i]}"
  done
}

# ── Process each role ──
for idx in "${!ROLES[@]}"; do
  role="${ROLES[$idx]}"
  role_name="${ROLE_NAMES[$idx]}"
  varname="SOLBK_NODELABEL_${role}"

  eval 'role_arr=("${'"${varname}"'[@]}")'
  if [[ ${#role_arr[@]} -eq 0 ]]; then
    continue
  fi

  extract_custom_labels role_arr
  if [[ ${#CUSTOM_LABELS[@]} -eq 0 ]]; then
    echo "── ${role_name} node: no custom labels to apply"
    echo
    continue
  fi

  echo "── ${role_name} node ──"
  echo "  Custom labels to apply:"
  for lbl in "${CUSTOM_LABELS[@]}"; do
    echo "    ${lbl}"
  done
  echo

  show_node_menu
  echo
  SELECTION=""
  while [[ -z "${SELECTION}" ]]; do
    echo -n "  Select node for ${role_name} [1-${#NODES[@]}]: "
    read -r choice
    if [[ "${choice}" =~ ^[0-9]+$ ]] && [[ ${choice} -ge 1 ]] && [[ ${choice} -le ${#NODES[@]} ]]; then
      SELECTION="${NODES[$((choice-1))]}"
    else
      echo "  Invalid selection. Try again."
    fi
  done

  echo "  Applying labels to node '${SELECTION}'..."
  for entry in "${CUSTOM_LABELS[@]}"; do
    # Convert "key: value" to "key=value"
    local_key="${entry%%:*}"
    local_val="${entry#*: }"
    local_key="${local_key## }"
    local_key="${local_key%% }"
    local_val="${local_val## }"
    local_val="${local_val%% }"

    ${KUBE} label node "${SELECTION}" "${local_key}=${local_val}" --overwrite
    if [[ $? -eq 0 ]]; then
      echo "  [ OK  ] ${local_key}=${local_val}"
    else
      echo "  [ERROR] Failed to apply ${local_key}=${local_val}"
    fi
  done
  echo
done

echo "Done."

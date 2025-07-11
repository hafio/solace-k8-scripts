#!/bin/bash

SELECT_ENV_FILE="000-env.sh"
if [[ -f "`dirname $0`/${SELECT_ENV_FILE}" ]]; then
	source "`dirname $0`/${SELECT_ENV_FILE}"
else 
	echo "Environment file '${SELECT_ENV_FILE}' not found"
	exit 1
fi

# ========== Timestamp for AGE calculation ==========
now=$(date +%s)

# ========== Pods ==========
echo -e "\n### PODS ###"

declare -A stores=()
while read -r pvc; do
  name=$(echo "$pvc" | jq -r '.metadata.name')
  ns=$(echo "$pvc" | jq -r '.metadata.namespace')
  size=$(echo "$pvc" | jq -r '.status.capacity.storage')
  stores+=([${ns}---${name}]="${size}")
done < <(${KUBE} get pvc --all-namespaces -o json | jq -c '.items[] | select(.metadata.name | contains("-pubsubplus-"))')

while read -r pod; do
    name=$(echo "$pod" | jq -r '.metadata.name')
    ns=$(echo "$pod" | jq -r '.metadata.namespace')
    status=$(echo "$pod" | jq -r '.status.phase')
    ip=$(echo "$pod" | jq -r '.status.podIP // "-"')
    node=$(echo "$pod" | jq -r '.spec.nodeName // "-"')
    ready_total=$(echo "$pod" | jq '.status.containerStatuses | length')
    ready_count=$(echo "$pod" | jq '[.status.containerStatuses[] | select(.ready == true)] | length')
    restarts=$(echo "$pod" | jq '[.status.containerStatuses[]?.restartCount] | add // 0')
    pvc=$(echo "$pod" | jq -r '.spec.volumes[0]."persistentVolumeClaim"."claimName"')
    cpu=$(echo "$pod" | jq -r '.spec.containers[0].resources.limits.cpu')
    mem=$(echo "$pod" | jq -r '.spec.containers[0].resources.limits.memory')

    created=$(echo "$pod" | jq -r '.metadata.creationTimestamp')
    created_sec=$(date -d "$created" +%s 2>/dev/null || gdate -d "$created" +%s)
    age_sec=$((now - created_sec))
    if (( age_sec >= 86400 )); then
        age="$((age_sec / 86400))d"
    elif (( age_sec >= 3600 )); then
        age="$((age_sec / 3600))h"
    elif (( age_sec >= 60 )); then
        age="$((age_sec / 60))m"
    else
        age="${age_sec}s"
    fi
    
    OUT_LMT+=$(printf "\n%-30s %-20s %-5s %-10s %-10s" "${name}" "${ns}" "${cpu}" "${mem}" "${stores[${ns}---${pvc}]}")
    OUT_STS+=$(printf "\n%-30s %-20s %d/%d    %-10s %-9d %-6s %-15s %-20s" "${name}" "${ns}" "${ready_count}" "${ready_total}" "${status}" "${restarts}" "${age}" "${ip}" "${node}")
done < <(${KUBE} get pods --all-namespaces -o json | jq -c '.items[] | select(.metadata.name | contains("-pubsubplus-"))')

printf "\n%-30s %-20s %-5s %-10s %-10s" "NAME" "NAMESPACE" "CPU" "MEMORY" "DISK"
echo -e "${OUT_LMT}"

printf "\n%-30s %-20s %-6s %-10s %-9s %-6s %-15s %-20s" "NAME" "NAMESPACE" "READY" "STATUS" "RESTARTS" "AGE" "IP" "NODE"
echo -e "${OUT_STS}"

# ========== Services ==========
echo -e "\n### SERVICES ###"
printf "%-30s %-20s %-15s %-20s %-30s\n" \
  "NAME" "NAMESPACE" "TYPE" "CLUSTER-IP" "EXTERNAL-IP"

${KUBE} get svc --all-namespaces -o json | jq -c '.items[] | select(.metadata.name | contains("pubsubplus"))' | while read -r svc; do
    name=$(echo "$svc" | jq -r '.metadata.name')
    ns=$(echo "$svc" | jq -r '.metadata.namespace')
    type=$(echo "$svc" | jq -r '.spec.type')
    cluster_ip=$(echo "$svc" | jq -r '.spec.clusterIP // "-"')
    external_ip=$(echo "$svc" | jq -r '.status.loadBalancer.ingress[0].ip // .status.loadBalancer.ingress[0].hostname // "-"')

    printf "%-30s %-20s %-15s %-20s %-30s\n" \
        "$name" "$ns" "$type" "$cluster_ip" "$external_ip"
done

# ========== StatefulSets ==========
echo -e "\n### STATEFULSETS ###"
printf "%-30s %-20s %-7s %-10s %-5s %-6s\n" \
  "NAME" "NAMESPACE" "READY" "REPLICAS" "AGE" "SERVICE"

${KUBE} get statefulsets --all-namespaces -o json | jq -c '.items[] | select(.metadata.name | contains("-pubsubplus-"))' | while read -r sts; do
    name=$(echo "$sts" | jq -r '.metadata.name')
    ns=$(echo "$sts" | jq -r '.metadata.namespace')
    replicas=$(echo "$sts" | jq -r '.spec.replicas')
    ready_replicas=$(echo "$sts" | jq -r '.status.readyReplicas // 0')
    svc_name=$(echo "$sts" | jq -r '.spec.serviceName // "-"')

    created=$(echo "$sts" | jq -r '.metadata.creationTimestamp')
    created_sec=$(date -d "$created" +%s 2>/dev/null || gdate -d "$created" +%s)
    age_sec=$((now - created_sec))
    if (( age_sec >= 86400 )); then
        age="$((age_sec / 86400))d"
    elif (( age_sec >= 3600 )); then
        age="$((age_sec / 3600))h"
    elif (( age_sec >= 60 )); then
        age="$((age_sec / 60))m"
    else
        age="${age_sec}s"
    fi

    printf "%-30s %-20s %-7s %-10s %-5s %-6s\n" \
        "$name" "$ns" "$ready_replicas" "$replicas" "$age" "$svc_name"
done

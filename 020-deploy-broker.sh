#!/bin/bash

SELECT_ENV_FILE="000-env.sh"
if [[ -f "`dirname $0`/${SELECT_ENV_FILE}" ]]; then
  source "`dirname $0`/${SELECT_ENV_FILE}"
else 
  echo "Environment file '${SELECT_ENV_FILE}' not found"
  exit 1
fi

if [[ "${1:0:1}" == "?" ]] || [[ "${1:0:2}" == "-h" ]] || [[ "${1:0:3}" == "--h" ]]; then
	echo "Usage: $0 [OPTIONS]"
	echo "	OPTIONS:"
	echo "    --only-gen-yaml: generates the yaml file but does not deploy"
  echo "    --keep-yaml: does not remove the generated yaml file '.broker.yaml'"
	exit
fi

gen_yaml() {
  echo "apiVersion: pubsubplus.solace.com/v1beta1
kind: PubSubPlusEventBroker
metadata:
  namespace: ${SOLBK_NS:-solace-namespace}
  name: ${SOLBK_NAME:-solace-event-broker}
spec:
  image:
    repository: "`[[ -n "${IMAGEREPO_HOST}" ]] && echo "${IMAGEREPO_HOST}/"`"${SOLBK_IMAGE:-solace-pubsub-standard}
    tag: ${SOLBK_IMG_TAG:-latest}
    pullPolicy: IfNotPresent"
    
  [[ -n ${IMAGEREPO_SECRET} ]] && echo "    pullSecrets:
    - name: ${IMAGEREPO_SECRET}"
    
  echo "  adminCredentialsSecret: ${SOLBK_ADM_SECRET:-adminpassword123}
  redundancy: ${SOLBK_REDUNDANCY}"

  echo "  podDisruptionBudgetForHA: true
  systemScaling:
    maxConnections: ${SOLBK_SCALING_MAXCONN}
    maxQueueMessages: ${SOLBK_SCALING_MAXQMSG}
    maxSpoolUsage: ${SOLBK_SCALING_MAXPOOL}
    messagingNodeCpu: \"${SOLBK_MSGNODE_CPU}\"
    messagingNodeMemory: ${SOLBK_MSGNODE_MEM}
  storage:"
  [[ -n "${SOLBK_STORAGECLASS}" ]] && echo "    useStorageClass: ${SOLBK_STORAGECLASS}"
  echo "    messagingNodeStorageSize: ${SOLBK_STORAGE_MSGNODE}
    monitorNodeStorageSize: ${SOLBK_STORAGE_MONNODE}
  timezone: \"Asia/Singapore\""
    [[ -n "${SOLBK_SVR_SECRET}" ]] && echo "  tls:
    serverTlsConfigSecret: ${SOLBK_SVR_SECRET}
    enabled: true
    certFilename: tls.crt
    certKeyFilename: tls.key"
    
    if [[ -n "${SOLBK_ANTIAFFINITY_NS[@]}" ]] || [[ -n "${SOLBK_NODELABEL_PRI[@]}${SOLBK_NODELABEL_BKP[@]}${SOLBK_NODELABEL_MON[@]}" ]]; then
      if [[ -n "${SOLBK_NODELABEL_PRI[@]}" ]]; then
        POD_LABEL_PRI="      nodeSelector:"
        for NL in "${!SOLBK_NODELABEL_PRI[@]}"; do
          POD_LABEL_PRI+="\n        ${NL}: ${SOLBK_NODELABEL_PRI[${NL}]}"
        done
        POD_LABEL_PRI+="\n"
      fi
      if [[ -n "${SOLBK_NODELABEL_BKP[@]}" ]]; then
        POD_LABEL_BKP="      nodeSelector:"
        for NL in "${!SOLBK_NODELABEL_BKP[@]}"; do
          POD_LABEL_BKP+="\n        ${NL}: ${SOLBK_NODELABEL_BKP[${NL}]}"
        done
        POD_LABEL_BKP+="\n"
      fi
      if [[ -n "${SOLBK_NODELABEL_MON[@]}" ]]; then
        POD_LABEL_MON="      nodeSelector:"
        for NL in "${!SOLBK_NODELABEL_MON[@]}"; do
          POD_LABEL_MON+="\n        ${NL}: ${SOLBK_NODELABEL_MON[${NL}]}"
        done
        POD_LABEL_MON+="\n"
      fi
      if [[ -n "${SOLBK_ANTIAFFINITY_NS[@]}" ]]; then
        # construct pod anti affinity
        POD_ANTIAFFINITY_SPEC="      affinity:
        podAntiAffinity:
          preferredDuringSchedulingIgnoredDuringExecution:
          - weight: ${SOLBK_ANTIAFFINITY_WT}
            podAffinityTerm:
              topologyKey: kubernetes.io/hostname
              labelSelector:
                matchLabels:
                  app.kubernetes.io/name: pubsubpluseventbroker
              namespaces:"
        for ns in "${SOLBK_ANTIAFFINITY_NS[@]}"; do
          POD_ANTIAFFINITY_SPEC+="\n              - ${ns}"
        done
      fi
      
      echo "  nodeAssignment:"
      echo "  - name: Primary"
      echo -e "    spec:\n${POD_LABEL_PRI}${POD_ANTIAFFINITY_SPEC}"
      if [[ "${SOLBK_REDUNDANCY}" == "true" ]]; then
        echo "  - name: Backup"
        echo -e "    spec:\n${POD_LABEL_BKP}${POD_ANTIAFFINITY_SPEC}"
        echo "  - name: Monitor"
        echo -e "    spec:\n${POD_LABEL_MON}${POD_ANTIAFFINITY_SPEC}"
      fi
    fi
      
    echo "  service:
    type: LoadBalancer"
      
    if [[ -n "${SOLBK_LOADBALANCER_IP}" ]] || [[ -n "${SOLBK_IPPOOL}" ]]; then
      echo "    annotations:"
      [[ -n "${SOLBK_LOADBALANCER_IP}" ]] && echo "      metallb.universe.tf/loadBalancerIPs: ${SOLBK_LOADBALANCER_IP}
      metallb.io/loadBalancerIPs: ${SOLBK_LOADBALANCER_IP}"
      [[ -n "${SOLBK_IPPOOL}" ]] && echo "      metallb.universe.tf/address-pool: ${SOLBK_IPPOOL}
      metallb.io/address-pool: ${SOLBK_IPPOOL}"
    fi
        
    echo "    ports:
    - containerPort: 8080
      name: tcp-semp
      protocol: TCP
      servicePort: 8080
    - containerPort: 1943
      name: tls-semp
      protocol: TCP
      servicePort: 1943
    - containerPort: 55555
      name: tcp-smf
      protocol: TCP
      servicePort: 55555
    - containerPort: 55003
      name: tcp-smfcomp
      protocol: TCP
      servicePort: 55003
    - containerPort: 55443
      name: tls-smf
      protocol: TCP
      servicePort: 55443
    - containerPort: 55556
      name: tcp-smfroute
      protocol: TCP
      servicePort: 55556
    - containerPort: 8008
      name: tcp-web
      protocol: TCP
      servicePort: 8008
    - containerPort: 1443
      name: tls-web
      protocol: TCP
      servicePort: 1443
    - containerPort: 9000
      name: tcp-rest
      protocol: TCP
      servicePort: 9000
    - containerPort: 9443
      name: tls-rest
      protocol: TCP
      servicePort: 9443
    - containerPort: 5672
      name: tcp-amqp
      protocol: TCP
      servicePort: 5672
    - containerPort: 5671
      name: tls-amqp
      protocol: TCP
      servicePort: 5671
    - containerPort: 1883
      name: tcp-mqtt
      protocol: TCP
      servicePort: 1883
    - containerPort: 8883
      name: tls-mqtt
      protocol: TCP
      servicePort: 8883
    - containerPort: 8000
      name: tcp-mqttweb
      protocol: TCP
      servicePort: 8000
    - containerPort: 8443
      name: tls-mqttweb
      protocol: TCP
      servicePort: 8443"
}

while [[ $# -gt 0 ]]; do
	case "$1" in
		--keep-yaml)
			KEEP=true
			shift 1
			;;
    --only-gen-yaml)
       GENONLY=true
       shift 1
       ;;
		*)
			shift
			;;
	esac
done

gen_yaml > .broker.yaml

[[ "${GENONLY}" != "true" ]] && ${KUBE} apply -f .broker.yaml

[[ "${GENONLY}" != "true" ]] && [[ "${KEEP}" != "true" ]] && rm .broker.yaml

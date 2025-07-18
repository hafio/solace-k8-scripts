#!/bin/bash

SELECT_ENV_FILE="000-env.sh"
if [[ -f "`dirname $0`/${SELECT_ENV_FILE}" ]]; then
	source "`dirname $0`/${SELECT_ENV_FILE}"
else 
	echo "Environment file '${SELECT_ENV_FILE}' not found"
	exit 1
fi

if [[ "$1" == "?"* ]] || [[ "$1" == "-h" ]] || [[ "$1" == "--h"* ]]; then
  echo "Usage: $0
  
  Description:
    
    IMPORTANT: THIS SCRIPT IS FOR SOLACE KUBERNETES DEPLOYMENT ONLY
    
    Execute a series of Solace CLI \"show\" commands and save their output to \"configs/.out\" folder.
    Also execute gather diagnositcs on the broker. Zips the output of \"shows\" and transfer them to
    \$SOLBK_DIAG_DIR folder.
"
  exit
fi

echo 'end
home
no paging

! some commands for specific to appliance vs software

show acl-profile * > configs/.out/show-aclprofiles.out
show acl-profile * detail > configs/.out/show-aclprofiles-detail.out
show alarm > configs/.out/show-alarm.out
show authentication > configs/.out/show-auth.out
show authentication access-level > configs/.out/show-auth-access-level.out
show authentication access-level detail > configs/.out/show-auth-access-level-detail.out
show backup > configs/.out/show-backup.out
show bridge * > configs/.out/show-bridges.out
show bridge * detail > configs/.out/show-bridges-detail.out
show bridge * stats > configs/.out/show-bridge-stats.out
show bridge * stats queues > configs/.out/show-bridge-stats-queues.out
show cache-cluster * detail > configs/.out/show-cachecluster.out
show cache-instance * detail > configs/.out/show-cacheinstance.out
show client * > configs/.out/show-clients.out
show client * detail > configs/.out/show-clients-detail.out
show client-profile * > configs/.out/show-clientprofile.out
show client-profile * detail > configs/.out/show-clientprofile-detail.out
show client-username * > configs/.out/show-client-username.out
show client-username * detail > configs/.out/show-client-username-detail.out
show clock detail > configs/.out/show-clock-detail.out
show cluster * detail > configs/.out/show-cluster-detail.out
show cluster * link * detail > configs/.out/show-cluster-link-detail.out
show compression > configs/.out/show-compression.out
show config-sync > configs/.out/show-config-sync.out
show config-sync database > configs/.out/show-config-sync-database.out
show config-sync database detail > configs/.out/show-config-sync-database-detail.out
show cspf stats > configs/.out/show-cspf-stats.out
show current-config all > configs/.out/show-currentconfig-all.out
show current-config message-vpn * > configs/.out/show-currentconfig-vpns.out
show debug lldp > configs/.out/show-debug-lldp.out
show disk > configs/.out/show-disk.out
show disk detail > configs/.out/show-disk-detail.out
show distributed-cache * detail > configs/.out/show-distributedcache.out
show dns > configs/.out/show-dns.out
show hardware details > configs/.out/show-hardware-details.out
show hardware post > configs/.out/show-hardware-post.out
show hostname > configs/.out/show-hostname.out
show interface detail > configs/.out/show-interface-detail.out
show ip vrf management > configs/.out/show-vrf-mgmt.out
show ip vrf msg-backbone > configs/.out/show-vrf-msg-backbone.out
show jndi connection-factory * detail > configs/.out/show-jndi-cf.out
show jndi queue * detail > configs/.out/show-jndi-queues.out
show jndi summary > configs/.out/show-jndi-summary.out
show jndi topic * detail > configs/.out/show-jndi-topics.out
show kerberos keytab > configs/.out/show-kerberose-keytab.out
show kerberos keytab detail > configs/.out/show-kerberose-keytab-details.out
show ldap-profile * detail > configs/.out/show-ldap-profile-detail.out
show memory > configs/.out/show-memory.out
show message-spool detail > configs/.out/show-message-spool-detail.out
show message-spool message-vpn * detail > configs/.out/show-message-spool-vpn-detail.out
show message-spool rates > configs/.out/show-message-spool-rates.out
show message-spool stats > configs/.out/show-message-spool-stats.out
show message-vpn * > configs/.out/show-vpns.out
show message-vpn * detail > configs/.out/show-vpn-details.out
show message-vpn * dynamic-message-routing > configs/.out/show-vpn-dmr.out
show message-vpn * dynamic-message-routing dmr-bridge * > configs/.out/show-vpn-dmr-bridge.out
show message-vpn * mqtt > configs/.out/show-vpn-mqtt.out
show message-vpn * mqtt mqtt-session * > configs/.out/show-vpn-mqtt-session.out
show message-vpn * mqtt retain cache * > configs/.out/show-vpn-mqtt-retain-cache.out
show message-vpn * replication > configs/.out/show-vpns-replication.out
show message-vpn * rest > configs/.out/show-vpn-rest.out
show message-vpn * rest rest-delivery-point * detail > configs/.out/show-vpn-rdp-detail.out
show message-vpn * service > configs/.out/show-vpn-service.out
show mqtt > configs/.out/show-mqtt.out
show oauth-profile * detail > configs/.out/show-oauth-profile-detail.out
show product-key > configs/.out/show-product-key.out
show queue * > configs/.out/show-queues.out
show queue * detail > configs/.out/show-queues-details.out
show redundancy > configs/.out/show-redundancy.out
show redundancy detail > configs/.out/show-redundancy-detail.out
show redundancy group > configs/.out/show-redundancy-group.out
show replay-log * > configs/.out/show-replay-log.out
show replicated-topic * > configs/.out/show-replicated-topics.out
show replication > configs/.out/show-replication.out
show router-name > configs/.out/show-routername.out
show routing > configs/.out/show-routing.out
show service > configs/.out/show-service.out
show service semp > configs/.out/show-service.semp.out
show service virtual-hostname * > configs/.out/show-service-virtual-hostname.out
show service web-transport > configs/.out/show-service-web-transport.out
show snmp > configs/.out/show-snmp.out
show snmp trap * > configs/.out/show-snmp-trap.out
show ssl allow-tls-version > configs/.out/show-ssl-allowed-tls.out
show ssl certificate-files > configs/.out/show-ssl-certificate-files.out
show ssl cipher-suite-list default > configs/.out/show-ssl-cipher-default.out
show ssl cipher-suite-list management > configs/.out/show-ssl-cipher-management.out
show ssl cipher-suite-list msg-backbone > configs/.out/show-ssl-cipher-msg-backbone.out
show ssl cipher-suite-list ssh > configs/.out/show-ssl-cipher-ssh.out
show ssl server-certificate > configs/.out/show-ssl-server-certificate.out
show ssl server-certificate detail > configs/.out/show-ssl-server-certificate-detail.out
show syslog > configs/.out/show-syslog.out
show system detail > configs/.out/show-system.out
show system health > configs/.out/show-system-health.out
show telemetry > configs/.out/show-telemetry.out
show topic-endpoint * detail > configs/.out/show-topicendpoints.out
show username * > configs/.out/show-username.out
show username * detail > configs/.out/show-username-detail.out
show version > configs/.out/show-version.out
show web-manager > configs/.out/show-web-manager.out

! gather diagnostics 3 days
end
home
enable
admin
gather-diagnostics days-of-history 3 no-encrypt' > .tmp.cli

echo '#!/bin/bash

cd /usr/sw/jail
rm -f gather-configs.zip
mv configs/.out cli-out
zip gather-configs.zip -q -r cli-out/*
' > .tmp.sh
chmod 777 .tmp.sh

declare nodes=("p" "b" "m")
[[ "${SOLBK_REDUNDANCY}" != "true" ]] && nodes=("p")

mkdir -p ${SOLBK_DIAG_DIR}

for node in "${nodes[@]}"; do
	echo "Gathering Diagnostics for '${node}' node..."
  
  echo " - Creating tmp directory..."
  ${KUBE} exec -n ${SOLBK_NS} ${SOLBK_NAME}-pubsubplus-${node}-0 -- mkdir -p /usr/sw/jail/configs/.out
  
  # copy files
  echo " - Copy Files..."
	${KUBE} cp -n ${SOLBK_NS} .tmp.cli ${SOLBK_NAME}-pubsubplus-${node}-0:/usr/sw/jail/cliscripts/.gather-configs.cli
	${KUBE} cp -n ${SOLBK_NS} .tmp.sh ${SOLBK_NAME}-pubsubplus-${node}-0:/usr/sw/jail/zip-configs.sh
  
  # run diagnostics and config retrieval
  echo " - Running CLI scripts..."
	${KUBE} exec -n ${SOLBK_NS} ${SOLBK_NAME}-pubsubplus-${node}-0 -- /usr/sw/loads/currentload/bin/cli -Apes .gather-configs.cli > .tmp
  
  # zip all cli outputs and copy over
  echo " - Zipping ouputs..."
  ${KUBE} exec -n ${SOLBK_NS} ${SOLBK_NAME}-pubsubplus-${node}-0 -- /usr/sw/jail/zip-configs.sh
  PODHOSTNAME=`${KUBE} exec -n ${SOLBK_NS} ${SOLBK_NAME}-pubsubplus-${node}-0 -- hostname`
  ${KUBE} cp -n ${SOLBK_NS} ${SOLBK_NAME}-pubsubplus-${node}-0:/usr/sw/jail/gather-configs.zip "${SOLBK_DIAG_DIR}/gather-configs-${PODHOSTNAME}-`date +%Y-%m-%d-T%H%M`.zip"
  
  # clean up zip file and output files
  echo " - Cleaning up..."
  ${KUBE} exec -n ${SOLBK_NS} ${SOLBK_NAME}-pubsubplus-${node}-0 -- rm -rf /usr/sw/jail/cli-out /usr/sw/jail/zip-configs.zip
  
  #get filename for gather-diagnostics
  echo " - Retrieving gather-diagnostics..."
	grep "Diagnostics saved" .tmp > .tmp2
	if [[ `cat .tmp2 | wc -l` -eq 1 ]]; then
    DIAG_FILE=`cat .tmp2`
    DIAG_FILE=${DIAG_FILE#*: }
    
    # copy file and delete
		${KUBE} cp -n ${SOLBK_NS} ${SOLBK_NAME}-pubsubplus-${node}-0:/usr/sw/jail/${DIAG_FILE} ${SOLBK_DIAG_DIR}/${DIAG_FILE#logs/}
		${KUBE} exec -n ${SOLBK_NS} ${SOLBK_NAME}-pubsubplus-${node}-0 -- rm -f /usr/sw/jail/${DIAG_FILE}
 	fi
done

echo -e "\nListing files in ${SOLBK_DIAG_DIR}"
ls ${SOLBK_DIAG_DIR}/*

rm -f .tmp .tmp2 .tmp.cli .tmp.sh

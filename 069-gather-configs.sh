#!/bin/bash

SELECT_ENV_FILE="000-env.sh"
if [[ -f "`dirname $0`/${SELECT_ENV_FILE}" ]]; then
	source "`dirname $0`/${SELECT_ENV_FILE}"
else 
	echo "Environment file '${SELECT_ENV_FILE}' not found"
	exit 1
fi

echoUsage() {
  echo "Usage: $0 [OPTIONS]
  
  Description:
    
    IMPORTANT: THIS SCRIPT IS FOR SOLACE KUBERNETES DEPLOYMENT ONLY
    
    Execute a series of Solace CLI \"show\" commands and save their output to \"configs/.out\" folder.
    Also execute gather diagnositcs on the broker. Zips the output of \"shows\" and transfer them to
    \$SOLBK_DIAG_DIR folder.
  
  OPTIONS:
  --days <days>: specify number of days for gather-diagnostic logs. Default = 3 (if not specified)
"
  exit
}

while [[ $# -gt 0 ]]; do
	case "$1" in
		-h|--h*|\?*)
			echoUsage
			;;
		--days)
      DAYS="$2"
			shift 2
			;;
    *)
      shift
      ;;
	esac
done

DAYS=${DAYS:-3}
TMPFILE=.tmp-${BASHPID}
TMPFILE2=.tmp2-${BASHPID}
CLITMPFILE=.tmp.cli-${BASHPID}
SHTMPFILE=.tmp.sh-${BASHPID}

echo 'end
home
no paging

! some commands for specific to appliance vs software

show acl-profile * > configs/cliout/show-aclprofiles.out
show acl-profile * detail > configs/cliout/show-aclprofiles-detail.out
show alarm > configs/cliout/show-alarm.out
show authentication > configs/cliout/show-auth.out
show authentication access-level > configs/cliout/show-auth-access-level.out
show authentication access-level detail > configs/cliout/show-auth-access-level-detail.out
show backup > configs/cliout/show-backup.out
show bridge * > configs/cliout/show-bridges.out
show bridge * detail > configs/cliout/show-bridges-detail.out
show bridge * stats > configs/cliout/show-bridge-stats.out
show bridge * stats queues > configs/cliout/show-bridge-stats-queues.out
show cache-cluster * detail > configs/cliout/show-cachecluster.out
show cache-instance * detail > configs/cliout/show-cacheinstance.out
show client * > configs/cliout/show-clients.out
show client * detail > configs/cliout/show-clients-detail.out
show client-certificate-authority ca-name * cert > configs/cliout/show-client-cert-auth-cert.out
show client-certificate-authority ca-name * detail > configs/cliout/show-client-cert-auth-detail.out
show client-profile * > configs/cliout/show-clientprofile.out
show client-profile * detail > configs/cliout/show-clientprofile-detail.out
show client-username * > configs/cliout/show-client-username.out
show client-username * detail > configs/cliout/show-client-username-detail.out
show clock detail > configs/cliout/show-clock-detail.out
show cluster * > configs/cliout/show-cluster.out
show cluster * detail > configs/cliout/show-cluster-detail.out
show cluster * link * detail > configs/cliout/show-cluster-link-detail.out
show compression > configs/cliout/show-compression.out
show config-sync > configs/cliout/show-config-sync.out
show config-sync database > configs/cliout/show-config-sync-database.out
show config-sync database detail > configs/cliout/show-config-sync-database-detail.out
show cspf stats > configs/cliout/show-cspf-stats.out
show current-config all > configs/cliout/show-currentconfig-all.out
show current-config message-vpn * > configs/cliout/show-currentconfig-vpns.out
show debug lldp > configs/cliout/show-debug-lldp.out
show disk > configs/cliout/show-disk.out
show disk detail > configs/cliout/show-disk-detail.out
show distributed-cache * detail > configs/cliout/show-distributedcache.out
show dns > configs/cliout/show-dns.out
show domain-certificate-authority ca-name * cert > configs/cliout/show-domain-cert-auth.out
show hardware details > configs/cliout/show-hardware-details.out
show hardware post > configs/cliout/show-hardware-post.out
show hostname > configs/cliout/show-hostname.out
show interface detail > configs/cliout/show-interface-detail.out
show ip vrf management > configs/cliout/show-vrf-mgmt.out
show ip vrf msg-backbone > configs/cliout/show-vrf-msg-backbone.out
show jndi connection-factory * detail > configs/cliout/show-jndi-cf.out
show jndi queue * detail > configs/cliout/show-jndi-queues.out
show jndi summary > configs/cliout/show-jndi-summary.out
show jndi topic * detail > configs/cliout/show-jndi-topics.out
show kerberos keytab > configs/cliout/show-kerberose-keytab.out
show kerberos keytab detail > configs/cliout/show-kerberose-keytab-details.out
show ldap-profile * detail > configs/cliout/show-ldap-profile-detail.out
show logging command > configs/cliout/show-logging-command.out
show logging config > configs/cliout/show-logging-config.out
show logging debug > configs/cliout/show-logging-debug.out
show logging event > configs/cliout/show-logging-event.out
show memory > configs/cliout/show-memory.out
show message-spool detail > configs/cliout/show-message-spool-detail.out
show message-spool message-vpn * detail > configs/cliout/show-message-spool-vpn-detail.out
show message-spool rates > configs/cliout/show-message-spool-rates.out
show message-spool stats > configs/cliout/show-message-spool-stats.out
show message-vpn * > configs/cliout/show-vpns.out
show message-vpn * authorization > configs/cliout/show-vpn-auth.out
show message-vpn * authorization authorization-group * > configs/cliout/show-vpn-auth-authgroup.out
show message-vpn * authorization authorization-group * detail > configs/cliout/show-vpn-auth-authgroup-detail.out
show message-vpn * detail > configs/cliout/show-vpn-details.out
show message-vpn * dynamic-message-routing > configs/cliout/show-vpn-dmr.out
show message-vpn * dynamic-message-routing dmr-bridge * > configs/cliout/show-vpn-dmr-bridge.out
show message-vpn * mqtt > configs/cliout/show-vpn-mqtt.out
show message-vpn * mqtt mqtt-session * > configs/cliout/show-vpn-mqtt-session.out
show message-vpn * mqtt retain cache * > configs/cliout/show-vpn-mqtt-retain-cache.out
show message-vpn * replication > configs/cliout/show-vpns-replication.out
show message-vpn * replication detail > configs/cliout/show-vpns-repl-detail.out
show message-vpn * rest > configs/cliout/show-vpn-rest.out
show message-vpn * rest rest-delivery-point * detail > configs/cliout/show-vpn-rdp-detail.out
show message-vpn * service > configs/cliout/show-vpn-service.out
show mqtt > configs/cliout/show-mqtt.out
show oauth-profile * detail > configs/cliout/show-oauth-profile-detail.out
show product-key > configs/cliout/show-product-key.out
show queue * > configs/cliout/show-queues.out
show queue * detail > configs/cliout/show-queues-details.out
show redundancy > configs/cliout/show-redundancy.out
show redundancy detail > configs/cliout/show-redundancy-detail.out
show redundancy group > configs/cliout/show-redundancy-group.out
show replay-log * > configs/cliout/show-replay-log.out
show replicated-topic * > configs/cliout/show-replicated-topics.out
show replication > configs/cliout/show-replication.out
show router-name > configs/cliout/show-routername.out
show routing > configs/cliout/show-routing.out
show service > configs/cliout/show-service.out
show service semp > configs/cliout/show-service.semp.out
show service virtual-hostname * > configs/cliout/show-service-virtual-hostname.out
show service web-transport > configs/cliout/show-service-web-transport.out
show snmp > configs/cliout/show-snmp.out
show snmp trap * > configs/cliout/show-snmp-trap.out
show ssl allow-tls-version > configs/cliout/show-ssl-allowed-tls.out
show ssl certificate-files > configs/cliout/show-ssl-certificate-files.out
show ssl cipher-suite-list default > configs/cliout/show-ssl-cipher-default.out
show ssl cipher-suite-list management > configs/cliout/show-ssl-cipher-management.out
show ssl cipher-suite-list msg-backbone > configs/cliout/show-ssl-cipher-msg-backbone.out
show ssl cipher-suite-list ssh > configs/cliout/show-ssl-cipher-ssh.out
show ssl server-certificate > configs/cliout/show-ssl-server-certificate.out
show ssl server-certificate detail > configs/cliout/show-ssl-server-certificate-detail.out
show syslog > configs/cliout/show-syslog.out
show system post > configs/cliout/show-system-post.out
show system detail > configs/cliout/show-system.out
show system health > configs/cliout/show-system-health.out
show telemetry > configs/cliout/show-telemetry.out
show topic-endpoint * detail > configs/cliout/show-topicendpoints.out
show username * > configs/cliout/show-username.out
show username * detail > configs/cliout/show-username-detail.out
show version > configs/cliout/show-version.out
show web-manager > configs/cliout/show-web-manager.out

! gather diagnostics '${DAYS}' days
end
home
enable
admin
gather-diagnostics days-of-history '${DAYS}' no-encrypt' > ${CLITMPFILE}

echo '#!/bin/bash

cd /usr/sw/jail
rm -f gather-configs.zip
mv configs/.out cli-out
zip gather-configs.zip -q -r cli-out/*
' > ${SHTMPFILE}
chmod 777 ${SHTMPFILE}

declare nodes=("p" "b" "m")
[[ "${SOLBK_REDUNDANCY}" != "true" ]] && nodes=("p")

mkdir -p ${SOLBK_DIAG_DIR}

for node in "${nodes[@]}"; do
	echo "Gathering Diagnostics for '${node}' node..."
  
  echo " - Creating tmp directory..."
  ${KUBE} exec -n ${SOLBK_NS} ${SOLBK_NAME}-pubsubplus-${node}-0 -- mkdir -p /usr/sw/jail/configs/.out
  
  # copy files
  echo " - Copy Files..."
	${KUBE} cp -n ${SOLBK_NS} ${CLITMPFILE} ${SOLBK_NAME}-pubsubplus-${node}-0:/usr/sw/jail/cliscripts/.gather-configs.cli
	${KUBE} cp -n ${SOLBK_NS} ${SHTMPFILE} ${SOLBK_NAME}-pubsubplus-${node}-0:/usr/sw/jail/zip-configs.sh
  
  # run diagnostics and config retrieval
  echo " - Running CLI scripts..."
	${KUBE} exec -n ${SOLBK_NS} ${SOLBK_NAME}-pubsubplus-${node}-0 -- /usr/sw/loads/currentload/bin/cli -Apes .gather-configs.cli > ${TMPFILE}
  
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
	grep "Diagnostics saved" ${TMPFILE} > ${TMPFILE2}
	if [[ `cat ${TMPFILE2} | wc -l` -eq 1 ]]; then
    DIAG_FILE=`cat ${TMPFILE2}`
    DIAG_FILE=${DIAG_FILE#*: }
    
    # copy file and delete cli + script
		${KUBE} cp -n ${SOLBK_NS} ${SOLBK_NAME}-pubsubplus-${node}-0:/usr/sw/jail/${DIAG_FILE} ${SOLBK_DIAG_DIR}/${DIAG_FILE#logs/}
		${KUBE} exec -n ${SOLBK_NS} ${SOLBK_NAME}-pubsubplus-${node}-0 -- rm -rf /usr/sw/jail/${DIAG_FILE} /usr/sw/jail/cliscripts/.gather-configs.cli /usr/sw/jail/zip-configs.sh
 	fi
done

echo -e "\nListing files in ${SOLBK_DIAG_DIR}"
ls ${SOLBK_DIAG_DIR}/*

rm -f ${TMPFILE} ${TMPFILE2} ${CLITMPFILE} ${SHTMPFILE}
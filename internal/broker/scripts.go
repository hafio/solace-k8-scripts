package broker

import (
	"fmt"
	"sort"
	"strings"

	"solace/internal/config"
)

// The functions here are pure: each returns the exact Solace CLI script body a
// config/verify op uploads and executes. Keeping them pure (no transport, no
// I/O) makes them golden-testable in scripts_test.go, the same discipline as
// internal/render. They are faithful ports of the heredocs in the numbered bash
// scripts; comments cite the source line ranges.

// showRedundancyScript is the one-line probe used by leader/redundancy polling
// (050 line 34, 061 line 25).
func showRedundancyScript() string { return "show redundancy\n" }

// showRedundancyDetailScript is the timeout diagnostic dumped by 050 (lines 66-67).
func showRedundancyDetailScript() string { return "no paging\nshow redundancy detail\n" }

// assertLeaderScript restores the Primary as config-sync leader for the router
// and all VPNs (050 lines 38-45).
func assertLeaderScript() string {
	return `home
no paging
enable
admin
config-sync assert-leader router
config-sync assert-leader message-vpn *
show config-sync database
`
}

// revertActivityScript reverts activity back to the local node (050 lines 23-27).
func revertActivityScript() string {
	return `home
no paging
enable
admin
redundancy revert-activity
`
}

// releaseActivityScript releases activity from the Primary (061 lines 30-34).
func releaseActivityScript() string {
	return `home
no paging
enable
configure
redundancy release-activity
`
}

// noReleaseActivityScript un-releases the Primary (061 lines 38-42).
func noReleaseActivityScript() string {
	return `home
no paging
enable
configure
no redundancy release-activity
`
}

// revertActivityConfigureScript reverts activity from the Backup during a
// redundancy test (061 lines 46-50). The trailing space after the command is
// preserved from the source script.
func revertActivityConfigureScript() string {
	return "home\nno paging\nenable\nadmin\nredundancy revert-activity \n"
}

// serverCertScript applies the uploaded TLS server certificate (051 lines 40-43).
// dt is the date stamp (YYYY-MM-DD) that names the uploaded tls-<dt>.crt.key file.
func serverCertScript(dt string) string {
	return fmt.Sprintf("enable\nconfigure\nssl server-certificate %s\nshow ssl server-certificate detail\n", serverCertFile(dt))
}

// serverCertFile is the in-broker filename the concatenated key+cert+CAs are
// uploaded as, and the name the CLI loads (051 lines 42, 51).
func serverCertFile(dt string) string { return "tls-" + dt + ".crt.key" }

// domainCertsScript loads each domain certificate authority (052 lines 20-34).
// cas maps CA name -> certificate filename (already uploaded to the certs dir).
// CA names are emitted in sorted order for deterministic output (bash iterated a
// hash in unspecified order).
func domainCertsScript(cas map[string]string) string {
	var b strings.Builder
	b.WriteString("no paging\nenable\nconfigure\nssl\n")
	for _, ca := range sortedKeys(cas) {
		fmt.Fprintf(&b, "create domain-certificate-authority %s\ncertificate file %s\nexit\n", ca, cas[ca])
	}
	b.WriteString("end\nshow domain-certificate-authority ca-name *\n")
	return b.String()
}

// removeDomainCertsScript deletes each domain certificate authority (150 lines
// 20-31). cas are the CA names to remove, emitted in the order given (the caller
// sorts them for determinism, since bash iterated a hash in unspecified order).
func removeDomainCertsScript(cas []string) string {
	var b strings.Builder
	b.WriteString("no paging\nenable\nconfigure\n")
	for _, ca := range cas {
		fmt.Fprintf(&b, "no ssl domain-certificate-authority %s\n", ca)
	}
	b.WriteString("home\nshow domain-certificate-authority ca-name *\n")
	return b.String()
}

// disableDefaultVPNScript shuts down the default message-VPN, its default
// client-username, and every service (053 lines 20-51).
func disableDefaultVPNScript() string {
	return `home
enable
configure
message-vpn "default"
  authentication
    basic shutdown
    client-certificate
      shutdown
      exit
    exit
  service smf plain-text shutdown
  service smf ssl shutdown
  service web-transport plain-text shutdown
  service web-transport ssl shutdown
  service rest incoming plain-text shutdown
  service rest incoming ssl shutdown
  service mqtt plain-text shutdown
  service mqtt ssl shutdown
  service mqtt websocket shutdown
  service mqtt websocket-secure shutdown
  service amqp plain-text shutdown
  service amqp ssl shutdown
  no ssl allow-downgrade-to-plain-text
  exit

client-username "default" message-vpn "default"
  shutdown
  exit

message-vpn "default"
  shutdown
  exit
`
}

// showVPNScript lists all message-VPNs. 053 (lines 58-61) wraps it in
// home/enable/configure; 054 (line 20) uses the bare form to parse VPN names.
func showVPNScript() string      { return "home\nenable\nconfigure\nshow message-vpn *\n" }
func showVPNBareScript() string  { return "show message-vpn *\n" }

// disableDefaultUsersScript shuts down the "default" client-username in each of
// the given VPNs (054 lines 33-42).
func disableDefaultUsersScript(vpns []string) string {
	var b strings.Builder
	b.WriteString("home\nenable\nconfigure\n")
	for _, vpn := range vpns {
		fmt.Fprintf(&b, "client-username default message-vpn %q\nshutdown\nexit\n", vpn)
	}
	b.WriteString("end\nshow client-username default message-vpn *\n")
	return b.String()
}

// additionalUsersScript creates each management (CLI) user with its password and
// global access level. It has no bash ancestor: the operator offers no way to
// deliver extra users declaratively -- extra data keys in the credentials Secret are
// ignored, and extraEnvVars/extraEnvVarsSecret would expose the passwords in the
// pod's environment -- so on k8s the users are created over the CLI instead.
//
// `create` fails when the user already exists, which is deliberate: the caller
// surfaces that as an error rather than silently reconciling a password the operator
// may have changed on purpose. Both values are quoted, and the password's characters
// are constrained upstream (config.cliForbiddenPassword) to the set the CLI accepts
// inside quotes, so no escaping is possible or needed here.
//
// CONTAINS SECRETS: the returned body carries every password, so the caller must
// upload it on stdin and must not echo the CLI's own output, which repeats the
// command lines back.
func additionalUsersScript(users []config.AdditionalUser) string {
	var b strings.Builder
	b.WriteString("home\nno paging\nenable\nconfigure\n")
	for _, u := range users {
		fmt.Fprintf(&b, "create username %q password %q\n", u.Username, u.Password)
		fmt.Fprintf(&b, "global-access-level %s\nexit\n", u.AccessLevel)
	}
	b.WriteString("end\n")
	return b.String()
}

// productKeysScript applies each product key (057 lines 24-29).
func productKeysScript(keys []string) string {
	var b strings.Builder
	b.WriteString("enable\nadmin\n")
	for _, k := range keys {
		fmt.Fprintf(&b, "product-key %s\n", k)
	}
	b.WriteString("show product-key\n")
	return b.String()
}

// parseVPNNames extracts message-VPN names from `show message-vpn *` output,
// porting the parser in 054 (lines 24-31): skip until a 30-dash separator, then
// take the first token of each subsequent non-comment line's first 32 columns.
func parseVPNNames(output string) []string {
	var vpns []string
	parsing := false
	for _, raw := range strings.Split(output, "\n") {
		line := strings.TrimRight(raw, "\r")
		switch {
		case strings.HasPrefix(line, strings.Repeat("-", 30)):
			parsing = true
		case parsing && !strings.HasPrefix(line, "#"):
			col := line
			if len(col) > 32 {
				col = col[:32]
			}
			vpns = append(vpns, strings.Fields(col)...)
		}
	}
	return vpns
}

// gatherConfigsScript is the ~110-command show + gather-diagnostics run collected
// by 069 (lines 38-161). days sets days-of-history for gather-diagnostics.
func gatherConfigsScript(days int) string {
	var b strings.Builder
	b.WriteString("home\nno paging\n\n! some commands for specific to appliance vs software\n\n")
	for _, cmd := range gatherShowCommands {
		fmt.Fprintf(&b, "show %s > configs/cliout/show-%s.out\n", cmd.args, cmd.out)
	}
	fmt.Fprintf(&b, "\n! gather diagnostics '%d' days\nend\nhome\nenable\nadmin\ngather-diagnostics days-of-history '%d' no-encrypt\n", days, days)
	return b.String()
}

// zipConfigsScript is the in-broker helper that zips the collected show output
// (069 lines 163-169).
func zipConfigsScript() string {
	return `#!/bin/bash

cd /usr/sw/jail
rm -f gather-configs.zip
mv configs/cliout cli-out
zip gather-configs.zip -q -r cli-out/*
`
}

// showCmd pairs a `show` argument list with the output-file stem it is written to.
type showCmd struct{ args, out string }

// gatherShowCommands is the ordered show-command table from 069 (lines 43-154).
var gatherShowCommands = []showCmd{
	{"acl-profile *", "aclprofiles"},
	{"acl-profile * detail", "aclprofiles-detail"},
	{"alarm", "alarm"},
	{"authentication", "auth"},
	{"authentication access-level", "auth-access-level"},
	{"authentication access-level detail", "auth-access-level-detail"},
	{"backup", "backup"},
	{"bridge *", "bridges"},
	{"bridge * detail", "bridges-detail"},
	{"bridge * stats", "bridge-stats"},
	{"bridge * stats queues", "bridge-stats-queues"},
	{"cache-cluster * detail", "cachecluster"},
	{"cache-instance * detail", "cacheinstance"},
	{"client *", "clients"},
	{"client * detail", "clients-detail"},
	{"client-certificate-authority ca-name * cert", "client-cert-auth-cert"},
	{"client-certificate-authority ca-name * detail", "client-cert-auth-detail"},
	{"client-profile *", "clientprofile"},
	{"client-profile * detail", "clientprofile-detail"},
	{"client-username *", "client-username"},
	{"client-username * detail", "client-username-detail"},
	{"clock detail", "clock-detail"},
	{"cluster *", "cluster"},
	{"cluster * detail", "cluster-detail"},
	{"cluster * link * detail", "cluster-link-detail"},
	{"compression", "compression"},
	{"config-sync", "config-sync"},
	{"config-sync database", "config-sync-database"},
	{"config-sync database detail", "config-sync-database-detail"},
	{"cspf stats", "cspf-stats"},
	{"current-config all", "currentconfig-all"},
	{"current-config message-vpn *", "currentconfig-vpns"},
	{"debug lldp", "debug-lldp"},
	{"disk", "disk"},
	{"disk detail", "disk-detail"},
	{"distributed-cache * detail", "distributedcache"},
	{"dns", "dns"},
	{"domain-certificate-authority ca-name * cert", "domain-cert-auth"},
	{"hardware details", "hardware-details"},
	{"hardware post", "hardware-post"},
	{"hostname", "hostname"},
	{"interface detail", "interface-detail"},
	{"ip vrf management", "vrf-mgmt"},
	{"ip vrf msg-backbone", "vrf-msg-backbone"},
	{"jndi connection-factory * detail", "jndi-cf"},
	{"jndi queue * detail", "jndi-queues"},
	{"jndi summary", "jndi-summary"},
	{"jndi topic * detail", "jndi-topics"},
	{"kerberos keytab", "kerberose-keytab"},
	{"kerberos keytab detail", "kerberose-keytab-details"},
	{"ldap-profile * detail", "ldap-profile-detail"},
	{"logging command", "logging-command"},
	{"logging config", "logging-config"},
	{"logging debug", "logging-debug"},
	{"logging event", "logging-event"},
	{"memory", "memory"},
	{"message-spool detail", "message-spool-detail"},
	{"message-spool message-vpn * detail", "message-spool-vpn-detail"},
	{"message-spool rates", "message-spool-rates"},
	{"message-spool stats", "message-spool-stats"},
	{"message-vpn *", "vpns"},
	{"message-vpn * authorization", "vpn-auth"},
	{"message-vpn * authorization authorization-group *", "vpn-auth-authgroup"},
	{"message-vpn * authorization authorization-group * detail", "vpn-auth-authgroup-detail"},
	{"message-vpn * detail", "vpn-details"},
	{"message-vpn * dynamic-message-routing", "vpn-dmr"},
	{"message-vpn * dynamic-message-routing dmr-bridge *", "vpn-dmr-bridge"},
	{"message-vpn * mqtt", "vpn-mqtt"},
	{"message-vpn * mqtt mqtt-session *", "vpn-mqtt-session"},
	{"message-vpn * mqtt retain cache *", "vpn-mqtt-retain-cache"},
	{"message-vpn * replication", "vpns-replication"},
	{"message-vpn * replication detail", "vpns-repl-detail"},
	{"message-vpn * rest", "vpn-rest"},
	{"message-vpn * rest rest-delivery-point * detail", "vpn-rdp-detail"},
	{"message-vpn * service", "vpn-service"},
	{"mqtt", "mqtt"},
	{"oauth-profile * detail", "oauth-profile-detail"},
	{"product-key", "product-key"},
	{"queue *", "queues"},
	{"queue * detail", "queues-details"},
	{"redundancy", "redundancy"},
	{"redundancy detail", "redundancy-detail"},
	{"redundancy group", "redundancy-group"},
	{"replay-log *", "replay-log"},
	{"replicated-topic *", "replicated-topics"},
	{"replication", "replication"},
	{"router-name", "routername"},
	{"routing", "routing"},
	{"service", "service"},
	{"service semp", "service.semp"},
	{"service virtual-hostname *", "service-virtual-hostname"},
	{"service web-transport", "service-web-transport"},
	{"snmp", "snmp"},
	{"snmp trap *", "snmp-trap"},
	{"ssl allow-tls-version", "ssl-allowed-tls"},
	{"ssl certificate-files", "ssl-certificate-files"},
	{"ssl cipher-suite-list default", "ssl-cipher-default"},
	{"ssl cipher-suite-list management", "ssl-cipher-management"},
	{"ssl cipher-suite-list msg-backbone", "ssl-cipher-msg-backbone"},
	{"ssl cipher-suite-list ssh", "ssl-cipher-ssh"},
	{"ssl server-certificate", "ssl-server-certificate"},
	{"ssl server-certificate detail", "ssl-server-certificate-detail"},
	{"syslog", "syslog"},
	{"system post", "system-post"},
	{"system detail", "system"},
	{"system health", "system-health"},
	{"telemetry", "telemetry"},
	{"topic-endpoint * detail", "topicendpoints"},
	{"username *", "username"},
	{"username * detail", "username-detail"},
	{"version", "version"},
	{"web-manager", "web-manager"},
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// Package k8s implements the Kubernetes platform of the solace-util CLI: the operator
// bundle, cluster/broker lifecycle over kubectl, and the concrete broker.Transport
// that wraps `kubectl exec/cp`. It is the Go port of the numbered bash scripts at
// the repo root (001-069, 010-020 deploy, 110-120 delete, and the operational
// helpers). Cluster/operator operations run over an engine.Runner directly; runtime
// broker operations run through internal/broker over the transport in transport.go.
package k8s

import "solace/internal/config"

// brokerSuffix is the operator's fixed name suffix for every broker resource
// (StatefulSets, pods, PVCs, the LB service): <name>-pubsubplus[...].
const brokerSuffix = "-pubsubplus"

// podName returns the broker pod for a role: <name>-pubsubplus-<p|b|m>-0. The
// operator names pods off the per-role StatefulSet's single replica (050:30,
// enter-solace-cli.sh:18). Broker pods are single-container, so no `-c` is ever used.
func podName(cfg *config.Config, role config.Role) string {
	return cfg.K8s.Name + brokerSuffix + "-" + role.Letter() + "-0"
}

// pvcName returns the PersistentVolumeClaim backing a role's pod:
// data-<name>-pubsubplus-<role>-0 (120:65-69).
func pvcName(cfg *config.Config, role config.Role) string {
	return "data-" + cfg.K8s.Name + brokerSuffix + "-" + role.Letter() + "-0"
}

// stsName returns the StatefulSet for a role: <name>-pubsubplus-<role> (no `-0`
// suffix, unlike the pod/PVC) (replicas-start-broker.sh:17,24,29).
func stsName(cfg *config.Config, role config.Role) string {
	return cfg.K8s.Name + brokerSuffix + "-" + role.Letter()
}

// lbServiceName returns the load-balancer service: <name>-pubsubplus, with no role
// suffix -- one service fronts the active broker (desc-lb.sh:16).
func lbServiceName(cfg *config.Config) string {
	return cfg.K8s.Name + brokerSuffix
}

// HARoles returns the broker roles present in this deployment: [primary] for a
// standalone broker, [primary, backup, monitor] for a redundancy group. It bounds
// the per-role resource operations (PVC deletion, replica scaling, diagnostics).
func HARoles(cfg *config.Config) []config.Role {
	if cfg.RedundancyEnabled() {
		return []config.Role{config.Primary, config.Backup, config.Monitor}
	}
	return []config.Role{config.Primary}
}

// RestartOrder returns the roles in the order a manual pod bounce should follow:
// monitor first, then backup, then primary -- least message-routing impact first,
// with the node most likely to be serving traffic left for last. Standalone has
// only the one broker.
//
// The order is by configured role, not by which node is currently active: after a
// failover the config's "primary" may be the standby. Check `verify redundancy`
// first, or restart roles one at a time in the order you want.
func RestartOrder(cfg *config.Config) []config.Role {
	if cfg.RedundancyEnabled() {
		return []config.Role{config.Monitor, config.Backup, config.Primary}
	}
	return []config.Role{config.Primary}
}

// ProductKeyRoles returns the roles a product key is applied to: [primary] for a
// standalone broker, [primary, backup] for a redundancy group -- never the monitor,
// which carries no message spool (057).
func ProductKeyRoles(cfg *config.Config) []config.Role {
	if cfg.RedundancyEnabled() {
		return []config.Role{config.Primary, config.Backup}
	}
	return []config.Role{config.Primary}
}

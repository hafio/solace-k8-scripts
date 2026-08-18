package config

import "fmt"

// Role is a broker node role. The single-letter forms (p|b|m) appear in k8s pod
// names; the long forms are the CLI-facing positional args.
type Role string

const (
	Primary Role = "p"
	Backup  Role = "b"
	Monitor Role = "m"
)

// ParseRole normalizes a role argument to its single-letter form, porting
// pick_pod. Accepts p|primary, b|backup, m|monitor; empty defaults to primary.
func ParseRole(s string) (Role, error) {
	switch s {
	case "p", "primary":
		return Primary, nil
	case "b", "backup":
		return Backup, nil
	case "m", "monitor":
		return Monitor, nil
	case "":
		return Primary, nil
	default:
		return "", fmt.Errorf("invalid node role %q (expected p|b|m or primary|backup|monitor)", s)
	}
}

// RoleNames returns the long role names in redundancy order, for completing the
// [role] positionals and --pod. ParseRole stays the only validator -- the p|b|m
// forms it also accepts are not worth suggesting, and a suggestion list that
// drifted from it is pinned by TestRoleNamesParse.
func RoleNames() []string { return []string{"primary", "backup", "monitor"} }

// Letter returns the single-letter role, matching pod-name suffixes.
func (r Role) Letter() string { return string(r) }

// NodeIdentity is a host's resolved Solace identity for container deployment,
// porting node_env/resolve_node's THIS_HOSTNAME/THIS_NODETYPE/THIS_ACTIVESTANDBY.
type NodeIdentity struct {
	Hostname      string // routername
	NodeType      string // message_routing | monitoring
	ActiveStandby string // primary | backup | "" (monitor / standalone)
}

// ResolveNode returns this host's identity for the given role, honoring
// redundancy mode. In HA (redundancy: yes) the role selects from the node table;
// in standalone (redundancy: no) there is one message_routing node named after
// the primary, with no active/standby role, and the role arg is ignored.
func (c *Config) ResolveNode(role Role) NodeIdentity {
	if !c.RedundancyEnabled() {
		return NodeIdentity{Hostname: c.Nodes.Primary.Name, NodeType: "message_routing"}
	}
	switch role {
	case Backup:
		return NodeIdentity{Hostname: c.Nodes.Backup.Name, NodeType: "message_routing", ActiveStandby: "backup"}
	case Monitor:
		return NodeIdentity{Hostname: c.Nodes.Monitor.Name, NodeType: "monitoring"}
	default:
		return NodeIdentity{Hostname: c.Nodes.Primary.Name, NodeType: "message_routing", ActiveStandby: "primary"}
	}
}

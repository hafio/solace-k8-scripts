package broker

import (
	"context"
	"fmt"
	"os"
	"strings"

	"solace/internal/config"
)

// This file holds the node-local HA state machines used by the container
// platform, where the transport talks only to the single broker on THIS host and
// there is no cross-node control point. They are the node-local counterparts of
// verify_ops.go's Leader/Redundancy, which drive both nodes from one control point
// (showRDPair) and therefore only fit the k8s model. Each function here polls the
// single local `show redundancy` output and performs its half of a two-host
// handshake -- the operator runs the matching command on each host. They reuse the
// package's unexported helpers (showRD, activity, field, rdEnabledUp,
// primaryRedundancyUp, poll, RunCLI) and script builders unchanged.

// LocalRole resolves which redundancy role THIS host plays. An explicit roleArg
// (primary|backup|monitor or p|b|m) wins; otherwise it detects the role by
// matching the host's name against the configured node table (nodes.primary/
// backup/monitor .name), tolerating an FQDN-vs-short-name mismatch. It fails loud
// when detection is ambiguous so a mis-targeted HA operation never runs silently.
func (o *Ops) LocalRole(roleArg string) (config.Role, error) {
	if roleArg != "" {
		return config.ParseRole(roleArg)
	}
	host, err := o.hostname()
	if err != nil {
		return "", fmt.Errorf("detect node role: read hostname: %w", err)
	}
	for _, m := range []struct {
		name string
		role config.Role
	}{
		{o.Cfg.Nodes.Primary.Name, config.Primary},
		{o.Cfg.Nodes.Backup.Name, config.Backup},
		{o.Cfg.Nodes.Monitor.Name, config.Monitor},
	} {
		if m.name != "" && hostMatches(host, m.name) {
			return m.role, nil
		}
	}
	return "", fmt.Errorf("cannot determine node role from hostname %q; pass primary|backup|monitor explicitly", host)
}

// LeaderLocal asserts the config-sync leader from THIS host, which must be the
// primary (the user's spec: "assert leader should always be executed in the
// primary node"). HA-only. It fails loud on the backup/monitor rather than
// running, waits for local redundancy to be healthy, then runs assert-leader.
func (o *Ops) LeaderLocal(ctx context.Context, roleArg string) error {
	if o.skipIfStandalone("config leader") {
		return nil
	}
	role, err := o.LocalRole(roleArg)
	if err != nil {
		return err
	}
	if role != config.Primary {
		return fmt.Errorf("config leader must run on the primary node; this host is the %s node", roleName(role))
	}

	o.logf("Waiting for redundancy state to be restored fully...")
	if err := o.poll(ctx, "redundancy to be restored on Primary", func(ctx context.Context) (bool, error) {
		out, err := o.showRD(ctx, role)
		if err != nil {
			return false, err
		}
		return primaryRedundancyUp(out), nil
	}); err != nil {
		if detail, dErr := o.RunCLI(ctx, role, "show-redundancy-detail", showRedundancyDetailScript()); dErr == nil {
			o.show(detail)
		}
		return err
	}

	out, err := o.RunCLI(ctx, role, "assert-leader", assertLeaderScript())
	if err != nil {
		return err
	}
	o.show([]byte(lastLines(string(out), 12)))
	return nil
}

// RedundancyLocal exercises failover from THIS host, running the primary or backup
// half of the handshake per the user's spec. HA-only; the monitor is rejected
// loud. The primary releases (if active) then waits for the backup to fail back;
// the backup becomes active, dwells, and reverts to standby. The operator runs it
// on the primary and backup concurrently -- running only one side times out
// (bounded by PollAttempts) rather than hanging.
func (o *Ops) RedundancyLocal(ctx context.Context, roleArg string) error {
	if o.skipIfStandalone("verify redundancy") {
		return nil
	}
	role, err := o.LocalRole(roleArg)
	if err != nil {
		return err
	}
	if role == config.Monitor {
		return fmt.Errorf("verify redundancy cannot run on the monitor node; run it on the primary and backup nodes")
	}

	local, err := o.showRD(ctx, role)
	if err != nil {
		return err
	}
	if role == config.Primary {
		return o.redundancyLocalPrimary(ctx, role, local)
	}
	return o.redundancyLocalBackup(ctx, role, local)
}

// redundancyLocalPrimary is the primary half: if active, release activity to fall
// over to the backup (release + un-release, matching releaseToBackup, so the
// primary stays eligible to reclaim); then, active-start or standby-start alike,
// wait for the backup to fail back to this primary. Spec: "primary: if currently
// active, should release activity to fall over ... wait for backup to fail back
// ...; if currently standby, just wait for backup to fail back".
func (o *Ops) redundancyLocalPrimary(ctx context.Context, role config.Role, local string) error {
	if activity(local, activityLocalActive) == 1 {
		if !primaryRedundancyUp(local) {
			o.show([]byte(local))
			return fmt.Errorf("redundancy configuration/status is not healthy on the Primary")
		}
		o.logf("[Info] Primary node is active; releasing activity to fall over to the Backup.")
		if _, err := o.RunCLI(ctx, role, "release", releaseActivityScript()); err != nil {
			return err
		}
		if err := o.poll(ctx, "Primary to be released to the Backup", func(ctx context.Context) (bool, error) {
			out, err := o.showRD(ctx, role)
			if err != nil {
				return false, err
			}
			return field(out, labelConfigStatus) == "Enabled-Released" &&
				field(out, labelRedundancyStatus) == "Down" &&
				activity(out, activityMateActive) == 1, nil
		}); err != nil {
			return err
		}
		o.logf("[Info] Primary node is released. Backup node is active.")

		if _, err := o.RunCLI(ctx, role, "no-release", noReleaseActivityScript()); err != nil {
			return err
		}
		if err := o.poll(ctx, "Primary to be un-released", func(ctx context.Context) (bool, error) {
			out, err := o.showRD(ctx, role)
			if err != nil {
				return false, err
			}
			return rdEnabledUp(out) && activity(out, activityMateActive) == 1, nil
		}); err != nil {
			return err
		}
		o.logf("[Info] Primary node is un-released. Waiting for the Backup to fail back...")
	} else {
		o.logf("[Info] Primary node is standby; waiting for the Backup to fail back...")
	}

	if err := o.poll(ctx, "Backup to fail back to the Primary", func(ctx context.Context) (bool, error) {
		out, err := o.showRD(ctx, role)
		if err != nil {
			return false, err
		}
		return rdEnabledUp(out) &&
			activity(out, activityLocalActive) == 1 &&
			activity(out, activityMateActive) == 0, nil
	}); err != nil {
		return err
	}
	o.logf("[Info] Backup failed back to the Primary successfully.")
	return nil
}

// redundancyLocalBackup is the backup half: if already active, revert activity to
// the primary and wait for standby; if inactive, wait to become active, hold for
// ActiveDwell (the spec's "after 10s of being active"), then revert and wait for
// standby. Spec: "backup: if currently inactive, wait to become active, and then
// execute revert activity after 10s ...; if currently active, revert activity to
// primary and declare success".
func (o *Ops) redundancyLocalBackup(ctx context.Context, role config.Role, local string) error {
	if activity(local, activityLocalActive) != 1 {
		o.logf("[Info] Backup node is inactive; waiting to become active...")
		if err := o.poll(ctx, "Backup to become active", func(ctx context.Context) (bool, error) {
			out, err := o.showRD(ctx, role)
			if err != nil {
				return false, err
			}
			return activity(out, activityLocalActive) == 1, nil
		}); err != nil {
			return err
		}
		o.logf("[Info] Backup node is active; holding for %s before reverting activity.", o.ActiveDwell)
		if err := o.dwell(ctx); err != nil {
			return err
		}
	} else {
		o.logf("[Info] Backup node is active; reverting activity to the Primary.")
	}

	if _, err := o.RunCLI(ctx, role, "revert-activity", revertActivityConfigureScript()); err != nil {
		return err
	}
	if err := o.poll(ctx, "Backup to return to standby", func(ctx context.Context) (bool, error) {
		out, err := o.showRD(ctx, role)
		if err != nil {
			return false, err
		}
		return activity(out, activityLocalActive) == 0, nil
	}); err != nil {
		return err
	}
	o.logf("[Info] Backup node returned to standby successfully.")
	return nil
}

// hostname reads this host's name via the injected Hostname func, defaulting to
// os.Hostname when unset (New sets it; a directly-constructed Ops in tests injects
// a fixed value). The result is trimmed so a trailing newline never defeats a match.
func (o *Ops) hostname() (string, error) {
	fn := o.Hostname
	if fn == nil {
		fn = os.Hostname
	}
	h, err := fn()
	return strings.TrimSpace(h), err
}

// hostMatches reports whether host names the configured node, tolerating an FQDN
// on either side (pri.example.com matches pri and vice versa).
func hostMatches(host, name string) bool {
	return host == name || shortHost(host) == shortHost(name)
}

// shortHost is the label before the first dot of an FQDN (the host itself if none).
func shortHost(h string) string {
	if i := strings.Index(h, "."); i >= 0 {
		return h[:i]
	}
	return h
}

// roleName is the long-form role name for actionable HA guard errors.
func roleName(r config.Role) string {
	switch r {
	case config.Backup:
		return "backup"
	case config.Monitor:
		return "monitor"
	default:
		return "primary"
	}
}

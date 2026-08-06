package broker

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"solace/internal/config"
)

// Field labels and activity states parsed out of `show redundancy` output. They
// are named once here so the leader/redundancy state machines compare against a
// single source of truth -- a typo in a copied literal would silently break a
// state check.
const (
	labelConfigStatus      = "Configuration Status"
	labelRedundancyStatus  = "Redundancy Status"
	labelActiveStandbyRole = "Active-Standby Role"
	labelADBLink           = "ADB Link To Mate"
	labelADBHello          = "ADB Hello To Mate"
	labelActivityStatus    = "Activity Status"

	activityLocalActive = "Local Active"
	activityMateActive  = "Mate Active"

	cliShowRD        = "show-rd"
	cliGatherConfigs = "gather-configs"
)

// Login tests a SEMP login against the node, porting 060. The credentials ride a
// curl config on stdin (curl -K -), so the password never appears in an argv or
// a --dry-run echo (§3). It reports success and writes an outcome line to Out.
func (o *Ops) Login(ctx context.Context, role config.Role, user, pass string) (bool, error) {
	cfg := fmt.Sprintf("user = %q\n", user+":"+pass)
	out, err := o.T.OutputInput(ctx, role, []byte(cfg),
		"curl", "-is", "-K", "-", "http://localhost:8080/SEMP/v2/monitor")
	if err != nil {
		return false, fmt.Errorf("SEMP request failed: %w", err)
	}
	lines := httpStatusLines(string(out))
	for _, l := range lines {
		if isHTTP2xx(l) {
			fmt.Fprintln(o.out(), "Login OK")
			return true, nil
		}
	}
	status := "<no HTTP response from broker>"
	if len(lines) > 0 {
		status = lines[len(lines)-1]
	}
	fmt.Fprintf(o.out(), "Login failed. Reason: %s\n", status)
	return false, nil
}

// Leader restores redundancy and asserts the Primary as config-sync leader for
// the router and all VPNs, porting 050. HA-only: it no-ops for standalone.
func (o *Ops) Leader(ctx context.Context) error {
	if o.skipIfStandalone("config leader") {
		return nil
	}

	// Revert any released activity on the Backup first (050 lines 23-31).
	if _, err := o.RunCLI(ctx, config.Backup, "revert-activity", revertActivityScript()); err != nil {
		return err
	}

	o.logf("Waiting for redundancy state to be restored fully...")
	err := o.poll(ctx, "redundancy to be restored on Primary", func(ctx context.Context) (bool, error) {
		out, err := o.showRD(ctx, config.Primary)
		if err != nil {
			return false, err
		}
		return primaryRedundancyUp(out), nil
	})
	if err != nil {
		if detail, dErr := o.RunCLI(ctx, config.Primary, "show-redundancy-detail", showRedundancyDetailScript()); dErr == nil {
			o.show(detail)
		}
		return err
	}

	out, err := o.RunCLI(ctx, config.Primary, "assert-leader", assertLeaderScript())
	if err != nil {
		return err
	}
	o.show([]byte(lastLines(string(out), 12)))
	return nil
}

// Redundancy exercises failover, porting 061: confirm the Primary is active,
// release activity to the Backup, un-release, then revert back to the Primary.
// HA-only: it no-ops for standalone.
func (o *Ops) Redundancy(ctx context.Context) error {
	if o.skipIfStandalone("verify redundancy") {
		return nil
	}

	pri, err := o.showRD(ctx, config.Primary)
	if err != nil {
		return err
	}
	if !primaryRedundancyUp(pri) {
		o.show([]byte(pri))
		return fmt.Errorf("redundancy configuration/status is not healthy on the Primary")
	}

	// If the Primary is active, walk it through release -> un-release so the
	// Backup takes over and hands back cleanly.
	if activity(pri, activityLocalActive) == 1 {
		o.logf("[Info] Detected Primary node is active.")
		if err := o.releaseToBackup(ctx); err != nil {
			return err
		}
	}

	// Revert activity from the Backup back to the Primary.
	bk, err := o.showRD(ctx, config.Backup)
	if err != nil {
		return err
	}
	if activity(bk, activityLocalActive) != 1 {
		o.show([]byte(bk))
		return fmt.Errorf("neither Primary nor Backup appears to be active")
	}
	o.logf("[Info] Detected Backup node is active.")
	if err := o.revertToPrimary(ctx); err != nil {
		return err
	}
	o.logf("[Info] Reverted back to Primary node successfully.")
	return nil
}

// releaseToBackup releases activity from the Primary and un-releases it, leaving
// the Backup active (061 release / no-release steps).
func (o *Ops) releaseToBackup(ctx context.Context) error {
	if _, err := o.RunCLI(ctx, config.Primary, "release", releaseActivityScript()); err != nil {
		return err
	}
	if err := o.poll(ctx, "Primary to be released", func(ctx context.Context) (bool, error) {
		out, err := o.showRD(ctx, config.Primary)
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

	if _, err := o.RunCLI(ctx, config.Primary, "no-release", noReleaseActivityScript()); err != nil {
		return err
	}
	if err := o.poll(ctx, "Primary to be un-released", func(ctx context.Context) (bool, error) {
		p, b, err := o.showRDPair(ctx)
		if err != nil {
			return false, err
		}
		return rdEnabledUp(p) &&
			activity(p, activityMateActive) == 1 &&
			activity(b, activityLocalActive) == 1, nil
	}); err != nil {
		return err
	}
	o.logf("[Info] Primary node is un-released. Backup node is active.")
	return nil
}

// revertToPrimary reverts activity from the Backup and waits for the Primary to
// become the sole active node (061 revert step).
func (o *Ops) revertToPrimary(ctx context.Context) error {
	if _, err := o.RunCLI(ctx, config.Backup, "revert-activity", revertActivityConfigureScript()); err != nil {
		return err
	}
	return o.poll(ctx, "Primary to become active", func(ctx context.Context) (bool, error) {
		p, b, err := o.showRDPair(ctx)
		if err != nil {
			return false, err
		}
		return rdEnabledUp(p) &&
			activity(p, activityMateActive) == 0 &&
			activity(p, activityLocalActive) == 1 &&
			activity(b, activityLocalActive) == 0, nil
	})
}

// showRD runs `show redundancy` on role and returns its output as a string.
func (o *Ops) showRD(ctx context.Context, role config.Role) (string, error) {
	out, err := o.RunCLI(ctx, role, cliShowRD, showRedundancyScript())
	return string(out), err
}

// showRDPair fetches `show redundancy` from both the Primary and Backup, used by
// the cross-node poll conditions.
func (o *Ops) showRDPair(ctx context.Context) (primary, backup string, err error) {
	if primary, err = o.showRD(ctx, config.Primary); err != nil {
		return "", "", err
	}
	backup, err = o.showRD(ctx, config.Backup)
	return primary, backup, err
}

// Diagnostics runs the show-command sweep and gather-diagnostics on each of
// roles, then downloads the zipped output and the diagnostics bundle into
// destDir, porting 069. ts is the timestamp stamped into the local zip name.
func (o *Ops) Diagnostics(ctx context.Context, destDir, ts string, days int, roles ...config.Role) error {
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return fmt.Errorf("create diagnostics dir %q: %w", destDir, err)
	}
	for _, role := range roles {
		if err := o.gatherNode(ctx, role, destDir, ts, days); err != nil {
			return err
		}
	}
	o.logf("Diagnostics written to %s", destDir)
	return nil
}

// gatherNode runs the diagnostics collection for a single node and pulls the
// resulting archives back to destDir.
func (o *Ops) gatherNode(ctx context.Context, role config.Role, destDir, ts string, days int) error {
	o.logf("Gathering diagnostics for %q node...", role)
	zipPath := JailRoot + "/zip-configs.sh"
	if err := o.T.Run(ctx, role, "mkdir", "-p", JailRoot+"/configs/cliout"); err != nil {
		return err
	}
	if err := o.T.Upload(ctx, role, []byte(gatherConfigsScript(days)), cliScriptPath(cliGatherConfigs)); err != nil {
		return err
	}
	if err := o.T.Upload(ctx, role, []byte(zipConfigsScript()), zipPath); err != nil {
		return err
	}

	out, err := o.T.Output(ctx, role, CLIBinary, "-Apes", cliArg(cliGatherConfigs))
	if err != nil {
		return err
	}
	if err := o.T.Run(ctx, role, "bash", zipPath); err != nil {
		return err
	}

	hostBytes, err := o.T.Output(ctx, role, "hostname")
	if err != nil {
		return err
	}
	host := strings.TrimSpace(string(hostBytes))
	zipName := fmt.Sprintf("gather-configs-%s-%s.zip", host, ts)
	if err := o.T.Download(ctx, role, JailRoot+"/gather-configs.zip", filepath.Join(destDir, zipName)); err != nil {
		return err
	}
	if err := o.T.Run(ctx, role, "rm", "-rf", JailRoot+"/cli-out", JailRoot+"/gather-configs.zip"); err != nil {
		return err
	}

	if diag := field(string(out), "Diagnostics saved"); diag != "" {
		local := strings.TrimPrefix(diag, "logs/")
		if err := o.T.Download(ctx, role, JailRoot+"/"+diag, filepath.Join(destDir, local)); err != nil {
			o.logf("[WARN] failed to download diagnostics bundle %q: %v", diag, err)
		}
		if err := o.T.Run(ctx, role, "rm", "-rf", JailRoot+"/"+diag, cliScriptPath(cliGatherConfigs), zipPath); err != nil {
			o.logf("[WARN] failed to clean up diagnostics artifacts on %q: %v", role, err)
		}
	}
	return nil
}

// poll runs cond up to PollAttempts times, sleeping PollInterval between tries,
// and returns a timeout error if cond never reports true. It replaces the bash
// scripts' unbounded busy-wait loops with a bounded ceiling.
func (o *Ops) poll(ctx context.Context, desc string, cond func(context.Context) (bool, error)) error {
	for i := 0; i < o.PollAttempts; i++ {
		ok, err := cond(ctx)
		if err != nil {
			return err
		}
		if ok {
			return nil
		}
		if err := o.sleep(ctx); err != nil {
			return err
		}
	}
	return fmt.Errorf("timeout waiting for %s", desc)
}

// rdEnabledUp reports whether `show redundancy` output shows an enabled, up node.
func rdEnabledUp(out string) bool {
	return field(out, labelConfigStatus) == "Enabled" && field(out, labelRedundancyStatus) == "Up"
}

// activity counts `Activity Status` lines matching a given state (Local/Mate Active).
func activity(out, state string) int { return countContains(out, labelActivityStatus, state) }

// primaryRedundancyUp reports whether `show redundancy` output describes a
// healthy, active Primary, porting the field checks of 050/061.
func primaryRedundancyUp(out string) bool {
	return rdEnabledUp(out) &&
		field(out, labelActiveStandbyRole) == "Primary" &&
		field(out, labelADBLink) == "Up" &&
		field(out, labelADBHello) == "Up"
}

// httpStatusLines returns every "HTTP/..." status line (CR stripped) from a raw
// HTTP response. Login treats the request as successful if any of them is 2xx,
// matching the bash grep-then-regex check in 060, which tolerates a preceding
// "100 Continue" or a redirect line.
func httpStatusLines(out string) []string {
	var lines []string
	for _, raw := range strings.Split(out, "\n") {
		line := strings.TrimRight(raw, "\r")
		if strings.HasPrefix(strings.ToUpper(line), "HTTP/") {
			lines = append(lines, line)
		}
	}
	return lines
}

// isHTTP2xx reports whether an HTTP status line carries a 2xx code, matching the
// `\ 2[0-9][0-9]( |$)` check in 060.
func isHTTP2xx(status string) bool {
	fields := strings.Fields(status)
	if len(fields) < 2 {
		return false
	}
	code := fields[1]
	return len(code) == 3 && code[0] == '2' &&
		code[1] >= '0' && code[1] <= '9' && code[2] >= '0' && code[2] <= '9'
}

// lastLines returns the last n lines of s (its trailing newline preserved),
// porting the `| tail -12` display trim of 050.
func lastLines(s string, n int) string {
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return strings.Join(lines, "\n") + "\n"
}

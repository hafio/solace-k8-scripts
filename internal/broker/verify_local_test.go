package broker

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"solace/internal/config"
)

// These tests exercise the node-local HA state machines (LocalRole, LeaderLocal,
// RedundancyLocal) over the shared fakeTransport from broker_test.go. The
// transport is node-local, so a scripted `show redundancy` sequence stands in for
// this host's view across successive poll iterations. PollInterval and ActiveDwell
// are 0 (newTestOps leaves both at their zero value), so nothing sleeps.

// localCfg is a redundancy-group config with a named node table for role detection.
func localCfg(redundancy string) *config.Config {
	return &config.Config{
		Redundancy: redundancy,
		Nodes: config.Nodes{
			Primary: config.Node{Name: "pri-host"},
			Backup:  config.Node{Name: "bkp-host"},
			Monitor: config.Node{Name: "mon-host"},
		},
	}
}

// newLocalOps builds an Ops over ft with a fixed detected hostname.
func newLocalOps(t *testing.T, redundancy, host string, ft *fakeTransport) (*Ops, *bytes.Buffer) {
	t.Helper()
	o, buf := newTestOps(t, localCfg(redundancy), ft)
	o.Hostname = func() (string, error) { return host, nil }
	return o, buf
}

// rd renders a minimal `show redundancy` output for the given fields.
func rd(role, cfgStatus, rdStatus, activity string) string {
	return "Configuration Status : " + cfgStatus +
		"\nRedundancy Status : " + rdStatus +
		"\nActive-Standby Role : " + role +
		"\nADB Link To Mate : Up\nADB Hello To Mate : Up" +
		"\nActivity Status : " + activity + "\n"
}

// seqTransport answers show-rd with seq[i] (advancing i), and nil for every other
// CLI exec (release/no-release/revert-activity). It returns a pointer to the
// consumed count so a test can assert the whole sequence was walked. Once the
// sequence is exhausted it repeats the last entry rather than panicking, so a
// stuck poll surfaces as a wrong consumed-count instead of an index panic.
func seqTransport(seq []string) (*fakeTransport, *int) {
	i := 0
	ft := &fakeTransport{}
	ft.responder = func(_ config.Role, argv []string, _ []byte) ([]byte, error) {
		if !matchCLI(argv, "show-rd") {
			return nil, nil
		}
		out := seq[len(seq)-1]
		if i < len(seq) {
			out = seq[i]
			i++
		}
		return []byte(out), nil
	}
	return ft, &i
}

// --- LocalRole -------------------------------------------------------------

func TestLocalRole(t *testing.T) {
	cases := []struct {
		arg, host string
		want      config.Role
		wantErr   bool
	}{
		{"backup", "anything", config.Backup, false},   // explicit arg wins
		{"m", "pri-host", config.Monitor, false},        // explicit short form wins over host
		{"", "pri-host", config.Primary, false},         // detect primary
		{"", "bkp-host", config.Backup, false},          // detect backup
		{"", "mon-host.example.com", config.Monitor, false}, // FQDN vs short name
		{"", "stranger", "", true},                      // no match -> loud error
		{"nonsense", "pri-host", "", true},              // bad explicit arg
	}
	for _, tc := range cases {
		o, _ := newLocalOps(t, "yes", tc.host, &fakeTransport{})
		got, err := o.LocalRole(tc.arg)
		if tc.wantErr {
			if err == nil {
				t.Errorf("LocalRole(arg=%q host=%q) = %q, want error", tc.arg, tc.host, got)
			}
			continue
		}
		if err != nil || got != tc.want {
			t.Errorf("LocalRole(arg=%q host=%q) = %q,%v want %q", tc.arg, tc.host, got, err, tc.want)
		}
	}
}

// --- LeaderLocal -----------------------------------------------------------

func TestLeaderLocalStandaloneSkips(t *testing.T) {
	ft := &fakeTransport{}
	o, _ := newLocalOps(t, "no", "pri-host", ft)
	if err := o.LeaderLocal(context.Background(), ""); err != nil {
		t.Fatalf("LeaderLocal standalone error: %v", err)
	}
	if len(ft.outputs) != 0 || len(ft.uploads) != 0 {
		t.Error("LeaderLocal must make no calls in standalone mode")
	}
}

func TestLeaderLocalRejectsNonPrimary(t *testing.T) {
	for _, host := range []string{"bkp-host", "mon-host"} {
		ft := &fakeTransport{}
		o, _ := newLocalOps(t, "yes", host, ft)
		if err := o.LeaderLocal(context.Background(), ""); err == nil {
			t.Errorf("LeaderLocal on %q should fail loud", host)
		}
		if len(ft.uploads) != 0 {
			t.Errorf("LeaderLocal on %q must not upload before the guard", host)
		}
	}
	// Explicit backup arg is rejected the same way.
	ft := &fakeTransport{}
	o, _ := newLocalOps(t, "yes", "pri-host", ft)
	if err := o.LeaderLocal(context.Background(), "backup"); err == nil {
		t.Error("LeaderLocal with explicit backup arg should fail loud")
	}
}

func TestLeaderLocalSuccess(t *testing.T) {
	healthy := rd("Primary", "Enabled", "Up", "Local Active")
	ft := &fakeTransport{responder: func(_ config.Role, argv []string, _ []byte) ([]byte, error) {
		switch {
		case matchCLI(argv, "show-rd"):
			return []byte(healthy), nil
		case matchCLI(argv, "assert-leader"):
			return []byte("l1\nl2\nSync Complete\n"), nil
		}
		return nil, nil
	}}
	o, buf := newLocalOps(t, "yes", "pri-host", ft)
	if err := o.LeaderLocal(context.Background(), ""); err != nil {
		t.Fatalf("LeaderLocal error: %v", err)
	}
	if !uploadedForRole(ft, config.Primary, cliScriptPath("assert-leader")) {
		t.Error("LeaderLocal should assert leadership on the primary")
	}
	if !strings.Contains(buf.String(), "Sync Complete") {
		t.Errorf("LeaderLocal output = %q", buf.String())
	}
}

func TestLeaderLocalTimeoutDumpsDetail(t *testing.T) {
	down := rd("Primary", "Enabled", "Down", "Local Active")
	sawDetail := false
	ft := &fakeTransport{responder: func(_ config.Role, argv []string, _ []byte) ([]byte, error) {
		if matchCLI(argv, "show-redundancy-detail") {
			sawDetail = true
			return []byte("detail dump\n"), nil
		}
		return []byte(down), nil
	}}
	o, _ := newLocalOps(t, "yes", "pri-host", ft)
	o.PollAttempts = 2
	if err := o.LeaderLocal(context.Background(), ""); err == nil {
		t.Error("LeaderLocal should time out when redundancy never recovers")
	}
	if !sawDetail {
		t.Error("LeaderLocal should dump show-redundancy-detail on timeout")
	}
}

// --- RedundancyLocal guards ------------------------------------------------

func TestRedundancyLocalStandaloneSkips(t *testing.T) {
	ft := &fakeTransport{}
	o, _ := newLocalOps(t, "no", "pri-host", ft)
	if err := o.RedundancyLocal(context.Background(), ""); err != nil {
		t.Fatalf("RedundancyLocal standalone error: %v", err)
	}
	if len(ft.outputs) != 0 {
		t.Error("RedundancyLocal must make no calls in standalone mode")
	}
}

func TestRedundancyLocalRejectsMonitor(t *testing.T) {
	ft := &fakeTransport{}
	o, _ := newLocalOps(t, "yes", "mon-host", ft)
	if err := o.RedundancyLocal(context.Background(), ""); err == nil {
		t.Error("RedundancyLocal on the monitor should fail loud")
	}
	if len(ft.outputs) != 0 {
		t.Error("RedundancyLocal must reject the monitor before any show redundancy")
	}
}

// --- RedundancyLocal primary half ------------------------------------------

func TestRedundancyLocalPrimaryActive(t *testing.T) {
	seq := []string{
		rd("Primary", "Enabled", "Up", "Local Active"),            // initial: active + healthy
		rd("Primary", "Enabled-Released", "Down", "Mate Active"),  // released
		rd("Primary", "Enabled", "Up", "Mate Active"),             // un-released, backup active
		rd("Primary", "Enabled", "Up", "Local Active"),            // backup failed back
	}
	ft, consumed := seqTransport(seq)
	o, _ := newLocalOps(t, "yes", "pri-host", ft)
	if err := o.RedundancyLocal(context.Background(), ""); err != nil {
		t.Fatalf("RedundancyLocal primary-active error: %v", err)
	}
	if *consumed != len(seq) {
		t.Errorf("primary-active consumed %d/%d show-rd reads", *consumed, len(seq))
	}
	if !uploadedForRole(ft, config.Primary, cliScriptPath("release")) ||
		!uploadedForRole(ft, config.Primary, cliScriptPath("no-release")) {
		t.Error("primary-active should release then un-release activity")
	}
}

func TestRedundancyLocalPrimaryStandby(t *testing.T) {
	seq := []string{
		rd("Primary", "Enabled", "Up", "Mate Active"),  // initial: standby (mate active)
		rd("Primary", "Enabled", "Up", "Local Active"), // backup failed back
	}
	ft, consumed := seqTransport(seq)
	o, _ := newLocalOps(t, "yes", "pri-host", ft)
	if err := o.RedundancyLocal(context.Background(), "primary"); err != nil {
		t.Fatalf("RedundancyLocal primary-standby error: %v", err)
	}
	if *consumed != len(seq) {
		t.Errorf("primary-standby consumed %d/%d show-rd reads", *consumed, len(seq))
	}
	if ft.hasUpload(cliScriptPath("release")) {
		t.Error("primary-standby must not release activity (it only waits for fail-back)")
	}
}

// --- RedundancyLocal backup half -------------------------------------------

func TestRedundancyLocalBackupActive(t *testing.T) {
	seq := []string{
		rd("Backup", "Enabled", "Up", "Local Active"), // initial: active
		rd("Backup", "Enabled", "Up", "Mate Active"),  // reverted to standby
	}
	ft, consumed := seqTransport(seq)
	o, _ := newLocalOps(t, "yes", "bkp-host", ft)
	if err := o.RedundancyLocal(context.Background(), ""); err != nil {
		t.Fatalf("RedundancyLocal backup-active error: %v", err)
	}
	if *consumed != len(seq) {
		t.Errorf("backup-active consumed %d/%d show-rd reads", *consumed, len(seq))
	}
	if !uploadedForRole(ft, config.Backup, cliScriptPath("revert-activity")) {
		t.Error("backup-active should revert activity to the primary")
	}
}

func TestRedundancyLocalBackupInactive(t *testing.T) {
	seq := []string{
		rd("Backup", "Enabled", "Up", "Mate Active"),  // initial: inactive (mate active)
		rd("Backup", "Enabled", "Up", "Local Active"), // became active
		rd("Backup", "Enabled", "Up", "Mate Active"),  // reverted to standby
	}
	ft, consumed := seqTransport(seq)
	o, _ := newLocalOps(t, "yes", "bkp-host", ft)
	o.ActiveDwell = 0 // no dwell in tests
	if err := o.RedundancyLocal(context.Background(), "b"); err != nil {
		t.Fatalf("RedundancyLocal backup-inactive error: %v", err)
	}
	if *consumed != len(seq) {
		t.Errorf("backup-inactive consumed %d/%d show-rd reads", *consumed, len(seq))
	}
	if !uploadedForRole(ft, config.Backup, cliScriptPath("revert-activity")) {
		t.Error("backup-inactive should revert activity after becoming active")
	}
}

// --- RedundancyLocal error / timeout branches ------------------------------

func TestRedundancyLocalInitialShowError(t *testing.T) {
	ft := &fakeTransport{responder: func(_ config.Role, argv []string, _ []byte) ([]byte, error) {
		if matchCLI(argv, "show-rd") {
			return nil, fmt.Errorf("cli unreachable")
		}
		return nil, nil
	}}
	o, _ := newLocalOps(t, "yes", "pri-host", ft)
	if err := o.RedundancyLocal(context.Background(), ""); err == nil {
		t.Error("RedundancyLocal should propagate an initial show redundancy error")
	}
}

func TestRedundancyLocalBadRoleArg(t *testing.T) {
	o, _ := newLocalOps(t, "yes", "pri-host", &fakeTransport{})
	if err := o.RedundancyLocal(context.Background(), "nonsense"); err == nil {
		t.Error("RedundancyLocal should propagate a bad explicit role arg")
	}
}

func TestRedundancyLocalPrimaryUnhealthy(t *testing.T) {
	seq := []string{rd("Primary", "Enabled", "Down", "Local Active")} // active but redundancy Down
	ft, _ := seqTransport(seq)
	o, _ := newLocalOps(t, "yes", "pri-host", ft)
	if err := o.RedundancyLocal(context.Background(), ""); err == nil {
		t.Error("RedundancyLocal primary should fail loud when redundancy is not healthy")
	}
	if ft.hasUpload(cliScriptPath("release")) {
		t.Error("primary must not release activity when redundancy is unhealthy")
	}
}

func TestRedundancyLocalPrimaryFailBackTimeout(t *testing.T) {
	// Standby primary that never sees the backup fail back: seqTransport repeats the
	// last (Mate Active) reading, so the fail-back poll never satisfies.
	seq := []string{rd("Primary", "Enabled", "Up", "Mate Active")}
	ft, _ := seqTransport(seq)
	o, _ := newLocalOps(t, "yes", "pri-host", ft)
	o.PollAttempts = 2
	if err := o.RedundancyLocal(context.Background(), "primary"); err == nil {
		t.Error("RedundancyLocal primary should time out when the backup never fails back")
	}
	if ft.hasUpload(cliScriptPath("release")) {
		t.Error("standby primary must not release activity (it only waits for fail-back)")
	}
}

// --- pure / detection arms -------------------------------------------------

func TestRoleName(t *testing.T) {
	cases := map[config.Role]string{
		config.Primary: "primary",
		config.Backup:  "backup",
		config.Monitor: "monitor",
	}
	for r, want := range cases {
		if got := roleName(r); got != want {
			t.Errorf("roleName(%v) = %q, want %q", r, got, want)
		}
	}
}

func TestLocalRoleDefaultHostname(t *testing.T) {
	o, _ := newTestOps(t, localCfg("yes"), &fakeTransport{})
	o.Hostname = nil // force the os.Hostname default
	// The real host name will not match the pri/bkp/mon table, so detection fails loud.
	if _, err := o.LocalRole(""); err == nil {
		t.Error("LocalRole with the default os.Hostname should fail loud off the node table")
	}
}

// --- LocalRole hostname error ------------------------------------------------

// TestLocalRoleHostnameError closes LocalRole's hostname-read error branch: if
// the injected Hostname func fails, LocalRole must wrap and return the error
// rather than falling through to the node-table match against a garbage/empty
// host.
func TestLocalRoleHostnameError(t *testing.T) {
	o, _ := newTestOps(t, localCfg("yes"), &fakeTransport{})
	o.Hostname = func() (string, error) { return "", errors.New("boom") }
	_, err := o.LocalRole("")
	if err == nil || !strings.Contains(err.Error(), "detect node role") {
		t.Errorf("LocalRole hostname error = %v, want wrapped %q", err, "detect node role")
	}
}

// --- LeaderLocal error branches -----------------------------------------------

// TestLeaderLocalBadRoleArg mirrors TestRedundancyLocalBadRoleArg: an invalid
// explicit role arg must stop LeaderLocal before the primary-only guard even
// runs. LeaderLocal had no equivalent test before this.
func TestLeaderLocalBadRoleArg(t *testing.T) {
	ft := &fakeTransport{}
	o, _ := newLocalOps(t, "yes", "pri-host", ft)
	if err := o.LeaderLocal(context.Background(), "nonsense"); err == nil {
		t.Error("LeaderLocal should propagate a bad explicit role arg")
	}
	if len(ft.outputs) != 0 || len(ft.uploads) != 0 {
		t.Error("LeaderLocal must make no transport calls on a bad role arg")
	}
}

// TestLeaderLocalPollCondError mirrors the k8s TestLeaderPollCondError,
// closing an asymmetry between the two implementations: a transport error
// mid-poll must abort immediately (the detail dump still runs), not be
// silently retried as "not yet healthy".
func TestLeaderLocalPollCondError(t *testing.T) {
	sawDetail := false
	ft := &fakeTransport{responder: func(_ config.Role, argv []string, _ []byte) ([]byte, error) {
		switch {
		case matchCLI(argv, "show-rd"):
			return nil, errors.New("show boom")
		case matchCLI(argv, "show-redundancy-detail"):
			sawDetail = true
			return []byte("detail\n"), nil
		}
		return nil, nil
	}}
	o, _ := newLocalOps(t, "yes", "pri-host", ft)
	if err := o.LeaderLocal(context.Background(), ""); err == nil {
		t.Error("LeaderLocal should return the poll condition error")
	}
	if !sawDetail {
		t.Error("LeaderLocal should still dump show-redundancy-detail after the error")
	}
}

// TestLeaderLocalAssertLeaderError closes LeaderLocal's final assert-leader
// error branch: after a healthy poll, a failing assert-leader exec must stop
// the function and never call o.show.
func TestLeaderLocalAssertLeaderError(t *testing.T) {
	healthy := rd("Primary", "Enabled", "Up", "Local Active")
	ft := &fakeTransport{responder: func(_ config.Role, argv []string, _ []byte) ([]byte, error) {
		switch {
		case matchCLI(argv, "show-rd"):
			return []byte(healthy), nil
		case matchCLI(argv, "assert-leader"):
			return nil, errors.New("assert boom")
		}
		return nil, nil
	}}
	o, buf := newLocalOps(t, "yes", "pri-host", ft)
	if err := o.LeaderLocal(context.Background(), ""); err == nil {
		t.Error("LeaderLocal should return the assert-leader error")
	}
	if buf.Len() != 0 {
		t.Errorf("LeaderLocal must not show output when assert-leader fails, got %q", buf.String())
	}
}

// --- redundancyLocalPrimary phase guards --------------------------------------

// TestRedundancyLocalPrimaryReleaseError closes the primary's initial RunCLI
// guard: a failed release exec must stop before any show-rd poll is attempted.
func TestRedundancyLocalPrimaryReleaseError(t *testing.T) {
	active := rd("Primary", "Enabled", "Up", "Local Active")
	ft := &fakeTransport{responder: func(_ config.Role, argv []string, _ []byte) ([]byte, error) {
		if matchCLI(argv, "release") {
			return nil, errors.New("release boom")
		}
		return nil, nil
	}}
	o, _ := newLocalOps(t, "yes", "pri-host", ft)
	if err := o.redundancyLocalPrimary(context.Background(), config.Primary, active); err == nil {
		t.Error("redundancyLocalPrimary should return the release RunCLI error")
	}
	for _, out := range ft.outputs {
		if matchCLI(out.argv, "show-rd") {
			t.Error("redundancyLocalPrimary must not poll show-rd after release fails")
		}
	}
}

// TestRedundancyLocalPrimaryReleasedTimeout closes the "released to the
// Backup" poll guard: if release never converges, no-release must never be
// sent -- that would un-release a release that never took.
func TestRedundancyLocalPrimaryReleasedTimeout(t *testing.T) {
	active := rd("Primary", "Enabled", "Up", "Local Active")
	stuck := rd("Primary", "Enabled", "Up", "Local Active") // never Enabled-Released/Down/Mate
	ft := &fakeTransport{responder: func(_ config.Role, argv []string, _ []byte) ([]byte, error) {
		if matchCLI(argv, "show-rd") {
			return []byte(stuck), nil
		}
		return nil, nil
	}}
	o, _ := newLocalOps(t, "yes", "pri-host", ft)
	o.PollAttempts = 2
	if err := o.redundancyLocalPrimary(context.Background(), config.Primary, active); err == nil {
		t.Error("redundancyLocalPrimary should time out waiting to be released")
	}
	if ft.hasUpload(cliScriptPath("no-release")) {
		t.Error("redundancyLocalPrimary must not send no-release when released never became true")
	}
}

// TestRedundancyLocalPrimaryNoReleaseError closes the primary's second RunCLI
// guard: a failed no-release exec must surface rather than proceeding to the
// un-released poll.
func TestRedundancyLocalPrimaryNoReleaseError(t *testing.T) {
	active := rd("Primary", "Enabled", "Up", "Local Active")
	released := rd("Primary", "Enabled-Released", "Down", "Mate Active")
	ft := &fakeTransport{responder: func(_ config.Role, argv []string, _ []byte) ([]byte, error) {
		switch {
		case matchCLI(argv, "no-release"):
			return nil, errors.New("no-release boom")
		case matchCLI(argv, "show-rd"):
			return []byte(released), nil
		}
		return nil, nil
	}}
	o, _ := newLocalOps(t, "yes", "pri-host", ft)
	if err := o.redundancyLocalPrimary(context.Background(), config.Primary, active); err == nil {
		t.Error("redundancyLocalPrimary should return the no-release RunCLI error")
	}
}

// TestRedundancyLocalPrimaryUnreleasedTimeout closes the "un-released" poll
// guard: if un-release never converges, the function must report the timeout
// rather than waiting for the Backup to fail back on stale state.
func TestRedundancyLocalPrimaryUnreleasedTimeout(t *testing.T) {
	active := rd("Primary", "Enabled", "Up", "Local Active")
	released := rd("Primary", "Enabled-Released", "Down", "Mate Active") // stays released forever
	ft := &fakeTransport{responder: func(_ config.Role, argv []string, _ []byte) ([]byte, error) {
		if matchCLI(argv, "show-rd") {
			return []byte(released), nil
		}
		return nil, nil
	}}
	o, _ := newLocalOps(t, "yes", "pri-host", ft)
	o.PollAttempts = 2
	if err := o.redundancyLocalPrimary(context.Background(), config.Primary, active); err == nil {
		t.Error("redundancyLocalPrimary should time out waiting to be un-released")
	}
}

// --- redundancyLocalBackup phase guards ---------------------------------------

// TestRedundancyLocalBackupBecomeActiveTimeout closes the "become active" poll
// guard: if the backup never becomes active, the function must not proceed to
// revert a handshake that never started.
func TestRedundancyLocalBackupBecomeActiveTimeout(t *testing.T) {
	inactive := rd("Backup", "Enabled", "Up", "Mate Active")
	stuck := rd("Backup", "Enabled", "Up", "Mate Active") // never becomes Local Active
	ft := &fakeTransport{responder: func(_ config.Role, argv []string, _ []byte) ([]byte, error) {
		if matchCLI(argv, "show-rd") {
			return []byte(stuck), nil
		}
		return nil, nil
	}}
	o, _ := newLocalOps(t, "yes", "bkp-host", ft)
	o.PollAttempts = 2
	if err := o.redundancyLocalBackup(context.Background(), config.Backup, inactive); err == nil {
		t.Error("redundancyLocalBackup should time out waiting to become active")
	}
	if ft.hasUpload(cliScriptPath("revert-activity")) {
		t.Error("redundancyLocalBackup must not revert a handshake that never started")
	}
}

// TestRedundancyLocalBackupDwellCancelled is the only place in the package
// that exercises dwell()'s error propagation end-to-end (distinct from
// dwell()'s own unit, already covered via the shared wait() through the sleep
// tests): a cancelled context during the mandatory ActiveDwell hold must abort
// rather than reverting early or hanging.
func TestRedundancyLocalBackupDwellCancelled(t *testing.T) {
	inactive := rd("Backup", "Enabled", "Up", "Mate Active")
	active := rd("Backup", "Enabled", "Up", "Local Active")
	ft := &fakeTransport{responder: func(_ config.Role, argv []string, _ []byte) ([]byte, error) {
		if matchCLI(argv, "show-rd") {
			return []byte(active), nil
		}
		return nil, nil
	}}
	o, _ := newLocalOps(t, "yes", "bkp-host", ft)
	o.ActiveDwell = 0 // wait() returns ctx.Err() immediately for a non-positive duration
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := o.redundancyLocalBackup(ctx, config.Backup, inactive)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("redundancyLocalBackup dwell error = %v, want context.Canceled", err)
	}
	if ft.hasUpload(cliScriptPath("revert-activity")) {
		t.Error("redundancyLocalBackup must not revert activity when the dwell is cancelled")
	}
}

// TestRedundancyLocalBackupRevertActivityError closes the backup's RunCLI
// guard: a failed revert-activity exec must stop before the final poll.
func TestRedundancyLocalBackupRevertActivityError(t *testing.T) {
	active := rd("Backup", "Enabled", "Up", "Local Active")
	ft := &fakeTransport{responder: func(_ config.Role, argv []string, _ []byte) ([]byte, error) {
		if matchCLI(argv, "revert-activity") {
			return nil, errors.New("revert boom")
		}
		return nil, nil
	}}
	o, _ := newLocalOps(t, "yes", "bkp-host", ft)
	if err := o.redundancyLocalBackup(context.Background(), config.Backup, active); err == nil {
		t.Error("redundancyLocalBackup should return the revert-activity RunCLI error")
	}
}

// TestRedundancyLocalBackupStandbyTimeout closes the final "return to
// standby" poll guard: if the backup never reports standby again, the
// function must report the timeout rather than declaring success.
func TestRedundancyLocalBackupStandbyTimeout(t *testing.T) {
	active := rd("Backup", "Enabled", "Up", "Local Active")
	stillActive := rd("Backup", "Enabled", "Up", "Local Active") // never returns to standby
	ft := &fakeTransport{responder: func(_ config.Role, argv []string, _ []byte) ([]byte, error) {
		if matchCLI(argv, "show-rd") {
			return []byte(stillActive), nil
		}
		return nil, nil
	}}
	o, _ := newLocalOps(t, "yes", "bkp-host", ft)
	o.PollAttempts = 2
	if err := o.redundancyLocalBackup(context.Background(), config.Backup, active); err == nil {
		t.Error("redundancyLocalBackup should time out waiting to return to standby")
	}
}

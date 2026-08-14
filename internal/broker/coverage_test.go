package broker

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"solace/internal/config"
)

// runErrTransport wraps a fakeTransport (declared in broker_test.go) but makes
// Run return a fixed error, exercising the best-effort cleanup / gather error
// branches that the always-nil fakeTransport.Run cannot reach. Output, Upload,
// UploadFile, Download and OutputInput are inherited from the embedded fake.
type runErrTransport struct {
	*fakeTransport
	err error
}

func (t *runErrTransport) Run(ctx context.Context, role config.Role, argv ...string) error {
	_ = t.fakeTransport.Run(ctx, role, argv...) // still record the call
	return t.err
}

// uploadErrTransport wraps a fakeTransport but makes Upload and UploadFile
// return a fixed error, exercising the upload-failure branches that the
// always-nil fakeTransport.Upload/UploadFile cannot reach. Run, Output,
// OutputInput and Download are inherited from the embedded fake.
type uploadErrTransport struct {
	*fakeTransport
	err error
}

func (t *uploadErrTransport) Upload(ctx context.Context, role config.Role, data []byte, dest string) error {
	_ = t.fakeTransport.Upload(ctx, role, data, dest) // still record the call
	return t.err
}

func (t *uploadErrTransport) UploadFile(ctx context.Context, role config.Role, local, dest string) error {
	_ = t.fakeTransport.UploadFile(ctx, role, local, dest) // still record the call
	return t.err
}

// downloadErrTransport wraps a fakeTransport but makes Download return a fixed
// error, exercising the download-failure branches that the always-nil
// fakeTransport.Download cannot reach. When match is non-empty, only a
// Download whose remote path equals match fails and every other remote path
// succeeds -- this isolates gatherNode's best-effort bundle pull from its
// fail-loud main-archive download.
type downloadErrTransport struct {
	*fakeTransport
	err   error
	match string
}

func (t *downloadErrTransport) Download(ctx context.Context, role config.Role, remote, local string) error {
	_ = t.fakeTransport.Download(ctx, role, remote, local) // still record the call
	if t.match == "" || remote == t.match {
		return t.err
	}
	return nil
}

// runErrMatchTransport wraps a fakeTransport but fails Run only when
// match(argv) reports true, isolating a failure to one specific Run call in a
// sequence -- e.g. gatherNode's best-effort bundle cleanup, which must warn
// without the node's earlier fail-loud Run calls also failing.
type runErrMatchTransport struct {
	*fakeTransport
	err   error
	match func(argv []string) bool
}

func (t *runErrMatchTransport) Run(ctx context.Context, role config.Role, argv ...string) error {
	_ = t.fakeTransport.Run(ctx, role, argv...) // still record the call
	if t.match(argv) {
		return t.err
	}
	return nil
}

// --- New / out / show ------------------------------------------------------

func TestNewDefaults(t *testing.T) {
	ft := &fakeTransport{}
	cfg := &config.Config{}
	o := New(ft, cfg, nil)
	if o.T != ft {
		t.Error("New did not set Transport")
	}
	if o.Cfg != cfg {
		t.Error("New did not set Cfg")
	}
	if o.PollInterval != 2*time.Second {
		t.Errorf("New PollInterval = %v, want 2s", o.PollInterval)
	}
	if o.PollAttempts != 60 {
		t.Errorf("New PollAttempts = %d, want 60", o.PollAttempts)
	}
	if o.Out != os.Stdout {
		t.Error("New Out should default to os.Stdout")
	}
}

func TestOutDefaultsToStdout(t *testing.T) {
	o := &Ops{}
	if o.out() != os.Stdout {
		t.Error("out() should return os.Stdout when Out is nil")
	}
	buf := &bytes.Buffer{}
	o.Out = buf
	if o.out() != buf {
		t.Error("out() should return the configured Out")
	}
}

func TestShowWritesToStdout(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	old := os.Stdout
	os.Stdout = w

	o := &Ops{} // Out nil -> out() resolves to os.Stdout
	o.show([]byte("hello-stdout"))

	os.Stdout = old
	if err := w.Close(); err != nil {
		t.Fatalf("close pipe writer: %v", err)
	}
	data, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read pipe: %v", err)
	}
	if got := string(data); got != "hello-stdout" {
		t.Errorf("show wrote %q, want %q", got, "hello-stdout")
	}
}

// --- sleep / poll ----------------------------------------------------------

func TestSleepElapses(t *testing.T) {
	o := &Ops{PollInterval: time.Millisecond}
	if err := o.sleep(context.Background()); err != nil {
		t.Errorf("sleep with a tiny interval should return nil, got %v", err)
	}
}

func TestSleepZeroIntervalReturnsCtxErr(t *testing.T) {
	o := &Ops{PollInterval: 0}
	if err := o.sleep(context.Background()); err != nil {
		t.Errorf("sleep with zero interval and live context should return nil, got %v", err)
	}
}

func TestSleepCancelled(t *testing.T) {
	o := &Ops{PollInterval: time.Second}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := o.sleep(ctx); !errors.Is(err, context.Canceled) {
		t.Errorf("sleep on a cancelled context = %v, want context.Canceled", err)
	}
}

func TestPollCondError(t *testing.T) {
	o := &Ops{PollInterval: 0, PollAttempts: 3}
	want := errors.New("cond boom")
	err := o.poll(context.Background(), "x", func(context.Context) (bool, error) {
		return false, want
	})
	if !errors.Is(err, want) {
		t.Errorf("poll should propagate the cond error, got %v", err)
	}
}

func TestPollContextCancelled(t *testing.T) {
	o := &Ops{PollInterval: time.Second, PollAttempts: 5}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := o.poll(ctx, "x", func(context.Context) (bool, error) {
		return false, nil // never satisfied -> falls through to sleep, which sees the cancel
	})
	if !errors.Is(err, context.Canceled) {
		t.Errorf("poll on a cancelled context = %v, want context.Canceled", err)
	}
}

// --- removeCLI failure branch ----------------------------------------------

func TestRemoveCLIWarnsOnFailure(t *testing.T) {
	var logs []string
	rt := &runErrTransport{fakeTransport: &fakeTransport{}, err: errors.New("rm failed")}
	o := &Ops{
		T:            rt,
		Cfg:          &config.Config{},
		PollInterval: 0,
		Log:          func(f string, a ...any) { logs = append(logs, fmt.Sprintf(f, a...)) },
	}
	o.removeCLI(context.Background(), config.Primary, "a", "b")
	if len(logs) != 1 || !strings.Contains(logs[0], "[WARN] cleanup") {
		t.Errorf("removeCLI should log a cleanup warning on failure, got %v", logs)
	}
	// The rm argv must still have been issued with both script paths.
	if !ranContains(rt.fakeTransport, "rm", "-f", cliScriptPath("a"), cliScriptPath("b")) {
		t.Errorf("removeCLI rm argv missing paths: %v", rt.fakeTransport.runs)
	}
}

// --- pure helper edge cases ------------------------------------------------

func TestFieldLabelWithoutColon(t *testing.T) {
	// Label present on the line but no ": " separator -> "".
	if got := field("PlainLabelLine has no colon separator\n", "PlainLabelLine"); got != "" {
		t.Errorf("field with no colon = %q, want empty", got)
	}
}

func TestLastLinesEqualCount(t *testing.T) {
	if got := lastLines("a\nb\nc\n", 3); got != "a\nb\nc\n" {
		t.Errorf("lastLines n==count = %q, want %q", got, "a\nb\nc\n")
	}
}

// --- ExecCLI branches ------------------------------------------------------

func TestExecCLIWarnsOnErrorOutput(t *testing.T) {
	local := filepath.Join(t.TempDir(), "script.cli")
	ft := &fakeTransport{responder: func(_ config.Role, _ []string, _ []byte) ([]byte, error) {
		return []byte("invalid command detected\n"), nil
	}}
	var logs []string
	o := &Ops{
		T:            ft,
		Cfg:          &config.Config{},
		Out:          &bytes.Buffer{},
		PollInterval: 0,
		Log:          func(f string, a ...any) { logs = append(logs, fmt.Sprintf(f, a...)) },
	}
	if err := o.ExecCLI(context.Background(), config.Primary, local); err != nil {
		t.Fatalf("ExecCLI error: %v", err)
	}
	joined := strings.Join(logs, "\n")
	if !strings.Contains(joined, "[WARN] errors detected") {
		t.Errorf("ExecCLI should warn on error-looking output, got %v", logs)
	}
}

func TestExecCLIRunError(t *testing.T) {
	local := filepath.Join(t.TempDir(), "script.cli")
	ft := &fakeTransport{responder: func(_ config.Role, _ []string, _ []byte) ([]byte, error) {
		return []byte("partial\n"), errors.New("exec boom")
	}}
	o, _ := newTestOps(t, &config.Config{}, ft)
	if err := o.ExecCLI(context.Background(), config.Primary, local); err == nil {
		t.Error("ExecCLI should return an error when the CLI run fails")
	}
	// Cleanup is still attempted even on run failure.
	if !ranContains(ft, "rm", "-f") {
		t.Error("ExecCLI should clean up its script even when the run errored")
	}
}

// --- ServerCert read-error branches ----------------------------------------

func TestServerCertBundleReadError(t *testing.T) {
	cfg := &config.Config{TLS: config.TLS{Cert: "/no/such/cert.pem", CertKey: "/no/such/key.pem"}}
	o, _ := newTestOps(t, cfg, &fakeTransport{})
	if err := o.ServerCert(context.Background(), "2026-07-31", config.Primary); err == nil {
		t.Error("ServerCert should error when the key/cert files are unreadable")
	}
}

func TestServerCertCAReadError(t *testing.T) {
	dir := t.TempDir()
	key := filepath.Join(dir, "tls.key")
	crt := filepath.Join(dir, "tls.crt")
	writeFile(t, key, "KEY\n")
	writeFile(t, crt, "CERT\n")
	cfg := &config.Config{TLS: config.TLS{Cert: crt, CertKey: key, CAs: []string{"/no/such/ca.pem"}}}
	o, _ := newTestOps(t, cfg, &fakeTransport{})
	if err := o.ServerCert(context.Background(), "2026-07-31", config.Primary); err == nil {
		t.Error("ServerCert should error when a CA file is unreadable")
	}
}

// --- DomainCerts bad-filename branch ---------------------------------------

func TestDomainCertsBadFilename(t *testing.T) {
	ft := &fakeTransport{}
	o, _ := newTestOps(t, &config.Config{}, ft)
	// CA name is valid; the certificate filename contains a space.
	if err := o.DomainCerts(context.Background(), config.Primary, "certs", map[string]string{"myca": "bad name.pem"}); err == nil {
		t.Error("DomainCerts should reject a certificate filename with a space")
	}
	if len(ft.uploadFiles) != 0 {
		t.Error("DomainCerts must not upload when the filename is invalid")
	}
}

// --- Diagnostics branches --------------------------------------------------

func TestDiagnosticsTwoRolesNoBundle(t *testing.T) {
	dest := filepath.Join(t.TempDir(), "diag")
	ft := &fakeTransport{responder: func(_ config.Role, argv []string, _ []byte) ([]byte, error) {
		if len(argv) == 1 && argv[0] == "hostname" {
			return []byte("host\n"), nil
		}
		// gather-configs output has no "Diagnostics saved" line -> bundle skipped.
		return nil, nil
	}}
	o, _ := newTestOps(t, &config.Config{}, ft)
	if err := o.Diagnostics(context.Background(), dest, "20260731", 5, config.Primary, config.Backup); err != nil {
		t.Fatalf("Diagnostics error: %v", err)
	}
	// Two roles, each pulling only the gather-configs.zip (no diagnostics bundle).
	if len(ft.downloads) != 2 {
		t.Errorf("Diagnostics downloads = %v, want 2 (one zip per role)", ft.downloads)
	}
	for _, d := range ft.downloads {
		if d.remote != JailRoot+"/gather-configs.zip" {
			t.Errorf("unexpected download remote %q", d.remote)
		}
	}
}

func TestDiagnosticsRunError(t *testing.T) {
	dest := filepath.Join(t.TempDir(), "diag")
	rt := &runErrTransport{fakeTransport: &fakeTransport{}, err: errors.New("run boom")}
	o := &Ops{T: rt, Cfg: &config.Config{}, Out: &bytes.Buffer{}, PollInterval: 0}
	if err := o.Diagnostics(context.Background(), dest, "ts", 1, config.Primary); err == nil {
		t.Error("Diagnostics should surface a transport Run error")
	}
}

// --- Leader / Redundancy error paths ---------------------------------------

func TestLeaderPollCondError(t *testing.T) {
	// revert-activity succeeds, but every `show redundancy` errors, so the poll
	// condition propagates the error (and the timeout detail dump still runs).
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
	o, _ := newTestOps(t, &config.Config{Redundancy: "yes"}, ft)
	if err := o.Leader(context.Background()); err == nil {
		t.Error("Leader should return the poll condition error")
	}
	if !sawDetail {
		t.Error("Leader should still dump show-redundancy-detail after the error")
	}
}

func TestRedundancyShowRDError(t *testing.T) {
	ft := &fakeTransport{responder: func(_ config.Role, argv []string, _ []byte) ([]byte, error) {
		if matchCLI(argv, "show-rd") {
			return nil, errors.New("show boom")
		}
		return nil, nil
	}}
	o, _ := newTestOps(t, &config.Config{Redundancy: "yes"}, ft)
	if err := o.Redundancy(context.Background()); err == nil {
		t.Error("Redundancy should surface the initial show redundancy error")
	}
}

func TestRedundancyUnhealthyPrimary(t *testing.T) {
	unhealthy := "Configuration Status : Enabled\nRedundancy Status : Down\nActive-Standby Role : Primary\nADB Link To Mate : Up\nADB Hello To Mate : Up\n"
	ft := &fakeTransport{responder: func(_ config.Role, argv []string, _ []byte) ([]byte, error) {
		if matchCLI(argv, "show-rd") {
			return []byte(unhealthy), nil
		}
		return nil, nil
	}}
	o, buf := newTestOps(t, &config.Config{Redundancy: "yes"}, ft)
	if err := o.Redundancy(context.Background()); err == nil {
		t.Error("Redundancy should fail when the Primary is not healthy")
	}
	if !strings.Contains(buf.String(), "Redundancy Status : Down") {
		t.Errorf("Redundancy should show the unhealthy output, got %q", buf.String())
	}
}

func TestRedundancyNeitherActive(t *testing.T) {
	adb := "ADB Link To Mate : Up\nADB Hello To Mate : Up\n"
	mk := func(role, act string) string {
		return "Configuration Status : Enabled\nRedundancy Status : Up\nActive-Standby Role : " +
			role + "\n" + adb + "Activity Status : " + act + "\n"
	}
	ft := &fakeTransport{responder: func(role config.Role, argv []string, _ []byte) ([]byte, error) {
		if !matchCLI(argv, "show-rd") {
			return nil, nil
		}
		if role == config.Primary {
			return []byte(mk("Primary", "Mate Active")), nil // healthy but not locally active
		}
		return []byte(mk("Backup", "Mate Active")), nil // backup also not locally active
	}}
	o, _ := newTestOps(t, &config.Config{Redundancy: "yes"}, ft)
	if err := o.Redundancy(context.Background()); err == nil {
		t.Error("Redundancy should fail when neither node is active")
	}
}

// --- RunCLI upload error -----------------------------------------------------

// TestRunCLIUploadError closes the shared RunCLI primitive's upload-failure
// branch: every config/verify op is built on RunCLI, so a broker-side upload
// failure (network blip, full disk) must stop before the CLI binary is ever
// invoked, wrapped with the script name -- not silently continue to exec a
// script that never arrived.
func TestRunCLIUploadError(t *testing.T) {
	ut := &uploadErrTransport{fakeTransport: &fakeTransport{}, err: errors.New("upload boom")}
	o := &Ops{T: ut, Cfg: &config.Config{}, Out: &bytes.Buffer{}, PollInterval: 0}
	_, err := o.RunCLI(context.Background(), config.Primary, "probe", "body")
	if err == nil || !strings.Contains(err.Error(), "upload cli script") {
		t.Errorf("RunCLI upload error = %v, want wrapped %q", err, "upload cli script")
	}
	if len(ut.fakeTransport.outputs) != 0 {
		t.Error("RunCLI must not exec the CLI binary when the upload fails")
	}
}

// --- ServerCert upload / RunCLI error branches -------------------------------

// TestServerCertUploadError closes ServerCert's Upload-failure branch: the
// bundle carries the private key, so a broker-side upload failure must stop
// the loop immediately rather than attempting apply-server-certs against a
// file that never arrived.
func TestServerCertUploadError(t *testing.T) {
	dir := t.TempDir()
	key := filepath.Join(dir, "tls.key")
	crt := filepath.Join(dir, "tls.crt")
	writeFile(t, key, "KEY\n")
	writeFile(t, crt, "CERT\n")
	cfg := &config.Config{TLS: config.TLS{Cert: crt, CertKey: key}}
	ut := &uploadErrTransport{fakeTransport: &fakeTransport{}, err: errors.New("upload boom")}
	o := &Ops{T: ut, Cfg: cfg, Out: &bytes.Buffer{}, PollInterval: 0}
	err := o.ServerCert(context.Background(), "2026-07-31", config.Primary)
	if err == nil || !strings.Contains(err.Error(), "upload certificate") {
		t.Errorf("ServerCert upload error = %v, want wrapped %q", err, "upload certificate")
	}
	if len(ut.fakeTransport.outputs) != 0 {
		t.Error("ServerCert must not run apply-server-certs when the upload fails")
	}
}

// TestServerCertRunCLIError closes ServerCert's apply-server-certs error
// branch: a failed apply must not call o.show or continue to the next role.
func TestServerCertRunCLIError(t *testing.T) {
	dir := t.TempDir()
	key := filepath.Join(dir, "tls.key")
	crt := filepath.Join(dir, "tls.crt")
	writeFile(t, key, "KEY\n")
	writeFile(t, crt, "CERT\n")
	cfg := &config.Config{TLS: config.TLS{Cert: crt, CertKey: key}}
	ft := &fakeTransport{responder: func(_ config.Role, argv []string, _ []byte) ([]byte, error) {
		if matchCLI(argv, "apply-server-certs") {
			return nil, errors.New("apply boom")
		}
		return nil, nil
	}}
	o, buf := newTestOps(t, cfg, ft)
	if err := o.ServerCert(context.Background(), "2026-07-31", config.Primary); err == nil {
		t.Error("ServerCert should return the apply-server-certs error")
	}
	if buf.Len() != 0 {
		t.Errorf("ServerCert must not show output when apply-server-certs fails, got %q", buf.String())
	}
}

// --- DomainCerts upload / RunCLI error branches ------------------------------

// TestDomainCertsUploadFileError closes DomainCerts' per-CA UploadFile-failure
// branch: a failed CA upload must stop before load-domain-certs runs.
func TestDomainCertsUploadFileError(t *testing.T) {
	ut := &uploadErrTransport{fakeTransport: &fakeTransport{}, err: errors.New("upload boom")}
	o := &Ops{T: ut, Cfg: &config.Config{}, Out: &bytes.Buffer{}, PollInterval: 0}
	err := o.DomainCerts(context.Background(), config.Primary, "certs", map[string]string{"myca": "myca.pem"})
	if err == nil || !strings.Contains(err.Error(), "upload domain certificate") {
		t.Errorf("DomainCerts upload error = %v, want wrapped %q", err, "upload domain certificate")
	}
	if len(ut.fakeTransport.outputs) != 0 {
		t.Error("DomainCerts must not run load-domain-certs when a CA upload fails")
	}
}

// TestDomainCertsRunCLIError closes DomainCerts' load-domain-certs error
// branch: a failed load must not show output.
func TestDomainCertsRunCLIError(t *testing.T) {
	ft := &fakeTransport{responder: func(_ config.Role, argv []string, _ []byte) ([]byte, error) {
		if matchCLI(argv, "load-domain-certs") {
			return nil, errors.New("load boom")
		}
		return nil, nil
	}}
	o, buf := newTestOps(t, &config.Config{}, ft)
	err := o.DomainCerts(context.Background(), config.Primary, "certs", map[string]string{"myca": "myca.pem"})
	if err == nil {
		t.Error("DomainCerts should return the load-domain-certs error")
	}
	if buf.Len() != 0 {
		t.Errorf("DomainCerts must not show output when load-domain-certs fails, got %q", buf.String())
	}
}

// --- DisableDefaultVPN RunCLI error branches ---------------------------------

// TestDisableDefaultVPNDisableError closes the disable-default-vpn error
// branch: if disabling the VPN itself fails, DisableDefaultVPN must not go on
// to read back show-vpn or run the cleanup.
func TestDisableDefaultVPNDisableError(t *testing.T) {
	ft := &fakeTransport{responder: func(_ config.Role, argv []string, _ []byte) ([]byte, error) {
		if matchCLI(argv, "disable-default-vpn") {
			return nil, errors.New("disable boom")
		}
		return nil, nil
	}}
	o, _ := newTestOps(t, &config.Config{}, ft)
	if err := o.DisableDefaultVPN(context.Background(), config.Primary); err == nil {
		t.Error("DisableDefaultVPN should return the disable-default-vpn error")
	}
	for _, out := range ft.outputs {
		if matchCLI(out.argv, "show-vpn") {
			t.Error("DisableDefaultVPN must not read back show-vpn when disabling fails")
		}
	}
}

// TestDisableDefaultVPNShowError closes the show-vpn confirmation-read error
// branch: a failed readback must surface rather than being swallowed as
// success, and cleanup of the uploaded scripts must not run.
func TestDisableDefaultVPNShowError(t *testing.T) {
	ft := &fakeTransport{responder: func(_ config.Role, argv []string, _ []byte) ([]byte, error) {
		if matchCLI(argv, "show-vpn") {
			return nil, errors.New("show boom")
		}
		return nil, nil
	}}
	o, _ := newTestOps(t, &config.Config{}, ft)
	if err := o.DisableDefaultVPN(context.Background(), config.Primary); err == nil {
		t.Error("DisableDefaultVPN should return the show-vpn error")
	}
	if ranContains(ft, "rm", "-f") {
		t.Error("DisableDefaultVPN must not clean up its cli scripts when show-vpn fails")
	}
}

// --- DisableDefaultUsers RunCLI error branches -------------------------------

// TestDisableDefaultUsersShowVPNError closes the initial show-vpn listing's
// error branch: a failed VPN listing must stop before parsing/disabling
// anything.
func TestDisableDefaultUsersShowVPNError(t *testing.T) {
	ft := &fakeTransport{responder: func(_ config.Role, argv []string, _ []byte) ([]byte, error) {
		if matchCLI(argv, "show-vpn") {
			return nil, errors.New("show boom")
		}
		return nil, nil
	}}
	o, _ := newTestOps(t, &config.Config{}, ft)
	if err := o.DisableDefaultUsers(context.Background(), config.Primary); err == nil {
		t.Error("DisableDefaultUsers should return the show-vpn error")
	}
	if ft.hasUpload(cliScriptPath("disable-default-usernames")) {
		t.Error("DisableDefaultUsers must not disable anything when the VPN listing fails")
	}
}

// TestDisableDefaultUsersDisableError closes the disable-default-usernames
// error branch: a failed disable call must surface rather than reporting
// success.
func TestDisableDefaultUsersDisableError(t *testing.T) {
	row := func(name string) string { return name + strings.Repeat(" ", 40-len(name)) + "Yes" }
	list := strings.Join([]string{strings.Repeat("-", 40), row("default")}, "\r\n") + "\r\n"
	ft := &fakeTransport{responder: func(_ config.Role, argv []string, _ []byte) ([]byte, error) {
		switch {
		case matchCLI(argv, "show-vpn"):
			return []byte(list), nil
		case matchCLI(argv, "disable-default-usernames"):
			return nil, errors.New("disable boom")
		}
		return nil, nil
	}}
	o, _ := newTestOps(t, &config.Config{}, ft)
	if err := o.DisableDefaultUsers(context.Background(), config.Primary); err == nil {
		t.Error("DisableDefaultUsers should return the disable-default-usernames error")
	}
}

// --- ProductKeys RunCLI error branch ------------------------------------------

// TestProductKeysRunCLIErrorStopsLoop closes ProductKeys' per-role loop: a
// transport failure applying keys to one role must stop the loop rather than
// silently moving to the next role, which would leave an inconsistent HA pair.
func TestProductKeysRunCLIErrorStopsLoop(t *testing.T) {
	ft := &fakeTransport{responder: func(role config.Role, argv []string, _ []byte) ([]byte, error) {
		if role == config.Primary && matchCLI(argv, "product-keys") {
			return nil, errors.New("apply boom")
		}
		return []byte("Product key applied.\n"), nil
	}}
	o, _ := newTestOps(t, &config.Config{}, ft)
	if err := o.ProductKeys(context.Background(), []string{"KEY-1"}, config.Primary, config.Backup); err == nil {
		t.Error("ProductKeys should stop the loop on a per-role transport error")
	}
	for _, out := range ft.outputs {
		if out.role == config.Backup {
			t.Errorf("ProductKeys must not continue to the Backup after the Primary failed: %v", ft.outputs)
		}
	}
}

// --- AdditionalUsers transport-level error -----------------------------------

// TestAdditionalUsersRunCLITransportError closes the transport-level failure
// branch: unlike TestAdditionalUsersReportsExistingUser (RunCLI succeeds; the
// broker's transcript merely contains an error string), this is a hard exec
// failure -- the deferred removeCLI must still delete the uploaded script,
// which carries every plaintext password.
func TestAdditionalUsersRunCLITransportError(t *testing.T) {
	ft := &fakeTransport{responder: func(_ config.Role, argv []string, _ []byte) ([]byte, error) {
		if matchCLI(argv, "additional-users") {
			return nil, errors.New("exec boom")
		}
		return nil, nil
	}}
	o, _ := newTestOps(t, &config.Config{}, ft)
	err := o.AdditionalUsers(context.Background(), config.Primary, appUsers())
	if err == nil || !strings.Contains(err.Error(), "exec boom") {
		t.Errorf("AdditionalUsers transport error = %v, want wrapped %q", err, "exec boom")
	}
	if !ft.removed(cliScriptPath("additional-users")) {
		t.Error("a hard transport failure must still remove the uploaded script")
	}
}

// --- ExecCLI UploadFile error -------------------------------------------------

// TestExecCLIUploadFileError closes ExecCLI's UploadFile-failure branch: a
// failed upload of the operator's script must be reported with the script's
// basename and must skip the exec/cleanup entirely -- nothing was actually
// placed on the broker to run or remove.
func TestExecCLIUploadFileError(t *testing.T) {
	local := filepath.Join(t.TempDir(), "myscript.cli")
	ut := &uploadErrTransport{fakeTransport: &fakeTransport{}, err: errors.New("upload boom")}
	o := &Ops{T: ut, Cfg: &config.Config{}, Out: &bytes.Buffer{}, PollInterval: 0}
	err := o.ExecCLI(context.Background(), config.Primary, local)
	if err == nil || !strings.Contains(err.Error(), "myscript.cli") {
		t.Errorf("ExecCLI upload error = %v, want it to name %q", err, "myscript.cli")
	}
	if len(ut.fakeTransport.outputs) != 0 {
		t.Error("ExecCLI must not exec or clean up when the upload fails")
	}
}

// --- RemoveDomainCerts RunCLI error -------------------------------------------

// TestRemoveDomainCertsRunCLIError closes the remove-domain-certs error
// branch: teardown must fail loud rather than report success when the broker
// rejects the removal, and must not run the cleanup rm.
func TestRemoveDomainCertsRunCLIError(t *testing.T) {
	ft := &fakeTransport{responder: func(_ config.Role, argv []string, _ []byte) ([]byte, error) {
		if matchCLI(argv, "remove-domain-certs") {
			return nil, errors.New("remove boom")
		}
		return nil, nil
	}}
	o, _ := newTestOps(t, &config.Config{}, ft)
	if err := o.RemoveDomainCerts(context.Background(), config.Primary, []string{"myca"}); err == nil {
		t.Error("RemoveDomainCerts should return the remove-domain-certs error")
	}
	if ranContains(ft, "rm", "-f") {
		t.Error("RemoveDomainCerts must not clean up its cli script when the removal fails")
	}
}

// --- Login transport error ----------------------------------------------------

// TestLoginTransportError distinguishes a transport/network failure (curl
// couldn't even run) from an HTTP-level auth failure, already covered by
// TestLoginFailure/TestLoginNoResponse: Login must return the error rather
// than reporting "Login failed" as if it got an HTTP response.
func TestLoginTransportError(t *testing.T) {
	ft := &fakeTransport{responder: func(_ config.Role, _ []string, _ []byte) ([]byte, error) {
		return nil, errors.New("curl boom")
	}}
	o, buf := newTestOps(t, &config.Config{}, ft)
	ok, err := o.Login(context.Background(), config.Primary, "admin", "s3cret")
	if ok || err == nil || !strings.Contains(err.Error(), "SEMP request failed") {
		t.Errorf("Login transport error: ok=%v err=%v, want false/wrapped %q", ok, err, "SEMP request failed")
	}
	if strings.Contains(buf.String(), "Login failed") {
		t.Errorf("Login must return before logging 'Login failed', got %q", buf.String())
	}
}

// --- Leader RunCLI error boundaries -------------------------------------------

// TestLeaderRevertActivityError closes Leader's initial Backup-revert error
// branch: if the revert fails, Leader must never attempt the redundancy poll
// at all.
func TestLeaderRevertActivityError(t *testing.T) {
	ft := &fakeTransport{responder: func(_ config.Role, argv []string, _ []byte) ([]byte, error) {
		if matchCLI(argv, "revert-activity") {
			return nil, errors.New("revert boom")
		}
		return nil, nil
	}}
	o, _ := newTestOps(t, &config.Config{Redundancy: "yes"}, ft)
	if err := o.Leader(context.Background()); err == nil {
		t.Error("Leader should return the revert-activity error")
	}
	for _, out := range ft.outputs {
		if matchCLI(out.argv, "show-rd") {
			t.Error("Leader must not poll show-rd after the initial revert-activity fails")
		}
	}
}

// TestLeaderAssertLeaderError closes Leader's final assert-leader error
// branch: after a healthy poll, a failing assert-leader exec must stop the
// function and never call o.show (no partial/misleading output shown to the
// operator).
func TestLeaderAssertLeaderError(t *testing.T) {
	healthy := "Configuration Status : Enabled\nRedundancy Status : Up\nActive-Standby Role : Primary\nADB Link To Mate : Up\nADB Hello To Mate : Up\n"
	ft := &fakeTransport{responder: func(_ config.Role, argv []string, _ []byte) ([]byte, error) {
		switch {
		case matchCLI(argv, "show-rd"):
			return []byte(healthy), nil
		case matchCLI(argv, "assert-leader"):
			return nil, errors.New("assert boom")
		}
		return nil, nil
	}}
	o, buf := newTestOps(t, &config.Config{Redundancy: "yes"}, ft)
	if err := o.Leader(context.Background()); err == nil {
		t.Error("Leader should return the assert-leader error")
	}
	if buf.Len() != 0 {
		t.Errorf("Leader must not show output when assert-leader fails, got %q", buf.String())
	}
}

// --- Redundancy top-level orchestration guards --------------------------------

// TestRedundancyReleaseError closes Redundancy's first orchestration guard: if
// releasing activity to the Backup fails, Redundancy must abort immediately
// rather than going on to read the Backup's show-rd (061's top-level drill).
func TestRedundancyReleaseError(t *testing.T) {
	priActive := rd("Primary", "Enabled", "Up", "Local Active")
	backupQueried := false
	ft := &fakeTransport{responder: func(role config.Role, argv []string, _ []byte) ([]byte, error) {
		switch {
		case matchCLI(argv, "show-rd") && role == config.Backup:
			backupQueried = true
			return nil, nil
		case matchCLI(argv, "show-rd"):
			return []byte(priActive), nil
		case matchCLI(argv, "release"):
			return nil, errors.New("release boom")
		}
		return nil, nil
	}}
	o, _ := newTestOps(t, &config.Config{Redundancy: "yes"}, ft)
	if err := o.Redundancy(context.Background()); err == nil {
		t.Error("Redundancy should return the release error")
	}
	if backupQueried {
		t.Error("Redundancy must not query the Backup's show-rd after release fails")
	}
}

// TestRedundancyBackupShowError closes Redundancy's post-release Backup read:
// once releaseToBackup completes cleanly, a failure reading the Backup's own
// show-rd must abort the drill rather than proceeding into revertToPrimary
// with a half-read pair.
func TestRedundancyBackupShowError(t *testing.T) {
	priActive := rd("Primary", "Enabled", "Up", "Local Active")
	priReleased := rd("Primary", "Enabled-Released", "Down", "Mate Active")
	priUnreleased := rd("Primary", "Enabled", "Up", "Mate Active")
	bkActive := rd("Backup", "Enabled", "Up", "Local Active")

	priCalls, bkCalls := 0, 0
	ft := &fakeTransport{responder: func(role config.Role, argv []string, _ []byte) ([]byte, error) {
		if !matchCLI(argv, "show-rd") {
			return nil, nil // release / no-release
		}
		if role == config.Primary {
			priCalls++
			switch priCalls {
			case 1:
				return []byte(priActive), nil
			case 2:
				return []byte(priReleased), nil
			default:
				return []byte(priUnreleased), nil
			}
		}
		bkCalls++
		if bkCalls == 1 {
			return []byte(bkActive), nil // releaseToBackup's own un-release poll read
		}
		return nil, errors.New("backup show boom") // Redundancy's post-release read
	}}
	o, _ := newTestOps(t, &config.Config{Redundancy: "yes"}, ft)
	if err := o.Redundancy(context.Background()); err == nil {
		t.Error("Redundancy should return the post-release Backup show-rd error")
	}
}

// TestRedundancyRevertToPrimaryError closes Redundancy's final orchestration
// guard: a failure in revertToPrimary must abort the drill rather than
// declaring the reversion successful.
func TestRedundancyRevertToPrimaryError(t *testing.T) {
	priStandby := rd("Primary", "Enabled", "Up", "Mate Active") // healthy, not active -> skip releaseToBackup
	bkActive := rd("Backup", "Enabled", "Up", "Local Active")
	ft := &fakeTransport{responder: func(role config.Role, argv []string, _ []byte) ([]byte, error) {
		switch {
		case matchCLI(argv, "show-rd") && role == config.Primary:
			return []byte(priStandby), nil
		case matchCLI(argv, "show-rd") && role == config.Backup:
			return []byte(bkActive), nil
		case matchCLI(argv, "revert-activity"):
			return nil, errors.New("revert boom")
		}
		return nil, nil
	}}
	o, _ := newTestOps(t, &config.Config{Redundancy: "yes"}, ft)
	if err := o.Redundancy(context.Background()); err == nil {
		t.Error("Redundancy should return the revertToPrimary error")
	}
}

// --- releaseToBackup phase guards ---------------------------------------------

// TestReleaseToBackupReleaseError closes releaseToBackup's initial RunCLI
// guard: a failed release exec must stop before any show-rd poll is attempted.
func TestReleaseToBackupReleaseError(t *testing.T) {
	ft := &fakeTransport{responder: func(_ config.Role, argv []string, _ []byte) ([]byte, error) {
		if matchCLI(argv, "release") {
			return nil, errors.New("release boom")
		}
		return nil, nil
	}}
	o, _ := newTestOps(t, &config.Config{Redundancy: "yes"}, ft)
	if err := o.releaseToBackup(context.Background()); err == nil {
		t.Error("releaseToBackup should return the release RunCLI error")
	}
	for _, out := range ft.outputs {
		if matchCLI(out.argv, "show-rd") {
			t.Error("releaseToBackup must not poll show-rd when release fails")
		}
	}
}

// TestReleaseToBackupReleasedTimeout closes the "released" poll guard: if the
// release never converges, no-release must never be sent -- that would
// un-release a release that never took.
func TestReleaseToBackupReleasedTimeout(t *testing.T) {
	stuck := rd("Primary", "Enabled", "Up", "Local Active") // never Enabled-Released/Down/Mate
	ft := &fakeTransport{responder: func(_ config.Role, argv []string, _ []byte) ([]byte, error) {
		if matchCLI(argv, "show-rd") {
			return []byte(stuck), nil
		}
		return nil, nil
	}}
	o, _ := newTestOps(t, &config.Config{Redundancy: "yes"}, ft)
	o.PollAttempts = 2
	if err := o.releaseToBackup(context.Background()); err == nil {
		t.Error("releaseToBackup should time out when the release never converges")
	}
	if ft.hasUpload(cliScriptPath("no-release")) {
		t.Error("releaseToBackup must not send no-release when released never became true")
	}
}

// TestReleaseToBackupNoReleaseError closes releaseToBackup's second RunCLI
// guard: a failed no-release exec must surface rather than proceeding to the
// un-released poll.
func TestReleaseToBackupNoReleaseError(t *testing.T) {
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
	o, _ := newTestOps(t, &config.Config{Redundancy: "yes"}, ft)
	if err := o.releaseToBackup(context.Background()); err == nil {
		t.Error("releaseToBackup should return the no-release RunCLI error")
	}
}

// TestReleaseToBackupUnreleasedTimeout closes the "un-released" poll guard: if
// the un-release never converges, the function must report the timeout
// rather than declaring the Backup active.
func TestReleaseToBackupUnreleasedTimeout(t *testing.T) {
	released := rd("Primary", "Enabled-Released", "Down", "Mate Active") // stays released forever
	ft := &fakeTransport{responder: func(_ config.Role, argv []string, _ []byte) ([]byte, error) {
		if matchCLI(argv, "show-rd") {
			return []byte(released), nil
		}
		return nil, nil
	}}
	o, _ := newTestOps(t, &config.Config{Redundancy: "yes"}, ft)
	o.PollAttempts = 2
	if err := o.releaseToBackup(context.Background()); err == nil {
		t.Error("releaseToBackup should time out when un-released never converges")
	}
}

// --- revertToPrimary RunCLI error ---------------------------------------------

// TestRevertToPrimaryRunCLIError closes revertToPrimary's RunCLI guard: if the
// Backup's revert-activity command fails to run, the function must not enter
// the final poll at all.
func TestRevertToPrimaryRunCLIError(t *testing.T) {
	ft := &fakeTransport{responder: func(_ config.Role, argv []string, _ []byte) ([]byte, error) {
		if matchCLI(argv, "revert-activity") {
			return nil, errors.New("revert boom")
		}
		return nil, nil
	}}
	o, _ := newTestOps(t, &config.Config{Redundancy: "yes"}, ft)
	if err := o.revertToPrimary(context.Background()); err == nil {
		t.Error("revertToPrimary should return the revert-activity RunCLI error")
	}
	for _, out := range ft.outputs {
		if matchCLI(out.argv, "show-rd") {
			t.Error("revertToPrimary must not poll show-rd after revert-activity fails")
		}
	}
}

// --- showRDPair primary error --------------------------------------------------

// TestShowRDPairPrimaryError closes showRDPair's short-circuit: when the
// Primary read fails it must return ("","",err) immediately and never query
// the Backup, avoiding acting on a half-read pair.
func TestShowRDPairPrimaryError(t *testing.T) {
	ft := &fakeTransport{responder: func(role config.Role, argv []string, _ []byte) ([]byte, error) {
		if matchCLI(argv, "show-rd") && role == config.Primary {
			return nil, errors.New("primary show boom")
		}
		return []byte("backup ok\n"), nil
	}}
	o, _ := newTestOps(t, &config.Config{}, ft)
	primary, backup, err := o.showRDPair(context.Background())
	if err == nil {
		t.Fatal("showRDPair should return the Primary's error")
	}
	if primary != "" || backup != "" {
		t.Errorf("showRDPair on error = (%q, %q), want empty strings", primary, backup)
	}
	if len(ft.outputs) != 1 {
		t.Errorf("showRDPair must not query the Backup after the Primary read fails, outputs=%v", ft.outputs)
	}
}

// --- Diagnostics mkdir failure --------------------------------------------------

// TestDiagnosticsMkdirError closes Diagnostics' setup guard: a systemic
// failure to create the destination dir must fail loud with the path in the
// message rather than silently attempting per-node gathers into a directory
// that doesn't exist.
func TestDiagnosticsMkdirError(t *testing.T) {
	blocker := filepath.Join(t.TempDir(), "blocker")
	writeFile(t, blocker, "not a directory\n")
	destDir := filepath.Join(blocker, "sub") // a path component is a regular file
	o, _ := newTestOps(t, &config.Config{}, &fakeTransport{})
	err := o.Diagnostics(context.Background(), destDir, "ts", 1, config.Primary)
	if err == nil || !strings.Contains(err.Error(), "create diagnostics dir") {
		t.Errorf("Diagnostics mkdir error = %v, want wrapped %q", err, "create diagnostics dir")
	}
}

// --- gatherNode download / bundle branches --------------------------------------

// TestGatherNodeDownloadError closes gatherNode's main-archive Download
// branch, otherwise untested anywhere in the package: a failed pull of the
// main zip must fail the whole node's diagnostics rather than reporting
// success with no local file.
func TestGatherNodeDownloadError(t *testing.T) {
	dest := filepath.Join(t.TempDir(), "diag")
	ft := &fakeTransport{responder: func(_ config.Role, argv []string, _ []byte) ([]byte, error) {
		if len(argv) == 1 && argv[0] == "hostname" {
			return []byte("host\n"), nil
		}
		return nil, nil // no "Diagnostics saved" line -> bundle branch irrelevant here
	}}
	dt := &downloadErrTransport{fakeTransport: ft, err: errors.New("download boom")}
	o := &Ops{T: dt, Cfg: &config.Config{}, Out: &bytes.Buffer{}, PollInterval: 0}
	if err := o.Diagnostics(context.Background(), dest, "20260731", 3, config.Primary); err == nil {
		t.Error("Diagnostics should surface the main archive's Download error")
	}
	zipPath := filepath.Join(dest, "gather-configs-host-20260731.zip")
	if _, statErr := os.Stat(zipPath); !os.IsNotExist(statErr) {
		t.Errorf("no local zip should exist after a failed download, stat err = %v", statErr)
	}
}

// TestGatherNodeBundleDownloadWarnsOnly closes gatherNode's one deliberately
// best-effort branch: a missing/unreadable diagnostics bundle must WARN, not
// fail the whole node's pull, since the main zip archive already succeeded.
func TestGatherNodeBundleDownloadWarnsOnly(t *testing.T) {
	dest := filepath.Join(t.TempDir(), "diag")
	ft := &fakeTransport{responder: func(_ config.Role, argv []string, _ []byte) ([]byte, error) {
		switch {
		case matchCLI(argv, "gather-configs"):
			return []byte("Diagnostics saved: logs/diag-node1.tgz\n"), nil
		case len(argv) == 1 && argv[0] == "hostname":
			return []byte("node1\n"), nil
		}
		return nil, nil
	}}
	dt := &downloadErrTransport{fakeTransport: ft, err: errors.New("bundle download boom"), match: JailRoot + "/logs/diag-node1.tgz"}
	var logs []string
	o := &Ops{T: dt, Cfg: &config.Config{}, Out: &bytes.Buffer{}, PollInterval: 0,
		Log: func(f string, a ...any) { logs = append(logs, fmt.Sprintf(f, a...)) }}
	if err := o.Diagnostics(context.Background(), dest, "20260731", 3, config.Primary); err != nil {
		t.Fatalf("Diagnostics should still succeed when only the bundle download fails: %v", err)
	}
	joined := strings.Join(logs, "\n")
	if !strings.Contains(joined, "[WARN]") || !strings.Contains(joined, "diag-node1.tgz") {
		t.Errorf("Diagnostics should warn naming the bundle, got %v", logs)
	}
}

// TestGatherNodeBundleCleanupWarnsOnly closes gatherNode's other best-effort
// branch: a failed cleanup rm of the bundle artifacts must WARN, not fail the
// whole node's pull -- losing this (e.g. someone "fixes" it to propagate)
// would break diagnostics collection on a transient cleanup hiccup.
func TestGatherNodeBundleCleanupWarnsOnly(t *testing.T) {
	dest := filepath.Join(t.TempDir(), "diag")
	ft := &fakeTransport{responder: func(_ config.Role, argv []string, _ []byte) ([]byte, error) {
		switch {
		case matchCLI(argv, "gather-configs"):
			return []byte("Diagnostics saved: logs/diag-node1.tgz\n"), nil
		case len(argv) == 1 && argv[0] == "hostname":
			return []byte("node1\n"), nil
		}
		return nil, nil
	}}
	rt := &runErrMatchTransport{fakeTransport: ft, err: errors.New("cleanup boom"), match: func(argv []string) bool {
		for _, a := range argv {
			if strings.Contains(a, "logs/diag-node1.tgz") {
				return true
			}
		}
		return false
	}}
	var logs []string
	o := &Ops{T: rt, Cfg: &config.Config{}, Out: &bytes.Buffer{}, PollInterval: 0,
		Log: func(f string, a ...any) { logs = append(logs, fmt.Sprintf(f, a...)) }}
	if err := o.Diagnostics(context.Background(), dest, "20260731", 3, config.Primary); err != nil {
		t.Fatalf("Diagnostics should still succeed when only the bundle cleanup fails: %v", err)
	}
	joined := strings.Join(logs, "\n")
	if !strings.Contains(joined, "[WARN]") || !strings.Contains(joined, "clean up diagnostics artifacts") {
		t.Errorf("Diagnostics should warn about the failed cleanup, got %v", logs)
	}
}

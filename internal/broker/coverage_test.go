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

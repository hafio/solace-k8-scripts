package broker

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"solace/internal/config"
)

// --- fake transport -------------------------------------------------------

type recUpload struct {
	role config.Role
	dest string
	data string
}
type recUploadFile struct {
	role  config.Role
	local string
	dest  string
}
type recDownload struct {
	role          config.Role
	remote, local string
}
type recRun struct {
	role config.Role
	argv []string
}
type recOutput struct {
	role  config.Role
	argv  []string
	stdin string
}

// fakeTransport records every call and answers Output/OutputInput via responder.
type fakeTransport struct {
	uploads     []recUpload
	uploadFiles []recUploadFile
	downloads   []recDownload
	runs        []recRun
	outputs     []recOutput
	responder   func(role config.Role, argv []string, stdin []byte) ([]byte, error)
}

func (f *fakeTransport) Run(_ context.Context, role config.Role, argv ...string) error {
	f.runs = append(f.runs, recRun{role, argv})
	return nil
}
func (f *fakeTransport) Output(_ context.Context, role config.Role, argv ...string) ([]byte, error) {
	f.outputs = append(f.outputs, recOutput{role, argv, ""})
	if f.responder != nil {
		return f.responder(role, argv, nil)
	}
	return nil, nil
}
func (f *fakeTransport) OutputInput(_ context.Context, role config.Role, in []byte, argv ...string) ([]byte, error) {
	f.outputs = append(f.outputs, recOutput{role, argv, string(in)})
	if f.responder != nil {
		return f.responder(role, argv, in)
	}
	return nil, nil
}
func (f *fakeTransport) Upload(_ context.Context, role config.Role, data []byte, dest string) error {
	f.uploads = append(f.uploads, recUpload{role, dest, string(data)})
	return nil
}
func (f *fakeTransport) UploadFile(_ context.Context, role config.Role, local, dest string) error {
	f.uploadFiles = append(f.uploadFiles, recUploadFile{role, local, dest})
	return nil
}
func (f *fakeTransport) Download(_ context.Context, role config.Role, remote, local string) error {
	f.downloads = append(f.downloads, recDownload{role, remote, local})
	return nil
}

// matchCLI reports whether argv is a `cli -Apes .<name>.cli` invocation.
func matchCLI(argv []string, name string) bool {
	return len(argv) == 3 && argv[0] == CLIBinary && argv[1] == "-Apes" && argv[2] == cliArg(name)
}

// uploadBody returns the last body uploaded to dest, or "" (and fails) if none.
func (f *fakeTransport) uploadBody(t *testing.T, dest string) string {
	t.Helper()
	for i := len(f.uploads) - 1; i >= 0; i-- {
		if f.uploads[i].dest == dest {
			return f.uploads[i].data
		}
	}
	t.Fatalf("no upload to %q; uploads=%v", dest, f.uploads)
	return ""
}

func (f *fakeTransport) hasUpload(dest string) bool {
	for _, u := range f.uploads {
		if u.dest == dest {
			return true
		}
	}
	return false
}

func newTestOps(t *testing.T, cfg *config.Config, ft *fakeTransport) (*Ops, *bytes.Buffer) {
	t.Helper()
	buf := &bytes.Buffer{}
	o := &Ops{T: ft, Cfg: cfg, Out: buf, PollInterval: 0, PollAttempts: 3}
	return o, buf
}

// --- pure helper unit tests -----------------------------------------------

func TestField(t *testing.T) {
	out := "Configuration Status : Enabled\r\nRedundancy Status : Up\r\nActive-Standby Role : Primary\r\n"
	cases := map[string]string{
		"Configuration Status": "Enabled",
		"Redundancy Status":    "Up",
		"Active-Standby Role":  "Primary",
		"Nonexistent Label":    "",
	}
	for label, want := range cases {
		if got := field(out, label); got != want {
			t.Errorf("field(%q) = %q, want %q", label, got, want)
		}
	}
}

func TestCountContains(t *testing.T) {
	out := "Activity Status : Local Active\r\nActivity Status : Mate Active\r\nActivity Status : Local Active\r\n"
	if got := countContains(out, "Activity Status", "Local Active"); got != 2 {
		t.Errorf("countContains Local Active = %d, want 2", got)
	}
	if got := countContains(out, "Activity Status", "Mate Active"); got != 1 {
		t.Errorf("countContains Mate Active = %d, want 1", got)
	}
}

func TestContainsAnyFold(t *testing.T) {
	if !containsAnyFold("Command FAILED to run", "error", "fail") {
		t.Error("containsAnyFold should match FAILED case-insensitively")
	}
	if containsAnyFold("all good", "error", "fail") {
		t.Error("containsAnyFold matched clean output")
	}
}

func TestValidName(t *testing.T) {
	for _, ok := range []string{"foo", "foo-bar", "foo.cli", "a_b.c-1"} {
		if err := validName("x", ok); err != nil {
			t.Errorf("validName(%q) unexpected error: %v", ok, err)
		}
	}
	for _, bad := range []string{"", "..", "a/b", "a;b", "a b", "a$b", "../x"} {
		if err := validName("x", bad); err == nil {
			t.Errorf("validName(%q) should have failed", bad)
		}
	}
}

func TestPathHelpers(t *testing.T) {
	if got := cliScriptPath("leader"); got != "/usr/sw/jail/cliscripts/.leader.cli" {
		t.Errorf("cliScriptPath = %q", got)
	}
	if got := cliArg("leader"); got != ".leader.cli" {
		t.Errorf("cliArg = %q", got)
	}
	if got := certPath("tls.crt.key"); got != "/usr/sw/jail/certs/tls.crt.key" {
		t.Errorf("certPath = %q", got)
	}
}

func TestLastLines(t *testing.T) {
	got := lastLines("a\nb\nc\nd\ne\n", 3)
	if got != "c\nd\ne\n" {
		t.Errorf("lastLines = %q", got)
	}
	if got := lastLines("only\n", 3); got != "only\n" {
		t.Errorf("lastLines fewer-than-n = %q", got)
	}
}

func TestHTTPStatusHelpers(t *testing.T) {
	if !isHTTP2xx("HTTP/1.1 200 OK") || !isHTTP2xx("HTTP/2 204") {
		t.Error("isHTTP2xx should accept 2xx")
	}
	for _, bad := range []string{"HTTP/1.1 401 Unauthorized", "HTTP/1.1 500", "", "garbage"} {
		if isHTTP2xx(bad) {
			t.Errorf("isHTTP2xx(%q) should be false", bad)
		}
	}
	lines := httpStatusLines("HTTP/1.1 100 Continue\r\nHTTP/1.1 200 OK\r\n\r\nbody\r\n")
	if len(lines) != 2 || lines[1] != "HTTP/1.1 200 OK" {
		t.Errorf("httpStatusLines = %v", lines)
	}
}

func TestPrimaryRedundancyUp(t *testing.T) {
	up := "Configuration Status : Enabled\nRedundancy Status : Up\nActive-Standby Role : Primary\nADB Link To Mate : Up\nADB Hello To Mate : Up\n"
	if !primaryRedundancyUp(up) {
		t.Error("primaryRedundancyUp should be true for a healthy Primary")
	}
	down := strings.Replace(up, "Redundancy Status : Up", "Redundancy Status : Down", 1)
	if primaryRedundancyUp(down) {
		t.Error("primaryRedundancyUp should be false when redundancy is Down")
	}
}

// --- RunCLI primitive ------------------------------------------------------

func TestRunCLIUploadsThenExecs(t *testing.T) {
	ft := &fakeTransport{responder: func(_ config.Role, argv []string, _ []byte) ([]byte, error) {
		if matchCLI(argv, "probe") {
			return []byte("output\n"), nil
		}
		return nil, nil
	}}
	o, _ := newTestOps(t, &config.Config{}, ft)
	out, err := o.RunCLI(context.Background(), config.Primary, "probe", "show version\n")
	if err != nil {
		t.Fatalf("RunCLI error: %v", err)
	}
	if string(out) != "output\n" {
		t.Errorf("RunCLI output = %q", out)
	}
	if body := ft.uploadBody(t, cliScriptPath("probe")); body != "show version\n" {
		t.Errorf("uploaded body = %q", body)
	}
	if len(ft.outputs) != 1 || !matchCLI(ft.outputs[0].argv, "probe") {
		t.Errorf("RunCLI exec argv = %v", ft.outputs)
	}
}

func TestRunCLIRejectsBadName(t *testing.T) {
	ft := &fakeTransport{}
	o, _ := newTestOps(t, &config.Config{}, ft)
	if _, err := o.RunCLI(context.Background(), config.Primary, "../evil", "body"); err == nil {
		t.Error("RunCLI should reject an invalid name")
	}
	if len(ft.uploads) != 0 {
		t.Error("RunCLI must not upload when the name is invalid")
	}
}

func TestSkipIfStandalone(t *testing.T) {
	ha, _ := newTestOps(t, &config.Config{Redundancy: "yes"}, &fakeTransport{})
	if ha.skipIfStandalone("x") {
		t.Error("skipIfStandalone should be false in HA")
	}
	sa, _ := newTestOps(t, &config.Config{Redundancy: "no"}, &fakeTransport{})
	if !sa.skipIfStandalone("x") {
		t.Error("skipIfStandalone should be true for standalone")
	}
}

// --- config ops ------------------------------------------------------------

func TestServerCert(t *testing.T) {
	dir := t.TempDir()
	key := filepath.Join(dir, "tls.key")
	crt := filepath.Join(dir, "tls.crt")
	ca := filepath.Join(dir, "ca.pem")
	writeFile(t, key, "KEY\n")
	writeFile(t, crt, "CERT\n")
	writeFile(t, ca, "CA\n")

	cfg := &config.Config{TLS: config.TLS{Cert: crt, CertKey: key, CAs: []string{ca}}}
	ft := &fakeTransport{}
	o, _ := newTestOps(t, cfg, ft)
	if err := o.ServerCert(context.Background(), "2026-07-31", config.Primary); err != nil {
		t.Fatalf("ServerCert error: %v", err)
	}
	dest := certPath(serverCertFile("2026-07-31"))
	if body := ft.uploadBody(t, dest); body != "KEY\nCERT\nCA\n" {
		t.Errorf("cert bundle = %q, want key+cert+ca", body)
	}
	if body := ft.uploadBody(t, cliScriptPath("apply-server-certs")); body != serverCertScript("2026-07-31") {
		t.Errorf("apply-server-certs body = %q", body)
	}
}

func TestServerCertRequiresCert(t *testing.T) {
	o, _ := newTestOps(t, &config.Config{}, &fakeTransport{})
	if err := o.ServerCert(context.Background(), "2026-07-31", config.Primary); err == nil {
		t.Error("ServerCert should error when tls.cert/certKey are unset")
	}
}

func TestDomainCerts(t *testing.T) {
	ft := &fakeTransport{}
	o, _ := newTestOps(t, &config.Config{}, ft)
	files := map[string]string{"myca": "myca.pem"}
	if err := o.DomainCerts(context.Background(), config.Primary, "certs", files); err != nil {
		t.Fatalf("DomainCerts error: %v", err)
	}
	if len(ft.uploadFiles) != 1 ||
		ft.uploadFiles[0].local != filepath.Join("certs", "myca.pem") ||
		ft.uploadFiles[0].dest != certPath("myca.pem") {
		t.Errorf("DomainCerts uploadFiles = %v", ft.uploadFiles)
	}
	if body := ft.uploadBody(t, cliScriptPath("load-domain-certs")); body != domainCertsScript(files) {
		t.Errorf("load-domain-certs body = %q", body)
	}
}

func TestDomainCertsRejectsBadName(t *testing.T) {
	ft := &fakeTransport{}
	o, _ := newTestOps(t, &config.Config{}, ft)
	if err := o.DomainCerts(context.Background(), config.Primary, "certs", map[string]string{"bad name": "x.pem"}); err == nil {
		t.Error("DomainCerts should reject a CA name with a space")
	}
	if len(ft.uploadFiles) != 0 {
		t.Error("DomainCerts must not upload before validating the name")
	}
}

func TestDomainCertsEmptySkips(t *testing.T) {
	ft := &fakeTransport{}
	o, _ := newTestOps(t, &config.Config{}, ft)
	if err := o.DomainCerts(context.Background(), config.Primary, "certs", nil); err != nil {
		t.Fatalf("DomainCerts empty error: %v", err)
	}
	if len(ft.uploadFiles) != 0 || len(ft.outputs) != 0 {
		t.Error("DomainCerts with no files should make no calls")
	}
}

func TestDisableDefaultVPN(t *testing.T) {
	ft := &fakeTransport{responder: func(_ config.Role, argv []string, _ []byte) ([]byte, error) {
		if matchCLI(argv, "show-vpn") {
			return []byte("vpn list\n"), nil
		}
		return nil, nil
	}}
	o, _ := newTestOps(t, &config.Config{}, ft)
	if err := o.DisableDefaultVPN(context.Background(), config.Primary); err != nil {
		t.Fatalf("DisableDefaultVPN error: %v", err)
	}
	if body := ft.uploadBody(t, cliScriptPath("disable-default-vpn")); body != disableDefaultVPNScript() {
		t.Errorf("disable-default-vpn body mismatch")
	}
	// The two uploaded scripts are cleaned up with a single rm -f.
	if !ranContains(ft, "rm", "-f") {
		t.Error("DisableDefaultVPN should clean up its cli scripts")
	}
}

func TestDisableDefaultUsers(t *testing.T) {
	row := func(name string) string { return name + strings.Repeat(" ", 40-len(name)) + "Yes" }
	list := strings.Join([]string{
		strings.Repeat("-", 40),
		row("default"),
		row("myvpn"),
	}, "\r\n") + "\r\n"
	ft := &fakeTransport{responder: func(_ config.Role, argv []string, _ []byte) ([]byte, error) {
		if matchCLI(argv, "show-vpn") {
			return []byte(list), nil
		}
		return nil, nil
	}}
	o, _ := newTestOps(t, &config.Config{}, ft)
	if err := o.DisableDefaultUsers(context.Background(), config.Primary); err != nil {
		t.Fatalf("DisableDefaultUsers error: %v", err)
	}
	body := ft.uploadBody(t, cliScriptPath("disable-default-usernames"))
	for _, want := range []string{
		`client-username default message-vpn "default"`,
		`client-username default message-vpn "myvpn"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("disable-default-usernames body missing %q", want)
		}
	}
}

func TestDisableDefaultUsersNoVPNs(t *testing.T) {
	ft := &fakeTransport{responder: func(_ config.Role, _ []string, _ []byte) ([]byte, error) {
		return []byte("no separator, nothing to parse\n"), nil
	}}
	o, _ := newTestOps(t, &config.Config{}, ft)
	if err := o.DisableDefaultUsers(context.Background(), config.Primary); err != nil {
		t.Fatalf("DisableDefaultUsers error: %v", err)
	}
	if ft.hasUpload(cliScriptPath("disable-default-usernames")) {
		t.Error("DisableDefaultUsers should not run when no VPNs are parsed")
	}
}

func TestProductKeys(t *testing.T) {
	ft := &fakeTransport{responder: func(_ config.Role, argv []string, _ []byte) ([]byte, error) {
		if matchCLI(argv, "product-keys") {
			return []byte("Product key applied.\n"), nil
		}
		return nil, nil
	}}
	o, _ := newTestOps(t, &config.Config{}, ft)
	if err := o.ProductKeys(context.Background(), []string{"KEY-1"}, config.Primary); err != nil {
		t.Fatalf("ProductKeys error: %v", err)
	}
	if body := ft.uploadBody(t, cliScriptPath("product-keys")); body != productKeysScript([]string{"KEY-1"}) {
		t.Errorf("product-keys body = %q", body)
	}
}

func TestProductKeysDetectsError(t *testing.T) {
	ft := &fakeTransport{responder: func(_ config.Role, _ []string, _ []byte) ([]byte, error) {
		return []byte("Command failed: invalid key\n"), nil
	}}
	o, _ := newTestOps(t, &config.Config{}, ft)
	if err := o.ProductKeys(context.Background(), []string{"BAD"}, config.Primary); err == nil {
		t.Error("ProductKeys should fail loud when the broker reports an error")
	}
}

func TestProductKeysEmpty(t *testing.T) {
	o, _ := newTestOps(t, &config.Config{}, &fakeTransport{})
	if err := o.ProductKeys(context.Background(), nil, config.Primary); err == nil {
		t.Error("ProductKeys should error with no keys")
	}
}

func TestExecCLI(t *testing.T) {
	local := filepath.Join(t.TempDir(), "myscript.cli")
	ft := &fakeTransport{responder: func(_ config.Role, argv []string, _ []byte) ([]byte, error) {
		if matchCLI(argv, "myscript.cli") {
			return []byte("OK\n"), nil
		}
		return nil, nil
	}}
	o, _ := newTestOps(t, &config.Config{}, ft)
	if err := o.ExecCLI(context.Background(), config.Primary, local); err != nil {
		t.Fatalf("ExecCLI error: %v", err)
	}
	if len(ft.uploadFiles) != 1 || ft.uploadFiles[0].dest != cliScriptPath("myscript.cli") || ft.uploadFiles[0].local != local {
		t.Errorf("ExecCLI uploadFiles = %v", ft.uploadFiles)
	}
	if !ranContains(ft, "rm", "-f") {
		t.Error("ExecCLI should clean up the uploaded script")
	}
}

func TestExecCLIRejectsBadName(t *testing.T) {
	ft := &fakeTransport{}
	o, _ := newTestOps(t, &config.Config{}, ft)
	if err := o.ExecCLI(context.Background(), config.Primary, "/some/dir/.."); err == nil {
		t.Error("ExecCLI should reject a base name of '..'")
	}
	if len(ft.uploadFiles) != 0 {
		t.Error("ExecCLI must not upload an invalid-named script")
	}
}

// --- verify ops ------------------------------------------------------------

func TestLogin(t *testing.T) {
	ft := &fakeTransport{responder: func(_ config.Role, _ []string, _ []byte) ([]byte, error) {
		return []byte("HTTP/1.1 200 OK\r\nContent-Type: application/json\r\n\r\n{}\r\n"), nil
	}}
	o, buf := newTestOps(t, &config.Config{}, ft)
	ok, err := o.Login(context.Background(), config.Primary, "admin", "s3cret")
	if err != nil || !ok {
		t.Fatalf("Login ok=%v err=%v", ok, err)
	}
	if !strings.Contains(buf.String(), "Login OK") {
		t.Errorf("Login output = %q", buf.String())
	}
	// The password must ride stdin, never the argv.
	if ft.outputs[0].stdin != "user = \"admin:s3cret\"\n" {
		t.Errorf("Login stdin = %q", ft.outputs[0].stdin)
	}
	if strings.Contains(strings.Join(ft.outputs[0].argv, " "), "s3cret") {
		t.Error("password leaked into curl argv")
	}
}

func TestLoginFailure(t *testing.T) {
	ft := &fakeTransport{responder: func(_ config.Role, _ []string, _ []byte) ([]byte, error) {
		return []byte("HTTP/1.1 401 Unauthorized\r\n"), nil
	}}
	o, buf := newTestOps(t, &config.Config{}, ft)
	ok, err := o.Login(context.Background(), config.Primary, "admin", "bad")
	if err != nil || ok {
		t.Fatalf("Login ok=%v err=%v, want false/nil", ok, err)
	}
	if !strings.Contains(buf.String(), "401") {
		t.Errorf("Login failure output = %q", buf.String())
	}
}

func TestLoginNoResponse(t *testing.T) {
	ft := &fakeTransport{responder: func(_ config.Role, _ []string, _ []byte) ([]byte, error) {
		return []byte(""), nil
	}}
	o, buf := newTestOps(t, &config.Config{}, ft)
	ok, _ := o.Login(context.Background(), config.Primary, "admin", "x")
	if ok || !strings.Contains(buf.String(), "no HTTP response") {
		t.Errorf("Login no-response ok=%v out=%q", ok, buf.String())
	}
}

func TestLeaderStandaloneSkips(t *testing.T) {
	ft := &fakeTransport{}
	o, _ := newTestOps(t, &config.Config{Redundancy: "no"}, ft)
	if err := o.Leader(context.Background()); err != nil {
		t.Fatalf("Leader standalone error: %v", err)
	}
	if len(ft.outputs) != 0 || len(ft.uploads) != 0 || len(ft.runs) != 0 {
		t.Error("Leader must make no calls in standalone mode")
	}
}

func TestLeaderSuccess(t *testing.T) {
	healthy := "Configuration Status : Enabled\nRedundancy Status : Up\nActive-Standby Role : Primary\nADB Link To Mate : Up\nADB Hello To Mate : Up\n"
	ft := &fakeTransport{responder: func(_ config.Role, argv []string, _ []byte) ([]byte, error) {
		switch {
		case matchCLI(argv, "show-rd"):
			return []byte(healthy), nil
		case matchCLI(argv, "assert-leader"):
			return []byte("l1\nl2\nSync Complete\n"), nil
		}
		return nil, nil
	}}
	o, buf := newTestOps(t, &config.Config{Redundancy: "yes"}, ft)
	if err := o.Leader(context.Background()); err != nil {
		t.Fatalf("Leader error: %v", err)
	}
	// revert-activity is run against the Backup, assert-leader against the Primary.
	if !uploadedForRole(ft, config.Backup, cliScriptPath("revert-activity")) {
		t.Error("Leader should revert activity on the Backup")
	}
	if !uploadedForRole(ft, config.Primary, cliScriptPath("assert-leader")) {
		t.Error("Leader should assert leadership on the Primary")
	}
	if !strings.Contains(buf.String(), "Sync Complete") {
		t.Errorf("Leader output = %q", buf.String())
	}
}

func TestLeaderTimeout(t *testing.T) {
	down := "Configuration Status : Enabled\nRedundancy Status : Down\nActive-Standby Role : Primary\nADB Link To Mate : Up\nADB Hello To Mate : Up\n"
	sawDetail := false
	ft := &fakeTransport{responder: func(_ config.Role, argv []string, _ []byte) ([]byte, error) {
		if matchCLI(argv, "show-redundancy-detail") {
			sawDetail = true
			return []byte("detail dump\n"), nil
		}
		return []byte(down), nil
	}}
	o, _ := newTestOps(t, &config.Config{Redundancy: "yes"}, ft)
	o.PollAttempts = 2
	if err := o.Leader(context.Background()); err == nil {
		t.Error("Leader should time out when redundancy never recovers")
	}
	if !sawDetail {
		t.Error("Leader should dump show-redundancy-detail on timeout")
	}
}

func TestRedundancySuccess(t *testing.T) {
	adb := "ADB Link To Mate : Up\nADB Hello To Mate : Up\n"
	base := func(role, cfg, rdc, act string) string {
		return "Configuration Status : " + cfg + "\nRedundancy Status : " + rdc +
			"\nActive-Standby Role : " + role + "\n" + adb + "Activity Status : " + act + "\n"
	}
	priSeq := []string{
		base("Primary", "Enabled", "Up", "Local Active"),           // a: initial, primary active
		base("Primary", "Enabled-Released", "Down", "Mate Active"), // c: released
		base("Primary", "Enabled", "Up", "Mate Active"),            // e: un-released (mate active)
		base("Primary", "Enabled", "Up", "Local Active"),           // h: reverted (local active)
	}
	bkSeq := []string{
		base("Backup", "Enabled", "Up", "Local Active"), // e: backup active
		base("Backup", "Enabled", "Up", "Local Active"), // f: revert precheck
		base("Backup", "Enabled", "Up", "Mate Active"),  // h: reverted (not active)
	}
	var pri, bk int
	ft := &fakeTransport{responder: func(role config.Role, argv []string, _ []byte) ([]byte, error) {
		if !matchCLI(argv, "show-rd") {
			return nil, nil // release / no-release / revert-activity execs
		}
		if role == config.Primary {
			out := priSeq[pri]
			pri++
			return []byte(out), nil
		}
		out := bkSeq[bk]
		bk++
		return []byte(out), nil
	}}
	o, _ := newTestOps(t, &config.Config{Redundancy: "yes"}, ft)
	if err := o.Redundancy(context.Background()); err != nil {
		t.Fatalf("Redundancy error: %v", err)
	}
	if pri != len(priSeq) || bk != len(bkSeq) {
		t.Errorf("Redundancy consumed pri=%d/%d bk=%d/%d", pri, len(priSeq), bk, len(bkSeq))
	}
}

func TestRedundancyStandaloneSkips(t *testing.T) {
	ft := &fakeTransport{}
	o, _ := newTestOps(t, &config.Config{Redundancy: "no"}, ft)
	if err := o.Redundancy(context.Background()); err != nil {
		t.Fatalf("Redundancy standalone error: %v", err)
	}
	if len(ft.outputs) != 0 {
		t.Error("Redundancy must make no calls in standalone mode")
	}
}

func TestDiagnostics(t *testing.T) {
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
	o, _ := newTestOps(t, &config.Config{}, ft)
	if err := o.Diagnostics(context.Background(), dest, "20260731", 3, config.Primary); err != nil {
		t.Fatalf("Diagnostics error: %v", err)
	}
	if _, err := os.Stat(dest); err != nil {
		t.Errorf("Diagnostics did not create dest dir: %v", err)
	}
	if body := ft.uploadBody(t, cliScriptPath("gather-configs")); body != gatherConfigsScript(3) {
		t.Error("gather-configs body mismatch")
	}
	wantDownloads := map[string]string{
		JailRoot + "/gather-configs.zip":  filepath.Join(dest, "gather-configs-node1-20260731.zip"),
		JailRoot + "/logs/diag-node1.tgz": filepath.Join(dest, "diag-node1.tgz"),
	}
	if len(ft.downloads) != len(wantDownloads) {
		t.Fatalf("Diagnostics downloads = %v", ft.downloads)
	}
	for _, d := range ft.downloads {
		if want, ok := wantDownloads[d.remote]; !ok || want != d.local {
			t.Errorf("unexpected download %+v", d)
		}
	}
}

// --- test helpers ----------------------------------------------------------

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func ranContains(ft *fakeTransport, argv ...string) bool {
	for _, r := range ft.runs {
		if len(r.argv) >= len(argv) {
			match := true
			for i := range argv {
				if r.argv[i] != argv[i] {
					match = false
					break
				}
			}
			if match {
				return true
			}
		}
	}
	return false
}

func uploadedForRole(ft *fakeTransport, role config.Role, dest string) bool {
	for _, u := range ft.uploads {
		if u.role == role && u.dest == dest {
			return true
		}
	}
	return false
}

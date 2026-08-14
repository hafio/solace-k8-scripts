package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"solace/internal/config"
)

// sampleEnv is the shared fixture: the user template doubles as a valid config
// for k8s, docker, and podman (same file render_test.go renders from). During
// `go test ./internal/cli` the cwd is the package dir, so this relative path
// resolves to the repo's env/sample.yaml.
const sampleEnv = "../../env/sample.yaml"

// withEnv appends the sample --env so a command reaching PersistentPreRunE loads
// a valid config instead of the missing env.yaml. The value carries a separator,
// so it resolves verbatim rather than through the base-dir/env lookup.
func withEnv(args ...string) []string {
	return append(append([]string{}, args...), "--env", sampleEnv)
}

// smokeAdminPass is a distinctive admin password used only by the standalone test
// env, so TestSecretsNeverEchoed can prove it never reaches stdout.
const smokeAdminPass = "SMOKE-PW-do-not-log-1234"

// writeStandaloneEnv writes a minimal single-broker (redundancy: no) env to a temp
// file and returns its path. The HA sample sets tls.serverSecret with absent cert
// files and defines all three nodes, which makes the secret-bearing prep steps fail
// and the HA-only config/verify steps poll or exercise failover -- unsuitable for a
// clean --dry-run pass. This env has no TLS and no nodes, so those steps self-skip.
func writeStandaloneEnv(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "standalone.yaml")
	content := "redundancy: no\n" +
		"image:\n" +
		"  repo: solace-pubsub-standard\n" +
		"  tag: \"10.10.1.128\"\n" +
		"admin:\n" +
		"  pass: " + smokeAdminPass + "\n" +
		"k8s:\n" +
		"  name: dev-broker\n" +
		"  namespace: solace\n" +
		"  adminSecret: solace-admin-secret\n" +
		"  updateStrategy: automatedRolling\n" +
		"  storage:\n" +
		"    class: standard\n" +
		"    msgNode: 30Gi\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write standalone env: %v", err)
	}
	return path
}

// runStandalone runs a k8s command under --dry-run against the standalone env.
func runStandalone(t *testing.T, path string, args ...string) (string, error) {
	t.Helper()
	full := append(append([]string{}, args...), "--dry-run", "--env", path)
	return runRoot(t, full)
}

// runCtr runs a container (docker/podman) command under --dry-run against the
// given env path. It is the container-facing name for the same "append
// --dry-run --env" run that runStandalone performs, kept distinct for
// readability at container call sites (against both the HA sample and a
// container-standalone env).
func runCtr(t *testing.T, path string, args ...string) (string, error) {
	t.Helper()
	return runStandalone(t, path, args...)
}

// writeCtrStandaloneEnv writes a minimal single-broker (redundancy: no) env that is
// valid for docker and podman: only nodes.primary.name is mandatory (data dir,
// network mode, and docker.mode all default). It carries no TLS, domainCerts, or
// productKeys, so the config steps that gate on those self-skip and none of them
// reach a poll loop -- keeping every dry-run pass fast and deterministic. The
// k8s-shaped writeStandaloneEnv cannot be reused: it has no nodes: block, which
// fails container validation (nodes.primary.name is required).
func writeCtrStandaloneEnv(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "ctr-standalone.yaml")
	content := "redundancy: no\n" +
		"image:\n" +
		"  repo: solace-pubsub-standard\n" +
		"  tag: \"10.10.1.128\"\n" +
		"admin:\n" +
		"  pass: " + smokeAdminPass + "\n" +
		"nodes:\n" +
		"  primary:\n" +
		"    name: pri-host\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write container standalone env: %v", err)
	}
	return path
}

// loadDirect writes yamlBody to a temp file and loads it for platform p, for
// tests that need a *config.Config to build an App directly (bypassing
// config.Load's implicit --dry-run wiring in App.load) so a fake Runner other
// than engine.Echo can be attached -- see opRunner below.
func loadDirect(t *testing.T, yamlBody string, p config.Platform) *config.Config {
	t.Helper()
	path := filepath.Join(t.TempDir(), "direct.yaml")
	if err := os.WriteFile(path, []byte(yamlBody), 0o644); err != nil {
		t.Fatalf("write direct env: %v", err)
	}
	cfg, err := config.Load(path, p)
	if err != nil {
		t.Fatalf("load direct env: %v", err)
	}
	return cfg
}

// --- fake engine.Runner for direct op-function calls ------------------------
//
// opK8sConfigAll/opK8sUp/opK8sPrepAll/opK8sDown/opCtrConfigAll etc. are unexported
// opFunc-shaped handlers that take only an *App, so a test in this package can call
// them directly rather than through cobra/runRoot. Doing so is what makes two
// things possible that engine.Echo cannot provide: (1) targeted fault injection --
// Echo's Run/Output methods never return an error, so the ordering property "step
// N fails, step N+1 never runs" has no double to exercise it without this, mirroring
// internal/container's capRunner/failOn (transport_test.go); and (2) canned Output
// content -- Echo always returns (nil, nil), which makes a poll-based HA state
// machine (config-sync leader, redundancy) exhaust its real PollInterval x
// PollAttempts budget (broker.New's 2s x 60 defaults) before timing out, since the
// CLI layer has no seam to shorten it. Supplying a healthy transcript lets the
// state machine succeed on its first check instead.
type opCall struct {
	method string // Run | RunInput | RunEnv | RunInteractive | Output | OutputInput
	name   string
	args   []string
	stdin  string // RunInput/OutputInput only
}

// opRunner is a fake engine.Runner. fail, when set, is consulted for every call
// after it is recorded: a non-nil return is propagated as that command's error. It
// receives the whole opCall (not just name/args, unlike container's failOn) so a
// test can target a call by its stdin body as well as its argv -- see
// failDisableDefaultUsersUpload, which needs exactly that to disambiguate two
// steps that issue the identical script name. output, when set, supplies
// Output/OutputInput's returned bytes whenever fail did not fire.
type opRunner struct {
	calls  []opCall
	fail   func(opCall) error
	output func(opCall) []byte
}

func (r *opRunner) Run(_ context.Context, name string, args ...string) error {
	return r.do(opCall{method: "Run", name: name, args: args})
}
func (r *opRunner) RunInput(_ context.Context, in []byte, name string, args ...string) error {
	return r.do(opCall{method: "RunInput", name: name, args: args, stdin: string(in)})
}
func (r *opRunner) RunEnv(_ context.Context, _ []string, name string, args ...string) error {
	return r.do(opCall{method: "RunEnv", name: name, args: args})
}
func (r *opRunner) RunInteractive(_ context.Context, name string, args ...string) error {
	return r.do(opCall{method: "RunInteractive", name: name, args: args})
}
func (r *opRunner) do(c opCall) error {
	r.calls = append(r.calls, c)
	if r.fail != nil {
		return r.fail(c)
	}
	return nil
}

// opCanI reports whether a recorded call is the k8s read-only permission probe
// (Cluster.Preflight's `auth can-i`), which every mutating operation issues first.
// The double answers it "yes" out of band so the op-level tests below stay about
// the work they were written for; a test that needs a refusal sets output/fail for
// this call itself, as TestOpK8sDeployStopsOnPreflightRefusal does.
func opCanI(c opCall) bool {
	for _, a := range c.args {
		if a == "can-i" {
			return true
		}
	}
	return false
}

func (r *opRunner) Output(_ context.Context, name string, args ...string) ([]byte, error) {
	c := opCall{method: "Output", name: name, args: args}
	r.calls = append(r.calls, c)
	if r.fail != nil {
		if err := r.fail(c); err != nil {
			return nil, err
		}
	}
	if r.output != nil {
		if out := r.output(c); out != nil {
			return out, nil
		}
	}
	if opCanI(c) {
		return []byte("yes\n"), nil
	}
	return nil, nil
}
func (r *opRunner) OutputInput(_ context.Context, in []byte, name string, args ...string) ([]byte, error) {
	c := opCall{method: "OutputInput", name: name, args: args, stdin: string(in)}
	r.calls = append(r.calls, c)
	if r.fail != nil {
		if err := r.fail(c); err != nil {
			return nil, err
		}
	}
	if r.output != nil {
		return r.output(c), nil
	}
	return nil, nil
}

// opArgvMatch reports whether substr appears in c's command name or any of its args.
func opArgvMatch(c opCall, substr string) bool {
	if strings.Contains(c.name, substr) {
		return true
	}
	for _, a := range c.args {
		if strings.Contains(a, substr) {
			return true
		}
	}
	return false
}

// hasCall reports whether any recorded call's name or args contain substr.
func (r *opRunner) hasCall(substr string) bool {
	for _, c := range r.calls {
		if opArgvMatch(c, substr) {
			return true
		}
	}
	return false
}

// dump renders the recorded call sequence for a failure message: which command
// ran, in what order, is the only thing that distinguishes "the abort worked" from
// "the injected failure never landed". Stdin is deliberately omitted -- several of
// these uploads carry passwords.
func (r *opRunner) dump() string {
	var b strings.Builder
	for i, c := range r.calls {
		fmt.Fprintf(&b, "  %2d %-11s %s %s\n", i+1, c.method, c.name, strings.Join(c.args, " "))
	}
	return b.String()
}

// callCount counts recorded calls whose name or args contain substr.
func (r *opRunner) callCount(substr string) int {
	n := 0
	for _, c := range r.calls {
		if opArgvMatch(c, substr) {
			n++
		}
	}
	return n
}

// opFailOn builds an opRunner.fail hook that errors on every call whose command
// name or args contain substr -- for a marker that is unique to one step, where
// failing its first (chronological) match aborts before any later call is made.
func opFailOn(substr string) func(opCall) error {
	return func(c opCall) error {
		if opArgvMatch(c, substr) {
			return fmt.Errorf("injected failure for %q", substr)
		}
		return nil
	}
}

// opFailOnCount builds a fail hook that errors only on the nth (1-indexed) call
// whose name or args contain substr, succeeding every other one -- for
// opK8sUp/opK8sDown's repeated, textually-identical steps (every `apply -f -` /
// `delete ... --ignore-not-found` call looks the same; only its position in the
// sequence identifies which step it belongs to).
func opFailOnCount(substr string, n int) func(opCall) error {
	count := 0
	return func(c opCall) error {
		if !opArgvMatch(c, substr) {
			return nil
		}
		count++
		if count == n {
			return fmt.Errorf("injected failure for %q (call #%d)", substr, n)
		}
		return nil
	}
}

// failDisableDefaultUsersUpload targets DisableDefaultUsers' own "show-vpn" probe
// without also aborting DisableDefaultVPN's: both steps run RunCLI with the
// identical script name "show-vpn" (broker/config_ops.go), so their argv is
// indistinguishable. What differs is the uploaded body: DisableDefaultVPN's own
// closing probe uploads the interactive-mode script (showVPNScript, which opens
// "home\nenable\nconfigure"), while DisableDefaultUsers uploads the bare one
// (showVPNBareScript, just "show message-vpn *"). Matching on the upload's stdin --
// invisible to argv-only matching -- is what makes the two distinguishable at all.
func failDisableDefaultUsersUpload(c opCall) error {
	if c.method == "RunInput" && opArgvMatch(c, ".show-vpn.cli") && !strings.Contains(c.stdin, "configure") {
		return fmt.Errorf("injected failure for disable-default-users' show-vpn upload")
	}
	return nil
}

// healthyShowRD is a canned `show redundancy` transcript satisfying
// primaryRedundancyUp (internal/broker/verify_ops.go), so a direct-call test can
// drive Leader's poll to succeed on the first check.
const healthyShowRD = "Configuration Status: Enabled\n" +
	"Redundancy Status: Up\n" +
	"Active-Standby Role: Primary\n" +
	"ADB Link To Mate: Up\n" +
	"ADB Hello To Mate: Up\n"

// capture redirects the given standard stream (os.Stdout or os.Stderr) through a
// pipe for the duration of fn and returns everything written.
func capture(t *testing.T, target **os.File, fn func()) string {
	t.Helper()
	old := *target
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	*target = w
	type res struct {
		b []byte
		e error
	}
	ch := make(chan res, 1)
	go func() {
		b, e := io.ReadAll(r)
		ch <- res{b, e}
	}()
	fn()
	if err := w.Close(); err != nil {
		t.Fatalf("close pipe: %v", err)
	}
	*target = old
	out := <-ch
	if out.e != nil {
		t.Fatalf("read pipe: %v", out.e)
	}
	return string(out.b)
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	return capture(t, &os.Stdout, fn)
}

func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	return capture(t, &os.Stderr, fn)
}

// runRoot builds a fresh command tree with a fresh App, runs it with args, and
// returns captured stdout plus the Execute error. A fresh App per call means no
// flag scratch state leaks between cases.
func runRoot(t *testing.T, args []string) (string, error) {
	t.Helper()
	return runRootWith(t, args, nil)
}

// runRootWith is runRoot, but lets a test configure the App before Execute --
// e.g. setting Interactive/PromptIn to deterministically drive the confirm-prompt
// branches (confirmDelete/confirmPurge/confirmRestart, the placement-labelling
// gate in opK8sUp) that a real terminal would otherwise gate on the test
// process's own, environment-dependent stdin. configure may be nil, in which case
// this is exactly runRoot.
func runRootWith(t *testing.T, args []string, configure func(*App)) (string, error) {
	t.Helper()
	var runErr error
	out := captureStdout(t, func() {
		app := &App{}
		if configure != nil {
			configure(app)
		}
		root := newRootCmd(app)
		root.SetArgs(args)
		root.SetOut(io.Discard)
		root.SetErr(io.Discard)
		runErr = root.Execute()
	})
	return out, runErr
}

// findCmd walks the tree from root by successive command names.
func findCmd(t *testing.T, root *cobra.Command, path ...string) *cobra.Command {
	t.Helper()
	cur := root
	for _, name := range path {
		var next *cobra.Command
		for _, c := range cur.Commands() {
			if c.Name() == name {
				next = c
				break
			}
		}
		if next == nil {
			t.Fatalf("command %q not found under %q", name, cur.CommandPath())
		}
		cur = next
	}
	return cur
}

func collectPaths(c *cobra.Command, acc *[]string) {
	*acc = append(*acc, c.CommandPath())
	for _, sub := range c.Commands() {
		collectPaths(sub, acc)
	}
}

// runStatusStderr runs `k8s status --dry-run` with the given flags and returns
// what reached stderr, where the resolved-env-file line is echoed.
func runStatusStderr(t *testing.T, args ...string) (string, error) {
	t.Helper()
	var runErr error
	errOut := captureStderr(t, func() {
		_, runErr = runRoot(t, append([]string{"k8s", "status", "--dry-run"}, args...))
	})
	return errOut, runErr
}

// TestEnvFileLookup covers the resolver as the CLI wires it: -e names a file,
// searched in the base dir then <base-dir>/env, and the winner is echoed so a
// base-dir copy shadowing the env/ copy is visible.
func TestEnvFileLookup(t *testing.T) {
	body, err := os.ReadFile(writeStandaloneEnv(t))
	if err != nil {
		t.Fatalf("read standalone env: %v", err)
	}
	root := t.TempDir()
	put := func(rel string) string {
		t.Helper()
		p := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatalf("MkdirAll: %v", err)
		}
		if err := os.WriteFile(p, body, 0o600); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}
		return p
	}

	t.Run("falls back to the env dir", func(t *testing.T) {
		want := put("env/dev.yaml")
		out, err := runStatusStderr(t, "--base-dir", root, "-e", "dev.yaml")
		if err != nil {
			t.Fatalf("status err = %v", err)
		}
		if !strings.Contains(out, "==> env file: "+want) {
			t.Errorf("stderr = %q, want the env file line for %q", out, want)
		}
	})

	t.Run("base dir shadows the env dir", func(t *testing.T) {
		want := put("dev.yaml")
		out, err := runStatusStderr(t, "--base-dir", root, "-e", "dev.yaml")
		if err != nil {
			t.Fatalf("status err = %v", err)
		}
		if !strings.Contains(out, "==> env file: "+want) {
			t.Errorf("stderr = %q, want the env file line for %q", out, want)
		}
	})

	t.Run("no extension is inferred", func(t *testing.T) {
		_, err := runStatusStderr(t, "--base-dir", root, "-e", "dev")
		if err == nil || !strings.Contains(err.Error(), `env file "dev" not found`) {
			t.Fatalf("-e dev err = %v, want a not-found error", err)
		}
	})

	t.Run("long and short flags agree", func(t *testing.T) {
		want := filepath.Join(root, "dev.yaml")
		out, err := runStatusStderr(t, "--base-dir", root, "--env", "dev.yaml")
		if err != nil {
			t.Fatalf("status err = %v", err)
		}
		if !strings.Contains(out, "==> env file: "+want) {
			t.Errorf("stderr = %q, want the env file line for %q", out, want)
		}
	})
}

func TestFirstArg(t *testing.T) {
	if got := firstArg(nil); got != "" {
		t.Errorf("firstArg(nil) = %q, want empty", got)
	}
	if got := firstArg([]string{"a", "b"}); got != "a" {
		t.Errorf("firstArg([a b]) = %q, want a", got)
	}
}

func TestFirstArgOr(t *testing.T) {
	cases := []struct {
		name string
		args []string
		def  string
		want string
	}{
		{"empty args -> def", nil, "broker", "broker"},
		{"empty first elem -> def", []string{""}, "broker", "broker"},
		{"non-empty -> arg", []string{"operator"}, "broker", "operator"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := firstArgOr(tc.args, tc.def); got != tc.want {
				t.Errorf("firstArgOr(%q, %q) = %q, want %q", tc.args, tc.def, got, tc.want)
			}
		})
	}
}

func TestNotImplemented(t *testing.T) {
	err := notImplemented("k8s foo")
	if err == nil {
		t.Fatal("notImplemented returned nil")
	}
	msg := err.Error()
	if !strings.Contains(msg, "k8s foo") || !strings.Contains(msg, "not implemented yet") {
		t.Errorf("notImplemented message = %q, want name + 'not implemented yet'", msg)
	}
}

func TestEmit(t *testing.T) {
	var emitErr error
	out := captureStdout(t, func() { emitErr = emit([]byte("hello world")) })
	if emitErr != nil {
		t.Fatalf("emit err = %v, want nil", emitErr)
	}
	if out != "hello world" {
		t.Errorf("emit wrote %q, want %q", out, "hello world")
	}
}

func TestWarnAndStep(t *testing.T) {
	out := captureStderr(t, func() { warn("bad %s", "thing") })
	if !strings.Contains(out, "[WARN] bad thing") {
		t.Errorf("warn wrote %q, want it to contain '[WARN] bad thing'", out)
	}
	out = captureStderr(t, func() { step("doing %d", 5) })
	if !strings.Contains(out, "==> doing 5") {
		t.Errorf("step wrote %q, want it to contain '==> doing 5'", out)
	}
}

func TestTreeStructure(t *testing.T) {
	root := newRootCmd(&App{})

	// Top-level platforms.
	for _, p := range []string{"k8s", "docker", "podman"} {
		findCmd(t, root, p) // fails the test if absent
	}

	var paths []string
	collectPaths(root, &paths)
	have := make(map[string]bool, len(paths))
	for _, p := range paths {
		have[p] = true
	}
	wantLeaves := []string{
		"solace k8s",
		"solace docker",
		"solace podman",
		"solace k8s prep operator",
		"solace k8s deploy",
		"solace k8s config exec-cli",
		"solace k8s verify diagnostics",
		"solace k8s copy from",
		"solace k8s copy into",
		"solace k8s operator deploy",
		"solace k8s gen",
		"solace k8s replicas start",
		"solace k8s restart",
		"solace docker deploy",
		"solace docker gen",
		"solace docker describe",
		"solace docker copy from",
		"solace docker copy into",
		"solace docker teardown domain-certs",
		"solace podman gen",
		"solace podman describe",
		"solace podman teardown domain-certs",
		"solace convert",
	}
	for _, want := range wantLeaves {
		if !have[want] {
			t.Errorf("command path %q missing from tree", want)
		}
	}
}

func TestFlagsRegistered(t *testing.T) {
	root := newRootCmd(&App{})
	cases := []struct {
		path  []string
		flags []string
	}{
		{[]string{"k8s", "delete"}, []string{"purge", "clear-data", "keep-data"}},
		{[]string{"k8s", "down"}, []string{"purge", "clear-data", "keep-data"}},
		{[]string{"docker", "delete"}, []string{"purge", "clear-data", "keep-data"}},
		{[]string{"docker", "down"}, []string{"purge", "clear-data", "keep-data"}},
		{[]string{"podman", "delete"}, []string{"purge", "clear-data", "keep-data"}},
		{[]string{"podman", "down"}, []string{"purge", "clear-data", "keep-data"}},
		{[]string{"k8s", "deploy"}, []string{"keep-yaml"}},
		{[]string{"k8s", "verify", "diagnostics"}, []string{"days"}},
		{[]string{"docker", "verify", "diagnostics"}, []string{"days"}},
		{[]string{"k8s", "config", "exec-cli"}, []string{"pod"}},
		{[]string{"k8s", "copy", "from"}, []string{"pod"}},
		{[]string{"k8s", "copy", "into"}, []string{"pod", "dir"}},
		{[]string{"docker", "copy", "into"}, []string{"dir"}},
		{[]string{"docker", "deploy"}, []string{"restart"}},
		{[]string{"docker", "up"}, []string{"restart"}},
		{[]string{"podman", "deploy"}, []string{"restart"}},
		{[]string{"podman", "up"}, []string{"restart"}},
	}
	for _, tc := range cases {
		t.Run(strings.Join(tc.path, "/"), func(t *testing.T) {
			cmd := findCmd(t, root, tc.path...)
			for _, name := range tc.flags {
				if cmd.Flags().Lookup(name) == nil {
					t.Errorf("%s: flag %q not registered", cmd.CommandPath(), name)
				}
			}
		})
	}
}

func TestHelpNoConfig(t *testing.T) {
	// --help short-circuits before PersistentPreRunE, so no env is needed.
	cases := [][]string{
		{"--help"},
		{"k8s", "--help"},
		{"docker", "--help"},
		{"podman", "--help"},
	}
	for _, args := range cases {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			if _, err := runRoot(t, args); err != nil {
				t.Errorf("help %q err = %v, want nil", args, err)
			}
		})
	}
}

func TestGenWired(t *testing.T) {
	cases := []struct {
		name   string
		args   []string
		prefix string
	}{
		{"k8s deploy --gen-only", []string{"k8s", "deploy", "--gen-only"}, "apiVersion:"},
		{"k8s gen broker", []string{"k8s", "gen", "broker"}, "apiVersion:"},
		{"k8s gen default", []string{"k8s", "gen"}, "apiVersion:"},
		{"docker deploy primary --gen-only", []string{"docker", "deploy", "primary", "--gen-only"}, "services:"},
		{"docker gen primary", []string{"docker", "gen", "primary"}, "services:"},
		{"docker gen primary --gen-secrets-only", []string{"docker", "gen", "primary", "--gen-secrets-only"}, "# docker secrets"},
		{"docker gen primary --gen-env-only", []string{"docker", "gen", "primary", "--gen-env-only"}, "routername="},
		{"podman deploy primary --gen-only", []string{"podman", "deploy", "primary", "--gen-only"}, "[Unit]"},
		{"podman gen primary", []string{"podman", "gen", "primary"}, "[Unit]"},
		{"podman gen primary --gen-secrets-only", []string{"podman", "gen", "primary", "--gen-secrets-only"}, "# podman secrets"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out, err := runRoot(t, withEnv(tc.args...))
			if err != nil {
				t.Fatalf("%s err = %v, want nil", tc.name, err)
			}
			if out == "" {
				t.Fatalf("%s produced empty stdout", tc.name)
			}
			if !strings.HasPrefix(out, tc.prefix) {
				t.Errorf("%s stdout prefix = %q, want it to start with %q", tc.name, firstLine(out), tc.prefix)
			}
		})
	}
}

// TestCtrWiredDryRun drives every container command that is safe under --dry-run
// against the HA sample env: each reaches its real handler over the Echo runner and
// returns no error, with the expected "+ <runtime> ..." (or systemctl/mkdir/chown)
// echo landing on stdout. Poll-driven steps (config leader / verify redundancy,
// which fail over or wait) and secret-bearing prep are covered by the guard,
// standalone, and error tests instead, so nothing here blocks on a poll loop. The
// sample is docker compose mode and rootful podman, so status/delete take the
// compose path and podman systemctl carries no --user.
func TestCtrWiredDryRun(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want string
	}{
		{"docker check", []string{"docker", "check"}, "+ docker version"},
		{"podman check", []string{"podman", "check"}, "+ podman version"},
		{"docker status", []string{"docker", "status"}, "+ docker ps"},
		{"podman status", []string{"podman", "status"}, "+ podman ps"},
		{"docker describe", []string{"docker", "describe"}, "+ docker inspect"},
		{"docker inspect alias", []string{"docker", "inspect"}, "+ docker inspect"},
		{"podman describe shows the unit", []string{"podman", "describe"}, "+ systemctl cat"},
		{"docker copy from", []string{"docker", "copy", "from", "a.log"}, "+ docker cp"},
		{"docker copy into", []string{"docker", "copy", "into", "a.cli", "--dir", "/tmp"}, "+ docker cp"},
		{"docker logs", []string{"docker", "logs"}, "+ docker logs -f"},
		{"docker cli", []string{"docker", "cli"}, "+ docker exec -it"},
		{"docker shell", []string{"docker", "shell"}, "+ docker exec -it"},
		{"docker prep", []string{"docker", "prep"}, "+ mkdir -p"},
		{"docker prep host", []string{"docker", "prep", "host"}, "+ chown"},
		{"docker deploy primary", []string{"docker", "deploy", "primary"}, "+ docker compose"},
		{"docker up primary", []string{"docker", "up", "primary"}, "+ docker compose"},
		{"podman deploy primary", []string{"podman", "deploy", "primary"}, "+ systemctl daemon-reload"},
		{"podman up primary", []string{"podman", "up", "primary"}, "+ systemctl daemon-reload"},
		{"docker delete", []string{"docker", "delete", "--yes", "--keep-data"}, "+ docker compose"},
		{"docker down", []string{"docker", "down", "--yes", "--keep-data"}, "+ docker compose"},
		{"podman delete", []string{"podman", "delete", "--yes", "--keep-data"}, "+ systemctl"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out, err := runCtr(t, sampleEnv, tc.args...)
			if err != nil {
				t.Fatalf("%s --dry-run err = %v, want nil", tc.name, err)
			}
			if !strings.Contains(out, tc.want) {
				t.Errorf("%s --dry-run stdout = %q, want it to contain %q", tc.name, out, tc.want)
			}
		})
	}
}

// TestCtrRoleGuards covers the fail-loud / self-skip role guards on the two
// node-local HA state machines (config leader, verify redundancy). None reach a
// poll loop: the HA cases are rejected before polling (wrong node or bad role) and
// the standalone cases return nil via skipIfStandalone, so every case resolves
// immediately under --dry-run.
func TestCtrRoleGuards(t *testing.T) {
	ha := sampleEnv
	standalone := writeCtrStandaloneEnv(t)
	cases := []struct {
		name    string
		env     string
		args    []string
		wantErr string // "" -> expect nil (self-skip path)
	}{
		{"config leader on monitor", ha, []string{"docker", "config", "leader", "monitor"}, "must run on the primary node"},
		{"config leader on backup", ha, []string{"podman", "config", "leader", "backup"}, "this host is the backup node"},
		{"verify redundancy on monitor", ha, []string{"docker", "verify", "redundancy", "monitor"}, "cannot run on the monitor node"},
		{"config leader bad role", ha, []string{"docker", "config", "leader", "bogus"}, "invalid node role"},
		{"config leader standalone skip", standalone, []string{"docker", "config", "leader"}, ""},
		{"verify redundancy standalone skip", standalone, []string{"podman", "verify", "redundancy"}, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := runCtr(t, tc.env, tc.args...)
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("%s err = %v, want nil (self-skip path)", tc.name, err)
				}
				return
			}
			if err == nil {
				t.Fatalf("%s err = nil, want an error containing %q", tc.name, tc.wantErr)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("%s err = %q, want it to contain %q", tc.name, err.Error(), tc.wantErr)
			}
		})
	}
}

// TestCtrConfigDryRun covers the post-deploy config steps that run cleanly on a
// container-standalone env under --dry-run: the VPN/user hardening and exec-cli
// echo their exec commands, while the cert/key- and product-key-gated steps
// self-skip (none configured). config leader is excluded -- it is HA-only and
// covered by TestCtrRoleGuards. None of these steps polls.
func TestCtrConfigDryRun(t *testing.T) {
	path := writeCtrStandaloneEnv(t)
	cases := []struct {
		name string
		args []string
	}{
		{"config all", []string{"docker", "config"}},
		{"config disable-default-vpn", []string{"docker", "config", "disable-default-vpn"}},
		{"config disable-default-users", []string{"docker", "config", "disable-default-users"}},
		{"config domain-certs (skip)", []string{"docker", "config", "domain-certs"}},
		{"config exec-cli", []string{"docker", "config", "exec-cli", "setup.cli"}},
		// opCtrTeardownDomainCerts' k8s sibling was already smoke-tested; this
		// closes the parity gap (it was never invoked by any test).
		{"teardown domain-certs (skip)", []string{"docker", "teardown", "domain-certs"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := runCtr(t, path, tc.args...); err != nil {
				t.Fatalf("%s (container-standalone) err = %v, want nil", tc.name, err)
			}
		})
	}
}

// TestCtrErrorPaths covers the actionable failures of the container config/verify
// steps on a container-standalone env under --dry-run: the cert and product-key
// steps demand configuration that is absent, exec-cli requires a file, and a login
// over the Echo runner cannot succeed against a non-existent broker. None polls.
func TestCtrErrorPaths(t *testing.T) {
	path := writeCtrStandaloneEnv(t)
	cases := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{"config server-cert (no tls)", []string{"docker", "config", "server-cert"}, "must both be set"},
		{"config product-keys (none)", []string{"docker", "config", "product-keys"}, "no product keys configured"},
		{"config exec-cli (no file)", []string{"docker", "config", "exec-cli"}, "CLI script file is required"},
		{"verify login (echo runner)", []string{"docker", "verify", "login"}, "SEMP login failed"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := runCtr(t, path, tc.args...)
			if err == nil {
				t.Fatalf("%s err = nil, want an error containing %q", tc.name, tc.wantErr)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("%s err = %q, want it to contain %q", tc.name, err.Error(), tc.wantErr)
			}
		})
	}
}

// TestCtrVerifyDiagnosticsDryRun is isolated because Diagnostics does an
// os.MkdirAll(diagDir) side-effect; t.Chdir(t.TempDir()) keeps the created dir out
// of the package directory. The container-standalone env's path is absolute, so it
// survives the chdir. On the Echo runner it echoes the node-local gather/download
// sequence without polling.
func TestCtrVerifyDiagnosticsDryRun(t *testing.T) {
	path := writeCtrStandaloneEnv(t)
	t.Chdir(t.TempDir())
	out, err := runCtr(t, path, "docker", "verify", "diagnostics")
	if err != nil {
		t.Fatalf("verify diagnostics --dry-run err = %v, want nil", err)
	}
	if !strings.Contains(out, "+ docker") {
		t.Errorf("verify diagnostics --dry-run stdout = %q, want a '+ docker ...' echo", out)
	}
}

// TestCtrRoleArgCount pins that the role-taking commands reject a second
// positional (cobra.MaximumNArgs(1)). Arg validation runs before PersistentPreRunE,
// so no env is loaded; --dry-run is a safety net in case that order ever changes.
func TestCtrRoleArgCount(t *testing.T) {
	cases := [][]string{
		{"docker", "config", "leader", "primary", "extra"},
		{"docker", "verify", "redundancy", "primary", "extra"},
	}
	for _, args := range cases {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			if _, err := runCtr(t, sampleEnv, args...); err == nil {
				t.Fatalf("%q err = nil, want a too-many-args error", args)
			}
		})
	}
}

// TestCtrRoleHelp confirms the role-taking commands expose --help (which
// short-circuits before PersistentPreRunE, so no env is needed). runRoot discards
// cobra's help output, so the assertion is purely that Execute returns no error.
func TestCtrRoleHelp(t *testing.T) {
	cases := [][]string{
		{"docker", "config", "leader", "--help"},
		{"podman", "verify", "redundancy", "--help"},
	}
	for _, args := range cases {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			if _, err := runRoot(t, args); err != nil {
				t.Errorf("help %q err = %v, want nil", args, err)
			}
		})
	}
}

// TestK8sWiredDryRun drives every k8s command that is safe to run against the HA
// sample env under --dry-run: each reaches its real handler over the Echo runner and
// returns no error. wantEcho commands shell out to kubectl (so a `+ kubectl ...` line
// lands on stdout); the skip-path commands (no configured labels / domain certs)
// return cleanly without touching the runner. Steps that need a live cluster to make
// sense on the HA sample (config leader/all -> redundancy poll, verify -> failover,
// server-cert/secrets -> absent cert files) are exercised in the standalone and error
// tests instead.
func TestK8sWiredDryRun(t *testing.T) {
	cases := []struct {
		name     string
		args     []string
		wantEcho bool
	}{
		{"check", []string{"k8s", "check"}, true},
		{"status", []string{"k8s", "status"}, true},
		{"show-all", []string{"k8s", "show-all"}, true},
		{"describe lb", []string{"k8s", "describe", "lb"}, true},
		{"describe broker", []string{"k8s", "describe", "broker"}, true},
		{"logs", []string{"k8s", "logs"}, true},
		{"cli", []string{"k8s", "cli"}, true},
		{"shell", []string{"k8s", "shell"}, true},
		{"replicas start", []string{"k8s", "replicas", "start"}, true},
		{"replicas stop", []string{"k8s", "replicas", "stop"}, true},
		{"inspect alias", []string{"k8s", "inspect", "lb"}, true},
		// restart deletes pods, so it takes the same --yes gate delete does.
		{"restart all", []string{"k8s", "restart", "--yes"}, true},
		{"restart one role", []string{"k8s", "restart", "backup", "--yes"}, true},
		// roleWord's remaining two cases (Backup is covered by "restart one role"
		// above): a swapped case would misname which pod the prompt is about to
		// bounce, so both must actually be reached, not assumed from Backup's.
		{"restart monitor role", []string{"k8s", "restart", "monitor", "--yes"}, true},
		{"restart primary role explicit", []string{"k8s", "restart", "primary", "--yes"}, true},
		{"deploy", []string{"k8s", "deploy"}, true},
		{"prep operator", []string{"k8s", "prep", "operator"}, true},
		{"prep namespace", []string{"k8s", "prep", "namespace"}, true},
		{"prep labels", []string{"k8s", "prep", "labels"}, false}, // no placement labels configured
		{"operator deploy", []string{"k8s", "operator", "deploy"}, true},
		{"operator delete", []string{"k8s", "operator", "delete"}, true},
		{"operator status", []string{"k8s", "operator", "status"}, true},
		{"operator logs", []string{"k8s", "operator", "logs"}, true},
		{"operator describe", []string{"k8s", "operator", "describe"}, true},
		{"config disable-default-vpn", []string{"k8s", "config", "disable-default-vpn"}, true},
		{"config disable-default-users", []string{"k8s", "config", "disable-default-users"}, true},
		{"config domain-certs", []string{"k8s", "config", "domain-certs"}, false}, // none configured
		{"config exec-cli", []string{"k8s", "config", "exec-cli", "setup.cli", "--pod", "p"}, true},
		{"copy from", []string{"k8s", "copy", "from", "somefile", "--pod", "p"}, true},
		{"copy into", []string{"k8s", "copy", "into", "somefile", "--pod", "p"}, true},
		{"teardown secrets", []string{"k8s", "teardown", "secrets"}, true},
		{"teardown namespace", []string{"k8s", "teardown", "namespace"}, true},
		{"teardown domain-certs", []string{"k8s", "teardown", "domain-certs"}, false}, // none configured
		// delete/down carry --yes --keep-data so neither confirm helper reads os.Stdin.
		{"delete", []string{"k8s", "delete", "--yes", "--keep-data"}, true},
		{"down", []string{"k8s", "down", "--yes", "--keep-data"}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			args := append([]string{}, tc.args...)
			args = append(args, "--dry-run")
			out, err := runRoot(t, withEnv(args...))
			if err != nil {
				t.Fatalf("%s --dry-run err = %v, want nil", tc.name, err)
			}
			if tc.wantEcho && !strings.Contains(out, "+ kubectl") {
				t.Errorf("%s --dry-run stdout = %q, want a '+ kubectl ...' echo", tc.name, out)
			}
			if !tc.wantEcho && strings.Contains(out, "+ kubectl") {
				t.Errorf("%s --dry-run stdout = %q, want no kubectl echo (skip path)", tc.name, out)
			}
		})
	}
}

// TestK8sStandaloneDryRun covers the commands whose behavior branches on redundancy:
// on a standalone env the HA-only steps self-skip (config leader / verify redundancy)
// and the secret-bearing prep steps have no TLS to guard, so config/prep/up all run
// clean under --dry-run.
func TestK8sStandaloneDryRun(t *testing.T) {
	path := writeStandaloneEnv(t)
	cases := []struct {
		name     string
		args     []string
		wantEcho bool
	}{
		{"config all", []string{"k8s", "config"}, true},
		{"config leader (skipped)", []string{"k8s", "config", "leader"}, false},
		{"verify redundancy (skipped)", []string{"k8s", "verify", "redundancy"}, false},
		{"prep secrets", []string{"k8s", "prep", "secrets"}, true},
		{"prep all", []string{"k8s", "prep"}, true},
		{"up", []string{"k8s", "up"}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out, err := runStandalone(t, path, tc.args...)
			if err != nil {
				t.Fatalf("%s (standalone) err = %v, want nil", tc.name, err)
			}
			if tc.wantEcho && !strings.Contains(out, "+ kubectl") {
				t.Errorf("%s (standalone) stdout = %q, want a '+ kubectl ...' echo", tc.name, out)
			}
			if !tc.wantEcho && strings.Contains(out, "+ kubectl") {
				t.Errorf("%s (standalone) stdout = %q, want no kubectl echo (skip path)", tc.name, out)
			}
		})
	}
}

// TestK8sVerifyDiagnosticsDryRun is isolated because Diagnostics does an
// os.MkdirAll(diagDir) side-effect; t.Chdir(t.TempDir()) keeps the created dir out of
// the package directory. On the HA sample it echoes the gather/download sequence for
// all three nodes without polling.
func TestK8sVerifyDiagnosticsDryRun(t *testing.T) {
	// Resolve the sample env to an absolute path before chdir; the relative sampleEnv
	// is anchored on the package dir, which t.Chdir moves us out of.
	absEnv, err := filepath.Abs(sampleEnv)
	if err != nil {
		t.Fatalf("abs sample env: %v", err)
	}
	t.Chdir(t.TempDir())
	out, err := runRoot(t, []string{"k8s", "verify", "diagnostics", "--dry-run", "--env", absEnv})
	if err != nil {
		t.Fatalf("verify diagnostics --dry-run err = %v, want nil", err)
	}
	if !strings.Contains(out, "+ kubectl") {
		t.Errorf("verify diagnostics --dry-run stdout = %q, want a '+ kubectl ...' echo", out)
	}
}

// TestK8sGenOperatorWired covers the operator render-only paths (`gen operator` and
// `operator deploy --gen`): both emit the embedded bundle to stdout with every
// template marker resolved. The bundle's first line is not stable enough for a
// prefix check, so this asserts on known interior content.
func TestK8sGenOperatorWired(t *testing.T) {
	cases := []struct {
		name string
		args []string
	}{
		{"gen operator", []string{"k8s", "gen", "operator"}},
		{"operator deploy --gen-only", []string{"k8s", "operator", "deploy", "--gen-only"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out, err := runRoot(t, withEnv(tc.args...))
			if err != nil {
				t.Fatalf("%s err = %v, want nil", tc.name, err)
			}
			if !strings.Contains(out, "kind: Deployment") || !strings.Contains(out, "pubsubplus-eventbroker-operator") {
				t.Errorf("%s stdout missing expected operator-bundle content (first line %q)", tc.name, firstLine(out))
			}
			if strings.Contains(out, "{{") {
				t.Errorf("%s stdout still contains an unresolved template marker {{", tc.name)
			}
		})
	}
}

// TestSecretsNeverEchoed is the §3 smoke check: a secret-bearing command under
// --dry-run must show its stdin as a byte count, never the secret value. `verify
// login` puts the admin credential on curl's stdin; the login itself fails under Echo
// (no broker), which is fine -- the assertion is purely that the password does not
// reach stdout.
func TestSecretsNeverEchoed(t *testing.T) {
	path := writeStandaloneEnv(t)
	out, _ := runStandalone(t, path, "k8s", "verify", "login")
	if !strings.Contains(out, "bytes on stdin") {
		t.Errorf("verify login --dry-run stdout = %q, want a 'bytes on stdin' redaction", out)
	}
	if strings.Contains(out, smokeAdminPass) {
		t.Error("verify login --dry-run leaked the admin password to stdout")
	}
}

// TestK8sErrorPaths covers the k8s handler error and rejection boundaries: config
// validation failures surfaced before any runner call, a failed SEMP login turned
// into a non-zero exit, a missing exec-cli argument, a bad --pod role, --gen rejected
// on non-artifact commands, and the mutually-exclusive data flags.
func TestK8sErrorPaths(t *testing.T) {
	standalone := writeStandaloneEnv(t)
	t.Run("server-cert without cert/key", func(t *testing.T) {
		_, err := runStandalone(t, standalone, "k8s", "config", "server-cert")
		if err == nil || !strings.Contains(err.Error(), "must both be set") {
			t.Fatalf("config server-cert err = %v, want 'must both be set'", err)
		}
	})
	t.Run("product-keys without keys", func(t *testing.T) {
		_, err := runStandalone(t, standalone, "k8s", "config", "product-keys")
		if err == nil || !strings.Contains(err.Error(), "no product keys configured") {
			t.Fatalf("config product-keys err = %v, want 'no product keys configured'", err)
		}
	})
	t.Run("additional-users without users", func(t *testing.T) {
		_, err := runStandalone(t, standalone, "k8s", "config", "additional-users")
		if err == nil || !strings.Contains(err.Error(), "no additional users configured") {
			t.Fatalf("config additional-users err = %v, want 'no additional users configured'", err)
		}
	})
	t.Run("verify login failure", func(t *testing.T) {
		_, err := runStandalone(t, standalone, "k8s", "verify", "login")
		if err == nil || !strings.Contains(err.Error(), "SEMP login failed") {
			t.Fatalf("verify login err = %v, want 'SEMP login failed'", err)
		}
	})
	t.Run("verify all reaches login on standalone", func(t *testing.T) {
		_, err := runStandalone(t, standalone, "k8s", "verify")
		if err == nil || !strings.Contains(err.Error(), "SEMP login failed") {
			t.Fatalf("verify err = %v, want 'SEMP login failed' (redundancy skipped)", err)
		}
	})
	t.Run("exec-cli without a file", func(t *testing.T) {
		_, err := runRoot(t, withEnv("k8s", "config", "exec-cli", "--dry-run"))
		if err == nil || !strings.Contains(err.Error(), "CLI script file is required") {
			t.Fatalf("exec-cli (no file) err = %v, want 'CLI script file is required'", err)
		}
	})
	t.Run("exec-cli with a bad --pod role", func(t *testing.T) {
		_, err := runRoot(t, withEnv("k8s", "config", "exec-cli", "setup.cli", "--pod", "bogus", "--dry-run"))
		if err == nil || !strings.Contains(err.Error(), "invalid node role") {
			t.Fatalf("exec-cli --pod bogus err = %v, want 'invalid node role'", err)
		}
	})
	t.Run("copy from with a bad --pod role", func(t *testing.T) {
		_, err := runRoot(t, withEnv("k8s", "copy", "from", "somefile", "--pod", "bogus", "--dry-run"))
		if err == nil || !strings.Contains(err.Error(), "invalid node role") {
			t.Fatalf("copy from --pod bogus err = %v, want 'invalid node role'", err)
		}
	})
	t.Run("copy into with a bad --pod role", func(t *testing.T) {
		_, err := runRoot(t, withEnv("k8s", "copy", "into", "somefile", "--pod", "bogus", "--dry-run"))
		if err == nil || !strings.Contains(err.Error(), "invalid node role") {
			t.Fatalf("copy into --pod bogus err = %v, want 'invalid node role'", err)
		}
	})
	t.Run("--gen-only rejected on delete", func(t *testing.T) {
		_, err := runRoot(t, withEnv("k8s", "delete", "--gen-only"))
		if err == nil || !strings.Contains(err.Error(), "--gen-only is only valid") {
			t.Fatalf("delete --gen-only err = %v, want '--gen-only is only valid'", err)
		}
	})
	t.Run("--gen-only rejected on status", func(t *testing.T) {
		_, err := runRoot(t, withEnv("k8s", "status", "--gen-only"))
		if err == nil || !strings.Contains(err.Error(), "--gen-only is only valid") {
			t.Fatalf("status --gen-only err = %v, want '--gen-only is only valid'", err)
		}
	})
	t.Run("--gen-env-only has no k8s artifact", func(t *testing.T) {
		_, err := runRoot(t, withEnv("k8s", "deploy", "--gen-env-only"))
		if err == nil || !strings.Contains(err.Error(), "no Kubernetes equivalent") {
			t.Fatalf("deploy --gen-env-only err = %v, want 'no Kubernetes equivalent'", err)
		}
	})
	t.Run("purge and keep-data are mutually exclusive", func(t *testing.T) {
		_, err := runRoot(t, withEnv("k8s", "delete", "--purge", "--keep-data"))
		if err == nil {
			t.Fatal("delete --purge --keep-data err = nil, want a mutually-exclusive parse error")
		}
	})
}

// TestConfirmFlagShortcuts covers the non-interactive short-circuits of the confirm
// helpers: --yes confirms a delete, and the data-retention decision is driven by the
// explicit flags without ever reading stdin (§: --yes never implies purge).
func TestConfirmFlagShortcuts(t *testing.T) {
	if !confirmDelete(&App{Yes: true}, "broker x") {
		t.Error("confirmDelete with --yes = false, want true")
	}
	if confirmPurge(&App{keepData: true}) {
		t.Error("confirmPurge with --keep-data = true, want false")
	}
	if !confirmPurge(&App{purge: true}) {
		t.Error("confirmPurge with --purge = false, want true")
	}
}

// TestConfirmNonTTY covers the unattended branches. Pointing os.Stdin at a pipe (not a
// character device) makes isTTY deterministically false, so confirmDelete refuses
// without --yes and confirmPurge keeps data -- with no prompt read, on any host.
func TestConfirmNonTTY(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	defer r.Close()
	defer w.Close()
	old := os.Stdin
	os.Stdin = r
	defer func() { os.Stdin = old }()

	var deleted bool
	_ = captureStderr(t, func() { deleted = confirmDelete(&App{}, "broker x") })
	if deleted {
		t.Error("confirmDelete non-TTY without --yes = true, want false")
	}
	if confirmPurge(&App{}) {
		t.Error("confirmPurge non-TTY = true, want false (keep)")
	}
}

// TestPromptYesNo pins the lenient delete confirmation: y/yes (any case) accept,
// everything else declines.
func TestPromptYesNo(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"y\n", true}, {"yes\n", true}, {"Y\n", true}, {"YES\n", true},
		{"n\n", false}, {"no\n", false}, {"\n", false}, {"maybe\n", false},
	}
	for _, tc := range cases {
		var out strings.Builder
		if got := promptYesNo(strings.NewReader(tc.in), &out, "? "); got != tc.want {
			t.Errorf("promptYesNo(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

// TestPromptYes pins the strict purge confirmation: only an exact (trimmed,
// case-insensitive) "yes" accepts; a bare "y" is not enough.
func TestPromptYes(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"yes\n", true}, {"YES\n", true}, {"  yes  \n", true},
		{"y\n", false}, {"no\n", false}, {"\n", false}, {"yess\n", false},
	}
	for _, tc := range cases {
		var out strings.Builder
		if got := promptYes(strings.NewReader(tc.in), &out, "? "); got != tc.want {
			t.Errorf("promptYes(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

func TestErrorPaths(t *testing.T) {
	t.Run("bad env path", func(t *testing.T) {
		// A value with a separator names one file: no env/ retry, so the error
		// lists that single candidate.
		_, err := runRoot(t, []string{"k8s", "status", "--env", "/no/such/file.yaml"})
		if err == nil || !strings.Contains(err.Error(), "not found: looked for") {
			t.Fatalf("bad --env err = %v, want a not-found error", err)
		}
	})
	t.Run("bad container role", func(t *testing.T) {
		_, err := runRoot(t, withEnv("docker", "deploy", "bogus"))
		if err == nil || !strings.Contains(err.Error(), "invalid node role") {
			t.Fatalf("docker deploy bogus err = %v, want 'invalid node role'", err)
		}
	})
	t.Run("bad container gen role", func(t *testing.T) {
		_, err := runRoot(t, withEnv("docker", "gen", "bogus"))
		if err == nil || !strings.Contains(err.Error(), "invalid node role") {
			t.Fatalf("docker gen bogus err = %v, want 'invalid node role'", err)
		}
	})
	t.Run("bad container up role", func(t *testing.T) {
		// ParseRole runs in RunE before any host operation, so the bogus role is
		// rejected without the (non-dry-run) Check/PrepHost/Deploy ever executing.
		_, err := runRoot(t, withEnv("docker", "up", "bogus"))
		if err == nil || !strings.Contains(err.Error(), "invalid node role") {
			t.Fatalf("docker up bogus err = %v, want 'invalid node role'", err)
		}
	})
	t.Run("bad k8s role leaf", func(t *testing.T) {
		_, err := runRoot(t, withEnv("k8s", "logs", "bogus"))
		if err == nil || !strings.Contains(err.Error(), "invalid node role") {
			t.Fatalf("k8s logs bogus err = %v, want 'invalid node role'", err)
		}
	})
	t.Run("unknown gen target", func(t *testing.T) {
		_, err := runRoot(t, withEnv("k8s", "gen", "bogus"))
		if err == nil || !strings.Contains(err.Error(), "unknown gen target") {
			t.Fatalf("k8s gen bogus err = %v, want 'unknown gen target'", err)
		}
	})
}

// TestDockerRunModeRejected pins the removal of run mode: an env file carrying
// the old value must fail with the reason and the fix, not a bare enum error.
func TestDockerRunModeRejected(t *testing.T) {
	path := filepath.Join(t.TempDir(), "runmode.yaml")
	content := "" +
		"redundancy: no\n" +
		"image:\n" +
		"  repo: solace-pubsub-standard\n" +
		"  tag: \"10.10.1.128\"\n" +
		"admin:\n" +
		"  pass: " + smokeAdminPass + "\n" +
		"docker:\n" +
		"  mode: run\n" +
		"  container:\n" +
		"    dataDir: /opt/solace/data\n" +
		"nodes:\n" +
		"  primary:\n" +
		"    name: solace-primary\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write temp env: %v", err)
	}
	_, err := runRoot(t, []string{"docker", "gen", "primary", "--env", path})
	if err == nil || !strings.Contains(err.Error(), "was removed") {
		t.Fatalf("docker.mode: run err = %v, want the removal message", err)
	}
	if !strings.Contains(err.Error(), "docker.compose") {
		t.Errorf("removal message should point at docker.compose, got: %v", err)
	}
}

// TestK8sGenSecretsWired covers the newly opt-in-renderable Secret manifests, via
// both the gen target and the flag. It uses the standalone env rather than the HA
// sample, whose tls.serverSecret points at cert files that do not exist in a
// checkout -- the same reason the secret-bearing prep steps are absent from
// TestK8sWiredDryRun.
func TestK8sGenSecretsWired(t *testing.T) {
	path := writeStandaloneEnv(t)
	for _, args := range [][]string{
		{"k8s", "gen", "secrets"},
		{"k8s", "prep", "secrets", "--gen-secrets-only"},
		{"k8s", "prep", "secrets", "--gen-only"},
		{"k8s", "deploy", "--gen-secrets-only"},
	} {
		name := strings.Join(args, " ")
		t.Run(name, func(t *testing.T) {
			out, err := runRoot(t, append(append([]string{}, args...), "--env", path))
			if err != nil {
				t.Fatalf("%s err = %v, want nil", name, err)
			}
			if !strings.HasPrefix(out, "apiVersion: v1") {
				t.Errorf("%s should render Secret manifests, got %q", name, firstLine(out))
			}
			// The manifests carry the value base64-encoded, so the raw password must
			// not appear -- but the rendering is still secret-bearing by design.
			if !strings.Contains(out, "kind: Secret") {
				t.Errorf("%s output is not a Secret manifest:\n%s", name, out)
			}
		})
	}
}

// TestGenSecretsRefusesEmptyValue: the printed script tells the operator to run
// it, so it must not be printable when running it would create an empty secret --
// `gen --gen-secrets-only` refuses on the same precondition `deploy` does. The HA
// sample with its PSK cleared is exactly the pre-`prep host` state.
func TestGenSecretsRefusesEmptyValue(t *testing.T) {
	body, err := os.ReadFile(sampleEnv)
	if err != nil {
		t.Fatalf("read sample env: %v", err)
	}
	blanked := strings.Replace(string(body),
		"psk: Q0hBTkdFLU1FLXByZXNoYXJlZC1rZXktYmFzZTY0", `psk: ""`, 1)
	if blanked == string(body) {
		t.Fatal("sample env no longer carries the psk line this test blanks")
	}
	path := filepath.Join(t.TempDir(), "no-psk.yaml")
	if err := os.WriteFile(path, []byte(blanked), 0o600); err != nil {
		t.Fatalf("write env: %v", err)
	}

	for _, platform := range []string{"docker", "podman"} {
		_, err := runRoot(t, []string{platform, "gen", "primary", "--gen-secrets-only", "--env", path})
		if err == nil || !strings.Contains(err.Error(), "nodes.psk") {
			t.Errorf("%s --gen-secrets-only with an empty PSK err = %v, want it to name nodes.psk", platform, err)
		}
	}
	// The deploy artifact only references secrets by name, so it stays renderable.
	if _, err := runRoot(t, []string{"docker", "gen", "primary", "--gen-only", "--env", path}); err != nil {
		t.Errorf("--gen-only should not need the secret values: %v", err)
	}
}

// TestGenFlagsAreExclusive covers checkGenFlags' combination arm: each flag
// selects a different artifact, so a pair is a user mistake rather than a
// silent precedence rule.
func TestGenFlagsAreExclusive(t *testing.T) {
	path := writeCtrStandaloneEnv(t)
	_, err := runRoot(t, []string{"docker", "gen", "--gen-only", "--gen-secrets-only", "--env", path})
	if err == nil || !strings.Contains(err.Error(), "cannot be combined") {
		t.Fatalf("combined gen flags err = %v, want 'cannot be combined'", err)
	}
}

// TestCtrGenFlagsRejectedOnNonArtifactCommands is the container half of the
// safety check the k8s tree already had: a gen flag on a destructive or
// read-only command must fail loud, never be ignored -- being ignored on
// `delete` would run the real delete while the user believed they asked for a
// dry render.
func TestCtrGenFlagsRejectedOnNonArtifactCommands(t *testing.T) {
	path := writeCtrStandaloneEnv(t)
	for _, platform := range []string{"docker", "podman"} {
		for _, cmd := range []string{"delete", "down", "status", "check"} {
			for _, flag := range []string{"--gen-only", "--gen-secrets-only", "--gen-env-only"} {
				name := platform + " " + cmd + " " + flag
				t.Run(name, func(t *testing.T) {
					_, err := runRoot(t, []string{platform, cmd, flag, "--env", path})
					if err == nil || !strings.Contains(err.Error(), "is only valid on artifact commands") {
						t.Fatalf("%s err = %v, want the artifact-command rejection", name, err)
					}
				})
			}
		}
	}
}

// TestGenNeverLeaksSecrets is the end-to-end guard for the secret
// externalization: the deploy artifacts a user prints, shares, or commits must
// reference the admin password by name, while --gen-secrets-only is the one
// rendering allowed to carry it.
func TestGenNeverLeaksSecrets(t *testing.T) {
	path := writeCtrStandaloneEnv(t)
	for _, platform := range []string{"docker", "podman"} {
		for _, flag := range []string{"--gen-only", "--gen-env-only"} {
			out, err := runRoot(t, []string{platform, "gen", flag, "--env", path})
			if err != nil {
				t.Fatalf("%s gen %s: %v", platform, flag, err)
			}
			if strings.Contains(out, smokeAdminPass) {
				t.Errorf("%s gen %s leaked the admin password:\n%s", platform, flag, out)
			}
		}
		out, err := runRoot(t, []string{platform, "gen", "--gen-secrets-only", "--env", path})
		if err != nil {
			t.Fatalf("%s gen --gen-secrets-only: %v", platform, err)
		}
		if !strings.Contains(out, smokeAdminPass) {
			t.Errorf("%s --gen-secrets-only must carry the value it creates the secret from:\n%s", platform, out)
		}
	}
}

// TestCtrVerifyAll covers opCtrVerifyAll's three role/redundancy arms without ever
// reaching the primary/backup poll path (which would block under Echo): (a) an HA
// env whose node table does not name this host -> LocalRole fails loud; (b) an HA
// env whose monitor row is this very host -> the monitor-skip branch runs, then the
// SEMP login fails over the Echo runner; (c) a standalone env -> redundancy is
// skipped and the login fails the same way. The primary/backup RedundancyLocal("")
// line is exercised directly in the broker tests instead.
func TestCtrVerifyAll(t *testing.T) {
	t.Run("HA host not in node table -> LocalRole error", func(t *testing.T) {
		_, err := runCtr(t, sampleEnv, "docker", "verify")
		if err == nil || !strings.Contains(err.Error(), "cannot determine node role") {
			t.Fatalf("verify (HA, unknown host) err = %v, want 'cannot determine node role'", err)
		}
	})

	t.Run("HA monitor host -> skip redundancy, login fails", func(t *testing.T) {
		host, err := os.Hostname()
		if err != nil {
			t.Skipf("os.Hostname unavailable: %v", err)
		}
		path := filepath.Join(t.TempDir(), "ha-monitor.yaml")
		content := "redundancy: yes\n" +
			"image:\n  repo: solace-pubsub-standard\n  tag: \"10.10.1.128\"\n" +
			"admin:\n  pass: " + smokeAdminPass + "\n" +
			"nodes:\n" +
			"  primary:\n    name: ctr-verifyall-primary\n    ip: 10.0.0.11\n" +
			"  backup:\n    name: ctr-verifyall-backup\n    ip: 10.0.0.12\n" +
			"  monitor:\n    name: '" + host + "'\n    ip: 10.0.0.13\n"
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write ha-monitor env: %v", err)
		}
		_, err = runCtr(t, path, "docker", "verify")
		if err == nil || !strings.Contains(err.Error(), "SEMP login failed") {
			t.Fatalf("verify (HA, this host is monitor) err = %v, want 'SEMP login failed' (redundancy skipped)", err)
		}
	})

	t.Run("standalone -> skip redundancy, login fails", func(t *testing.T) {
		path := writeCtrStandaloneEnv(t)
		_, err := runCtr(t, path, "docker", "verify")
		if err == nil || !strings.Contains(err.Error(), "SEMP login failed") {
			t.Fatalf("verify (standalone) err = %v, want 'SEMP login failed' (redundancy skipped)", err)
		}
	})
}

// TestCtrConfigAllArms drives opCtrConfigAll with every optional step configured, so
// the three gated arms (server-cert, domain-certs, product-keys) all run rather than
// self-skip. ServerCert reads the real temp cert/key files; DomainCerts/ProductKeys
// need no real files under Echo (uploads are cp; the CLI output is empty). The key
// material rides Upload's stdin, so the assertion also proves it never reaches stdout.
func TestCtrConfigAllArms(t *testing.T) {
	dir := t.TempDir()
	certPath := filepath.ToSlash(filepath.Join(dir, "tls.crt"))
	keyPath := filepath.ToSlash(filepath.Join(dir, "tls.key"))
	const keyMaterial = "PRIVATE-KEY-MATERIAL-do-not-log"
	if err := os.WriteFile(certPath, []byte("CERT-PEM\n"), 0o644); err != nil {
		t.Fatalf("write cert: %v", err)
	}
	if err := os.WriteFile(keyPath, []byte(keyMaterial+"\n"), 0o600); err != nil {
		t.Fatalf("write key: %v", err)
	}
	path := filepath.Join(dir, "cfgall.yaml")
	content := "redundancy: no\n" +
		"image:\n  repo: solace-pubsub-standard\n  tag: \"10.10.1.128\"\n" +
		"admin:\n  pass: " + smokeAdminPass + "\n" +
		"tls:\n  cert: " + certPath + "\n  certKey: " + keyPath + "\n" +
		"nodes:\n  primary:\n    name: pri-host\n" +
		"k8s:\n" +
		"  domainCerts:\n    folder: " + filepath.ToSlash(dir) + "\n    files:\n      myca: myca.pem\n" +
		"  productKeys:\n    - KEY-1\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write cfgall env: %v", err)
	}

	out, err := runCtr(t, path, "docker", "config")
	if err != nil {
		t.Fatalf("config all (every arm) err = %v, want nil", err)
	}
	if strings.Contains(out, keyMaterial) {
		t.Error("config all leaked the server-certificate private key to stdout")
	}
}

// TestCtrExecCLIPathSeparator covers opCtrExecCLI's used-as-is branch: a file
// argument containing a path separator is not joined under the cliScripts folder.
// The bare-filename (join) branch is covered by TestCtrConfigDryRun.
func TestCtrExecCLIPathSeparator(t *testing.T) {
	path := writeCtrStandaloneEnv(t)
	if _, err := runCtr(t, path, "docker", "config", "exec-cli", "sub/dir/x.cli"); err != nil {
		t.Fatalf("exec-cli with a path arg err = %v, want nil", err)
	}
}

// bashEnv is a minimal legacy container env file used by the convert tests.
const bashEnv = "#!/bin/bash\n" +
	"SOLBK_IMAGE=\"solace-pubsub-standard\"\n" +
	"SOLBK_IMG_TAG=\"10.10.1.128\"\n" +
	"SOLBK_ADM_PASS=\"" + smokeAdminPass + "\"\n" +
	"SOLBK_REDUNDANCY=\"no\"\n" +
	"SOLBK_NODE_PRI_NAME=\"pri-host\"\n" +
	"DOCKER_MODE=\"compose\"\n" +
	"SOMETHING_UNKNOWN=\"x\"\n"

// writeBashEnv writes bashEnv to a temp file and returns its path.
func writeBashEnv(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "legacy-env")
	if err := os.WriteFile(path, []byte(bashEnv), 0o600); err != nil {
		t.Fatalf("write bash env: %v", err)
	}
	return path
}

// TestConvertToStdout covers the default path: the YAML lands on stdout, and the
// unmapped-variable warning goes to stderr so it never pollutes the artifact.
func TestConvertToStdout(t *testing.T) {
	src := writeBashEnv(t)
	var out string
	errOut := captureStderr(t, func() {
		var err error
		out, err = runRoot(t, []string{"convert", src})
		if err != nil {
			t.Fatalf("convert err = %v, want nil", err)
		}
	})
	for _, want := range []string{"redundancy: \"no\"", "repo: solace-pubsub-standard", "name: pri-host"} {
		if !strings.Contains(out, want) {
			t.Errorf("converted YAML missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "SOMETHING_UNKNOWN") {
		t.Errorf("the unmapped-variable note must not reach stdout:\n%s", out)
	}
	if !strings.Contains(errOut, "SOMETHING_UNKNOWN") {
		t.Errorf("stderr = %q, want the unmapped-variable warning", errOut)
	}
}

// TestConvertToFile covers -o, including the refusal to clobber an existing file
// and the --force override.
func TestConvertToFile(t *testing.T) {
	src := writeBashEnv(t)
	dst := filepath.Join(t.TempDir(), "converted.yaml")

	_ = captureStderr(t, func() {
		if _, err := runRoot(t, []string{"convert", src, "-o", dst}); err != nil {
			t.Fatalf("convert -o err = %v, want nil", err)
		}
	})
	body, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("read converted file: %v", err)
	}
	if !strings.Contains(string(body), "image:") {
		t.Errorf("converted file looks wrong:\n%s", body)
	}

	_ = captureStderr(t, func() {
		_, err := runRoot(t, []string{"convert", src, "-o", dst})
		if err == nil || !strings.Contains(err.Error(), "refusing to overwrite") {
			t.Fatalf("second convert err = %v, want a refusal", err)
		}
	})
	_ = captureStderr(t, func() {
		if _, err := runRoot(t, []string{"convert", src, "-o", dst, "--force"}); err != nil {
			t.Fatalf("convert --force err = %v, want nil", err)
		}
	})
}

// TestConvertRoundTrip proves the converter's output is loadable: convert, then
// drive a real command with -e against the result.
func TestConvertRoundTrip(t *testing.T) {
	src := writeBashEnv(t)
	dst := filepath.Join(t.TempDir(), "converted.yaml")
	_ = captureStderr(t, func() {
		if _, err := runRoot(t, []string{"convert", src, "-o", dst}); err != nil {
			t.Fatalf("convert err = %v, want nil", err)
		}
	})
	out, err := runCtr(t, dst, "docker", "status")
	if err != nil {
		t.Fatalf("docker status against the converted env err = %v, want nil", err)
	}
	if !strings.Contains(out, "+ docker") {
		t.Errorf("converted env did not drive a real command:\n%s", out)
	}
}

func TestConvertErrorPaths(t *testing.T) {
	src := writeBashEnv(t)
	t.Run("bad platform", func(t *testing.T) {
		_, err := runRoot(t, []string{"convert", src, "--platform", "bogus"})
		if err == nil || !strings.Contains(err.Error(), "invalid --platform") {
			t.Fatalf("err = %v, want an invalid-platform error", err)
		}
	})
	t.Run("missing source file", func(t *testing.T) {
		_, err := runRoot(t, []string{"convert", filepath.Join(t.TempDir(), "nope")})
		if err == nil || !strings.Contains(err.Error(), "read bash env file") {
			t.Fatalf("err = %v, want a read error", err)
		}
	})
	t.Run("no source file", func(t *testing.T) {
		if _, err := runRoot(t, []string{"convert"}); err == nil {
			t.Fatal("convert with no argument should fail")
		}
	})
}

// TestBashEnvGivenToEnvFlag is the other half of the migration story: pointing
// -e at a legacy bash file must say it is not YAML and name the converter.
func TestBashEnvGivenToEnvFlag(t *testing.T) {
	src := writeBashEnv(t)
	_, err := runRoot(t, []string{"k8s", "status", "--dry-run", "-e", src})
	if err == nil {
		t.Fatal("a bash env file should not load")
	}
	for _, want := range []string{"not valid YAML", "this looks like a legacy bash env file", "solace convert"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not mention %q", err.Error(), want)
		}
	}
}

func TestExecute(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"solace", "--help"}
	var err error
	captureStdout(t, func() { err = Execute() })
	if err != nil {
		t.Fatalf("Execute(--help) err = %v, want nil", err)
	}
}

// TestK8sConfirmDeclined covers opK8sDelete/opK8sDown's confirm-declined branch: a
// non-interactive run (no --yes) must issue zero cluster calls -- the actual
// safety default. Every confirm-gated case in TestK8sWiredDryRun only exercises
// the --yes path, so the silent-decline default was unproven for either command.
func TestK8sConfirmDeclined(t *testing.T) {
	cases := []struct {
		name string
		args []string
	}{
		{"delete declined", []string{"k8s", "delete"}},
		{"down declined", []string{"k8s", "down"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			args := append(withEnv(tc.args...), "--dry-run")
			out, err := runRootWith(t, args, func(a *App) { a.Interactive = func() bool { return false } })
			if err != nil {
				t.Fatalf("%s err = %v, want nil", tc.name, err)
			}
			if strings.Contains(out, "+ kubectl") {
				t.Errorf("%s stdout = %q, want no kubectl echo (declined)", tc.name, out)
			}
		})
	}
}

// TestK8sRestartConfirmGate covers opK8sRestart's confirmation gate: a
// non-interactive run (no --yes) must bounce nothing, whether restarting every pod
// or a single one, and a bad role is rejected before any prompt is even possible.
func TestK8sRestartConfirmGate(t *testing.T) {
	t.Run("restart all declined (no --yes)", func(t *testing.T) {
		args := append(withEnv("k8s", "restart"), "--dry-run")
		out, err := runRootWith(t, args, func(a *App) { a.Interactive = func() bool { return false } })
		if err != nil {
			t.Fatalf("restart declined err = %v, want nil", err)
		}
		if strings.Contains(out, "+ kubectl") {
			t.Errorf("restart declined stdout = %q, want no kubectl echo", out)
		}
	})
	t.Run("restart one role declined (no --yes)", func(t *testing.T) {
		args := append(withEnv("k8s", "restart", "backup"), "--dry-run")
		out, err := runRootWith(t, args, func(a *App) { a.Interactive = func() bool { return false } })
		if err != nil {
			t.Fatalf("restart backup declined err = %v, want nil", err)
		}
		if strings.Contains(out, "+ kubectl") {
			t.Errorf("restart backup declined stdout = %q, want no kubectl echo", out)
		}
	})
	t.Run("bad role rejected before any prompt", func(t *testing.T) {
		args := append(withEnv("k8s", "restart", "bogus"), "--dry-run")
		_, err := runRoot(t, args)
		if err == nil || !strings.Contains(err.Error(), "invalid node role") {
			t.Fatalf("restart bogus err = %v, want 'invalid node role'", err)
		}
	})
}

// TestCtrConfirmDeclined covers opCtrDelete's confirm-declined branch, the
// container-side counterpart of TestK8sConfirmDeclined: a non-interactive delete
// (no --yes) must issue zero runtime calls.
func TestCtrConfirmDeclined(t *testing.T) {
	path := writeCtrStandaloneEnv(t)
	args := []string{"docker", "delete", "--dry-run", "--env", path}
	out, err := runRootWith(t, args, func(a *App) { a.Interactive = func() bool { return false } })
	if err != nil {
		t.Fatalf("docker delete declined err = %v, want nil", err)
	}
	if strings.Contains(out, "+ docker") {
		t.Errorf("docker delete declined stdout = %q, want no docker echo", out)
	}
}

// TestIsTTYClosedFile covers isTTY's fail-safe branch: when Stat() cannot be
// evaluated (a closed file, on both Windows and POSIX), it must treat the stream
// as non-interactive rather than risk blocking a confirm helper on a prompt.
func TestIsTTYClosedFile(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "closed-*")
	if err != nil {
		t.Fatalf("CreateTemp: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("close temp file: %v", err)
	}
	if isTTY(f) {
		t.Error("isTTY(closed file) = true, want false (fail-safe)")
	}
}

// TestCtrManagerConfirmWiring covers confirmRestart's wiring into ctrManager: a
// regression here (ctrManager forgetting to set m.Confirm, or confirmRestart
// losing its interactive guard) would silently let a non-interactive deploy bounce
// a live broker container unattended. It drives the closure through the App's
// Interactive seam rather than the ambient os.Stdin, so the outcome is
// deterministic on any host.
func TestCtrManagerConfirmWiring(t *testing.T) {
	a := &App{Interactive: func() bool { return false }}
	m := ctrManager(a)
	if m.Confirm == nil {
		t.Fatal("ctrManager did not wire Manager.Confirm")
	}
	if m.Confirm("restart now?") {
		t.Error("Confirm (non-interactive) = true, want false")
	}
}

// TestK8sLoginOutcomes covers k8sLogin's two real, distinct SEMP outcomes -- a
// transport failure (error propagates) vs. a successful login (nil) -- which
// engine.Echo's fixed (nil, nil) OutputInput return can never produce; every
// existing test only reaches the "no HTTP response" failure-but-not-error branch.
func TestK8sLoginOutcomes(t *testing.T) {
	cfg, err := config.Load(writeStandaloneEnv(t), config.K8s)
	if err != nil {
		t.Fatalf("load standalone env: %v", err)
	}
	t.Run("SEMP login succeeds", func(t *testing.T) {
		rr := &opRunner{output: func(opCall) []byte { return []byte("HTTP/1.1 200 OK\r\n\r\n") }}
		a := &App{Cfg: cfg, Platform: config.K8s, Runner: rr}
		var loginErr error
		captureStdout(t, func() { loginErr = k8sLogin(a, k8sOps(a), config.Primary) })
		if loginErr != nil {
			t.Errorf("k8sLogin (canned 200 OK) err = %v, want nil", loginErr)
		}
	})
	t.Run("SEMP transport failure propagates", func(t *testing.T) {
		rr := &opRunner{fail: opFailOn("curl")}
		a := &App{Cfg: cfg, Platform: config.K8s, Runner: rr}
		var loginErr error
		captureStdout(t, func() { loginErr = k8sLogin(a, k8sOps(a), config.Primary) })
		if loginErr == nil || !strings.Contains(loginErr.Error(), "SEMP request failed") {
			t.Errorf("k8sLogin (transport failure) err = %v, want it to contain 'SEMP request failed'", loginErr)
		}
	})
}

// TestCtrLoginOutcomes is ctrLogin's half of TestK8sLoginOutcomes.
func TestCtrLoginOutcomes(t *testing.T) {
	cfg, err := config.Load(writeCtrStandaloneEnv(t), config.Docker)
	if err != nil {
		t.Fatalf("load container standalone env: %v", err)
	}
	t.Run("SEMP login succeeds", func(t *testing.T) {
		rr := &opRunner{output: func(opCall) []byte { return []byte("HTTP/1.1 200 OK\r\n\r\n") }}
		a := &App{Cfg: cfg, Platform: config.Docker, Runner: rr}
		var loginErr error
		captureStdout(t, func() { loginErr = ctrLogin(a, ctrOps(a)) })
		if loginErr != nil {
			t.Errorf("ctrLogin (canned 200 OK) err = %v, want nil", loginErr)
		}
	})
	t.Run("SEMP transport failure propagates", func(t *testing.T) {
		rr := &opRunner{fail: opFailOn("curl")}
		a := &App{Cfg: cfg, Platform: config.Docker, Runner: rr}
		var loginErr error
		captureStdout(t, func() { loginErr = ctrLogin(a, ctrOps(a)) })
		if loginErr == nil || !strings.Contains(loginErr.Error(), "SEMP request failed") {
			t.Errorf("ctrLogin (transport failure) err = %v, want it to contain 'SEMP request failed'", loginErr)
		}
	})
}

// writeK8sConfigAllArmsEnv writes an HA k8s env with every optional
// opK8sConfigAll arm configured -- the TLS-secret server-cert fast path, domain
// certs, additional users, and product keys -- so a direct call actually enters
// every gated step instead of self-skipping like every existing k8s fixture
// (sample.yaml leaves domainCerts/additionalUsers/productKeys unset, and
// writeStandaloneEnv is not even HA).
func writeK8sConfigAllArmsEnv(t *testing.T) *config.Config {
	t.Helper()
	dir := t.TempDir()
	certPath := filepath.ToSlash(filepath.Join(dir, "tls.crt"))
	keyPath := filepath.ToSlash(filepath.Join(dir, "tls.key"))
	if err := os.WriteFile(certPath, []byte("CERT-PEM\n"), 0o644); err != nil {
		t.Fatalf("write cert: %v", err)
	}
	if err := os.WriteFile(keyPath, []byte("KEY-PEM-do-not-log\n"), 0o600); err != nil {
		t.Fatalf("write key: %v", err)
	}
	yamlBody := "redundancy: yes\n" +
		"image:\n  repo: solace-pubsub-standard\n  tag: \"10.10.1.128\"\n" +
		"admin:\n" +
		"  pass: " + smokeAdminPass + "\n" +
		"  additionalUsers:\n" +
		"    - username: extrauser\n" +
		"      accessLevel: read-only\n" +
		"      password: EXTRA-USER-PW-do-not-log-1\n" +
		"tls:\n" +
		"  serverSecret: solace-tls-secret\n" +
		"  cert: " + certPath + "\n" +
		"  certKey: " + keyPath + "\n" +
		"k8s:\n" +
		"  name: dev-broker\n" +
		"  namespace: solace\n" +
		"  adminSecret: solace-admin-secret\n" +
		"  updateStrategy: automatedRolling\n" +
		"  storage:\n    class: standard\n    msgNode: 30Gi\n" +
		"  domainCerts:\n    folder: " + filepath.ToSlash(dir) + "\n    files:\n      myca: myca.pem\n" +
		"  productKeys:\n    - KEY-1\n"
	return loadDirect(t, yamlBody, config.K8s)
}

// k8sConfigAllRunner builds an opRunner for opK8sConfigAll: it always supplies a
// healthy `show redundancy` transcript (so Leader's poll succeeds on the first
// check), with fail as the caller's targeted failure, if any.
func k8sConfigAllRunner(fail func(opCall) error) *opRunner {
	return &opRunner{
		fail: fail,
		output: func(c opCall) []byte {
			if opArgvMatch(c, ".show-rd.cli") {
				return []byte(healthyShowRD)
			}
			return nil
		},
	}
}

// TestOpK8sConfigAllArms drives opK8sConfigAll directly with every optional step
// configured, so every gated arm actually runs instead of self-skipping: the HA
// leader assertion and the TLS-secret server-cert fast path (cli-1, cli-1b), plus
// domain certs, additional users and product keys (cli-2). sample.yaml is already
// HA+TLS but leaves the other three unset, and no existing test's config enters
// any of these five arms at all -- an inverted condition in any of them would ship
// silently. It bypasses config.Load's --dry-run wiring for a fake Runner that
// supplies a healthy `show redundancy` transcript, so Leader's poll succeeds on the
// first check instead of exhausting broker.New's real 2s x 60 poll budget, which
// the CLI layer has no seam to shorten.
func TestOpK8sConfigAllArms(t *testing.T) {
	cfg := writeK8sConfigAllArmsEnv(t)
	rr := k8sConfigAllRunner(nil)
	a := &App{Cfg: cfg, Platform: config.K8s, Runner: rr}
	var configErr error
	captureStdout(t, func() { configErr = opK8sConfigAll(a) })
	if configErr != nil {
		t.Fatalf("opK8sConfigAll (every arm configured) err = %v, want nil", configErr)
	}
	for _, want := range []string{
		"revert-activity",     // Leader
		"assert-leader",       // Leader
		"apply",               // ServerCert via the tls.serverSecret fast path
		"load-domain-certs",   // DomainCerts
		"disable-default-vpn", // always runs; pins the marker the aborts test relies on
		"additional-users",    // AdditionalUsers
		"product-keys",        // ProductKeys
	} {
		if !rr.hasCall(want) {
			t.Errorf("opK8sConfigAll (every arm) never reached the %q step", want)
		}
	}
	for _, c := range rr.calls {
		for _, arg := range c.args {
			if strings.Contains(arg, "do-not-log") {
				t.Errorf("a secret value reached argv: %q", arg)
			}
		}
	}
}

// TestOpK8sConfigAllAborts drives opK8sConfigAll's six error-return arms in turn
// (leader, server-cert, domain-certs, disable-default-vpn, disable-default-users,
// additional-users), each time failing exactly that step's underlying command and
// asserting the NEXT step's command was never issued -- pinning the function's own
// documented ordering (harden-then-provision; additionalUsers is not re-runnable)
// rather than merely that some error came back.
func TestOpK8sConfigAllAborts(t *testing.T) {
	cases := []struct {
		name      string
		fail      func(opCall) error
		wantNever string
	}{
		{"leader fails -> server-cert never runs", opFailOn("revert-activity"), "apply"},
		{"server-cert fails -> domain-certs never runs", opFailOn("apply"), "load-domain-certs"},
		{"domain-certs fails -> disable-default-vpn never runs", opFailOn("load-domain-certs"), "disable-default-vpn"},
		{"disable-default-vpn fails -> disable-default-users never runs", opFailOn("disable-default-vpn"), "show-vpn"},
		{"disable-default-users fails -> additional-users never runs", failDisableDefaultUsersUpload, "additional-users"},
		{"additional-users fails -> product-keys never runs", opFailOn("additional-users"), "product-keys"},
	}
	// One fixture for every case, built from the PARENT t on purpose. t.TempDir()
	// embeds the test name in the path, and these subtests are named after the very
	// steps the fault injector and the assertions match on -- so a per-subtest temp
	// dir puts "additional-users" and "product-keys" into the `kubectl cp <local>`
	// argv of the domain-certs step, firing both the injection and the assertion on
	// the wrong call. The config is only read, so sharing it is safe.
	cfg := writeK8sConfigAllArmsEnv(t)
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rr := k8sConfigAllRunner(tc.fail)
			a := &App{Cfg: cfg, Platform: config.K8s, Runner: rr}
			var err error
			captureStdout(t, func() { err = opK8sConfigAll(a) })
			if err == nil {
				t.Fatal("opK8sConfigAll = nil, want the injected failure to abort the sequence")
			}
			if rr.hasCall(tc.wantNever) {
				t.Errorf("opK8sConfigAll issued the %q command after the injected failure; the abort did not stop the sequence.\nerr = %v\ncalls:\n%s",
					tc.wantNever, err, rr.dump())
			}
		})
	}
}

// writeK8sUpEnv builds the same minimal, valid k8s env as writeStandaloneEnv but
// with redundancy overridable, so opK8sUp's HA-only final Leader() branch can be
// exercised (writeStandaloneEnv is fixed at redundancy: no).
func writeK8sUpEnv(t *testing.T, redundancy string) *config.Config {
	t.Helper()
	yamlBody := "redundancy: " + redundancy + "\n" +
		"image:\n  repo: solace-pubsub-standard\n  tag: \"10.10.1.128\"\n" +
		"admin:\n  pass: " + smokeAdminPass + "\n" +
		"k8s:\n" +
		"  name: dev-broker\n" +
		"  namespace: solace\n" +
		"  adminSecret: solace-admin-secret\n" +
		"  updateStrategy: automatedRolling\n" +
		"  storage:\n    class: standard\n    msgNode: 30Gi\n"
	return loadDirect(t, yamlBody, config.K8s)
}

// k8sUpOutputHook supplies the canned Output content opK8sUp's happy path needs
// once a fake (non-Echo) Runner is in play: k8s.Cluster.isDryRun() is a concrete
// type assertion on engine.Echo, so a fake Runner takes CheckStorageClass's real
// validation branch (unlike engine.Echo, which skips it) -- it needs a
// WaitForFirstConsumer/true StorageClass answer to pass, and Leader (HA only)
// needs a healthy `show redundancy` transcript to avoid its real poll budget.
func k8sUpOutputHook(c opCall) []byte {
	switch {
	case opArgvMatch(c, "volumeBindingMode"):
		return []byte("WaitForFirstConsumer")
	case opArgvMatch(c, "allowVolumeExpansion"):
		return []byte("true")
	case opArgvMatch(c, ".show-rd.cli"):
		return []byte(healthyShowRD)
	default:
		return nil
	}
}

// TestOpK8sUpAssertsLeaderOnHA covers opK8sUp's final branch: on an HA config,
// `up` must assert the config-sync leader as its last step, not just deploy the
// broker and stop. It is unreachable via runRoot/engine.Echo -- Echo's fixed empty
// output never satisfies Leader's poll, which would otherwise run for broker.New's
// real 2s x 60 budget -- so this drives opK8sUp directly over a fake Runner.
func TestOpK8sUpAssertsLeaderOnHA(t *testing.T) {
	cfg := writeK8sUpEnv(t, "yes")
	rr := &opRunner{output: k8sUpOutputHook}
	a := &App{Cfg: cfg, Platform: config.K8s, Runner: rr}
	var upErr error
	captureStdout(t, func() { upErr = opK8sUp(a) })
	if upErr != nil {
		t.Fatalf("opK8sUp (HA) err = %v, want nil", upErr)
	}
	if !rr.hasCall("assert-leader") {
		t.Error("opK8sUp did not assert the config-sync leader after deploying an HA broker")
	}
}

// TestOpK8sUpAborts covers opK8sUp's five error-return arms (Check, OperatorApply,
// CreateNamespace, CreateSecrets, DeployBroker), each in a separate sub-test that
// fails exactly that step and asserts no later step's command was issued -- `up`'s
// own docstring promises every step aborts loud on failure, previously unverified
// (only the all-succeed path was tested).
func TestOpK8sUpAborts(t *testing.T) {
	t.Run("check fails -> nothing else runs", func(t *testing.T) {
		cfg := writeK8sUpEnv(t, "no")
		rr := &opRunner{fail: opFailOn("version"), output: k8sUpOutputHook}
		a := &App{Cfg: cfg, Platform: config.K8s, Runner: rr}
		var err error
		captureStdout(t, func() { err = opK8sUp(a) })
		if err == nil {
			t.Fatal("opK8sUp = nil, want the injected Check failure to abort")
		}
		if rr.hasCall("apply") {
			t.Error("opK8sUp issued an apply command after Check failed")
		}
	})

	steps := []string{"operator-apply", "create-namespace", "create-secrets", "deploy-broker"}
	for i, step := range steps {
		n := i + 1
		t.Run(step+" fails -> the next step never runs", func(t *testing.T) {
			cfg := writeK8sUpEnv(t, "no")
			rr := &opRunner{fail: opFailOnCount("apply", n), output: k8sUpOutputHook}
			a := &App{Cfg: cfg, Platform: config.K8s, Runner: rr}
			var err error
			captureStdout(t, func() { err = opK8sUp(a) })
			if err == nil {
				t.Fatalf("opK8sUp = nil, want the injected %s failure to abort", step)
			}
			if got := rr.callCount("apply"); got != n {
				t.Errorf("opK8sUp issued %d apply command(s) after %s failed, want exactly %d (no later step ran)", got, step, n)
			}
		})
	}
}

// TestOpK8sPrepAllAborts covers opK8sPrepAll's three error-return arms
// (OperatorApply, CreateNamespace, CreateSecrets): the same abort-ordering
// property as opK8sUp, on the four-step prep sequence (LabelNodes, the fourth
// step, self-skips with no custom labels configured and has no error-return of its
// own to test).
func TestOpK8sPrepAllAborts(t *testing.T) {
	steps := []string{"operator-apply", "create-namespace", "create-secrets"}
	for i, step := range steps {
		n := i + 1
		t.Run(step+" fails -> the next step never runs", func(t *testing.T) {
			cfg := writeK8sUpEnv(t, "no")
			rr := &opRunner{fail: opFailOnCount("apply", n)}
			a := &App{Cfg: cfg, Platform: config.K8s, Runner: rr}
			var err error
			captureStdout(t, func() { err = opK8sPrepAll(a) })
			if err == nil {
				t.Fatalf("opK8sPrepAll = nil, want the injected %s failure to abort", step)
			}
			if got := rr.callCount("apply"); got != n {
				t.Errorf("opK8sPrepAll issued %d apply command(s) after %s failed, want exactly %d", got, step, n)
			}
		})
	}
}

// TestOpK8sDownAborts covers opK8sDown's two error-return arms (DeleteBroker,
// DeleteSecrets): it must not remove the namespace out from under a broker- or
// secrets-deletion that actually failed, leaving orphaned state -- a real
// correctness property of the teardown order, not just error forwarding.
func TestOpK8sDownAborts(t *testing.T) {
	steps := []struct {
		name string
		n    int
	}{
		{"delete-broker", 1},
		{"delete-secrets", 2},
	}
	for _, s := range steps {
		t.Run(s.name+" fails -> delete-namespace never runs", func(t *testing.T) {
			cfg := writeK8sUpEnv(t, "no")
			rr := &opRunner{fail: opFailOnCount("delete", s.n)}
			a := &App{Cfg: cfg, Platform: config.K8s, Runner: rr, Yes: true, keepData: true}
			var err error
			captureStdout(t, func() { err = opK8sDown(a) })
			if err == nil {
				t.Fatalf("opK8sDown = nil, want the injected %s failure to abort", s.name)
			}
			if got := rr.callCount("delete"); got != s.n {
				t.Errorf("opK8sDown issued %d delete command(s) after %s failed, want exactly %d (delete-namespace never ran)", got, s.name, s.n)
			}
		})
	}
}

// writeCtrConfigAllArmsCfg builds a container config with every opCtrConfigAll arm
// configured (server-cert, domain certs, product keys), mirroring
// TestCtrConfigAllArms' own fixture but returning a *config.Config directly for a
// direct call with a fault-injecting Runner.
func writeCtrConfigAllArmsCfg(t *testing.T) *config.Config {
	t.Helper()
	dir := t.TempDir()
	certPath := filepath.ToSlash(filepath.Join(dir, "tls.crt"))
	keyPath := filepath.ToSlash(filepath.Join(dir, "tls.key"))
	if err := os.WriteFile(certPath, []byte("CERT-PEM\n"), 0o644); err != nil {
		t.Fatalf("write cert: %v", err)
	}
	if err := os.WriteFile(keyPath, []byte("KEY-PEM-do-not-log\n"), 0o600); err != nil {
		t.Fatalf("write key: %v", err)
	}
	yamlBody := "redundancy: no\n" +
		"image:\n  repo: solace-pubsub-standard\n  tag: \"10.10.1.128\"\n" +
		"admin:\n  pass: " + smokeAdminPass + "\n" +
		"tls:\n  cert: " + certPath + "\n  certKey: " + keyPath + "\n" +
		"nodes:\n  primary:\n    name: pri-host\n" +
		"k8s:\n" +
		"  domainCerts:\n    folder: " + filepath.ToSlash(dir) + "\n    files:\n      myca: myca.pem\n" +
		"  productKeys:\n    - KEY-1\n"
	return loadDirect(t, yamlBody, config.Docker)
}

// TestOpCtrConfigAllAborts covers opCtrConfigAll's four error-return arms
// (ServerCert, DomainCerts, DisableDefaultVPN, DisableDefaultUsers) -- the same
// abort-ordering argument as the k8s config-all cluster (TestOpK8sConfigAllAborts):
// proves a failed cert upload stops before domain-certs/hardening run rather than
// leaving the broker half-configured. The true-branch entries are already covered
// by TestCtrConfigAllArms.
func TestOpCtrConfigAllAborts(t *testing.T) {
	cases := []struct {
		name      string
		fail      func(opCall) error
		wantNever string
	}{
		{"server-cert fails -> domain-certs never runs", opFailOn("apply-server-certs"), "load-domain-certs"},
		{"domain-certs fails -> disable-default-vpn never runs", opFailOn("load-domain-certs"), "disable-default-vpn"},
		{"disable-default-vpn fails -> disable-default-users never runs", opFailOn("disable-default-vpn"), "show-vpn"},
		{"disable-default-users fails -> product-keys never runs", failDisableDefaultUsersUpload, "product-keys"},
	}
	// Built from the parent t for the same reason as TestOpK8sConfigAllAborts: these
	// subtests are named after the steps being matched, and a per-subtest t.TempDir()
	// would put those names into the `docker cp <local>` argv of the domain-certs
	// step, matching the wrong call.
	cfg := writeCtrConfigAllArmsCfg(t)
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rr := &opRunner{fail: tc.fail}
			a := &App{Cfg: cfg, Platform: config.Docker, Runner: rr}
			var err error
			captureStdout(t, func() { err = opCtrConfigAll(a) })
			if err == nil {
				t.Fatal("opCtrConfigAll = nil, want the injected failure to abort the sequence")
			}
			if rr.hasCall(tc.wantNever) {
				t.Errorf("opCtrConfigAll issued the %q command after the injected failure; the abort did not stop the sequence", tc.wantNever)
			}
		})
	}
}

// TestOpCtrVerifyAllRunsRedundancyLocal covers opCtrVerifyAll's actual failover
// exercise arm (the "else if" branch calling RedundancyLocal) rather than only its
// two skip/reject arms (TestCtrVerifyAll already covers the unknown-host and
// this-host-is-monitor arms). It sets this host as the primary and supplies a
// canned `show redundancy` transcript over a fake Runner, since engine.Echo's
// fixed empty output would otherwise send RedundancyLocal into its real poll loop
// (broker.New's 2s x 60 budget, which the CLI has no seam to shorten).
func TestOpCtrVerifyAllRunsRedundancyLocal(t *testing.T) {
	host, err := os.Hostname()
	if err != nil {
		t.Skipf("os.Hostname unavailable: %v", err)
	}
	yamlBody := "redundancy: yes\n" +
		"image:\n  repo: solace-pubsub-standard\n  tag: \"10.10.1.128\"\n" +
		"admin:\n  pass: " + smokeAdminPass + "\n" +
		"nodes:\n" +
		"  primary:\n    name: '" + host + "'\n    ip: 10.0.0.11\n" +
		"  backup:\n    name: ctr-redundancylocal-backup\n    ip: 10.0.0.12\n" +
		"  monitor:\n    name: ctr-redundancylocal-monitor\n    ip: 10.0.0.13\n"
	cfg := loadDirect(t, yamlBody, config.Docker)
	rr := &opRunner{output: func(c opCall) []byte {
		if opArgvMatch(c, ".show-rd.cli") {
			// Non-empty but deliberately unhealthy (no Configuration/Redundancy
			// Status lines): primaryRedundancyUp is false, so redundancyLocalPrimary
			// returns its health error immediately -- proving RedundancyLocal ran,
			// without ever entering a real poll loop.
			return []byte("Activity Status: Local Active\n")
		}
		return nil
	}}
	a := &App{Cfg: cfg, Platform: config.Docker, Runner: rr}
	captureStdout(t, func() { err = opCtrVerifyAll(a) })
	if err == nil || !strings.Contains(err.Error(), "redundancy configuration/status is not healthy") {
		t.Fatalf("opCtrVerifyAll (primary, active-but-unhealthy) err = %v, want the redundancy-unhealthy error", err)
	}
}

// TestK8sVerifyAllRedundancyUnhealthy covers opK8sVerifyAll's Redundancy
// error-return: on the HA sample under --dry-run, engine.Echo's empty `show
// redundancy` output makes primaryRedundancyUp false, so Redundancy fails on its
// first check (no poll -- unlike Leader, whose poll only runs once the initial
// check has already failed) and verify must never reach the SEMP login at all.
func TestK8sVerifyAllRedundancyUnhealthy(t *testing.T) {
	_, err := runRoot(t, withEnv("k8s", "verify", "--dry-run"))
	if err == nil || !strings.Contains(err.Error(), "redundancy configuration/status is not healthy") {
		t.Fatalf("k8s verify (HA sample) err = %v, want the redundancy-unhealthy error", err)
	}
}

// TestK8sTeardownDomainCertsConfigured covers domainCANames' loop body: every
// other test's domainCerts.files map is empty, so the map-to-slice conversion
// feeding RemoveDomainCerts is trivially correct by vacuity. This configures one
// CA and asserts the teardown actually issues a kubectl exec rather than
// self-skipping.
func TestK8sTeardownDomainCertsConfigured(t *testing.T) {
	path := filepath.Join(t.TempDir(), "domaincerts.yaml")
	content := "redundancy: no\n" +
		"image:\n  repo: solace-pubsub-standard\n  tag: \"10.10.1.128\"\n" +
		"admin:\n  pass: " + smokeAdminPass + "\n" +
		"k8s:\n" +
		"  name: dev-broker\n" +
		"  namespace: solace\n" +
		"  adminSecret: solace-admin-secret\n" +
		"  updateStrategy: automatedRolling\n" +
		"  storage:\n    class: standard\n    msgNode: 30Gi\n" +
		"  domainCerts:\n    folder: certs\n    files:\n      myca: myca.pem\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write env: %v", err)
	}
	out, err := runRoot(t, []string{"k8s", "teardown", "domain-certs", "--dry-run", "--env", path})
	if err != nil {
		t.Fatalf("teardown domain-certs (configured) err = %v, want nil", err)
	}
	if !strings.Contains(out, "+ kubectl") {
		t.Errorf("teardown domain-certs (configured) stdout = %q, want a '+ kubectl ...' echo", out)
	}
}

// TestConvertParseError covers runConvert's convert.Convert error return: a
// malformed legacy env file (an unterminated array assignment) is a real,
// actionable failure a migrating user can actually hit.
func TestConvertParseError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad-array.env")
	if err := os.WriteFile(path, []byte("SOLBK_TLS_CERTCAS=(\n  \"/a\"\n"), 0o600); err != nil {
		t.Fatalf("write malformed env: %v", err)
	}
	_, err := runRoot(t, []string{"convert", path})
	if err == nil || !strings.Contains(err.Error(), "unterminated array") {
		t.Fatalf("convert (unterminated array) err = %v, want it to name the parse failure", err)
	}
}

// TestConvertWriteError covers runConvert's os.WriteFile error return: an -o path
// in a directory that does not exist is a real mistake a migrating user can make.
func TestConvertWriteError(t *testing.T) {
	src := writeBashEnv(t)
	dst := filepath.Join(t.TempDir(), "missing-subdir", "out.yaml")
	_, err := runRoot(t, []string{"convert", src, "-o", dst})
	// runConvert formats the path with %q, so the expectation is built the same way:
	// a raw path would match on Linux and never on Windows, where %q escapes every
	// separator.
	want := fmt.Sprintf("write %q", dst)
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("convert -o (missing parent dir) err = %v, want it to contain %s", err, want)
	}
}

// TestK8sGenSecretsMissingCertFile covers emitK8sArtifact's GenSecrets error
// return: tls.serverSecret is configured but the referenced cert file cannot be
// read -- a real broken-reference case, otherwise caught only at real deploy time.
func TestK8sGenSecretsMissingCertFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "badcert.yaml")
	content := "redundancy: no\n" +
		"image:\n  repo: solace-pubsub-standard\n  tag: \"10.10.1.128\"\n" +
		"admin:\n  pass: " + smokeAdminPass + "\n" +
		"tls:\n" +
		"  serverSecret: solace-tls-secret\n" +
		"  cert: " + filepath.ToSlash(filepath.Join(dir, "missing.crt")) + "\n" +
		"  certKey: " + filepath.ToSlash(filepath.Join(dir, "missing.key")) + "\n" +
		"k8s:\n" +
		"  name: dev-broker\n" +
		"  namespace: solace\n" +
		"  adminSecret: solace-admin-secret\n" +
		"  updateStrategy: automatedRolling\n" +
		"  storage:\n    class: standard\n    msgNode: 30Gi\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write env: %v", err)
	}
	_, err := runRoot(t, []string{"k8s", "gen", "secrets", "--dry-run", "--env", path})
	if err == nil || !strings.Contains(err.Error(), "read tls.cert") {
		t.Fatalf("k8s gen secrets (missing cert file) err = %v, want it to wrap the tls.cert read failure", err)
	}
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}

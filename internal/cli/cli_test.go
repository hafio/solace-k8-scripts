package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"solace/internal/config"
	"solace/internal/engine"
)

// sampleEnv is the shared fixture: the user template doubles as a valid config
// for k8s, docker, and podman (same file render_test.go renders from). During
// `go test ./internal/cli` the cwd is the package dir, so this relative path
// resolves to the repo's env/sample.yaml.
const sampleEnv = "../../env/sample.yaml"

// withEnv appends the sample --env so a command reaching PreRunE loads
// a valid config instead of the missing env.yaml. The value carries a separator,
// so it resolves verbatim rather than through the base-dir/env lookup.
func withEnv(args ...string) []string {
	return append(append([]string{}, args...), "--env", sampleEnv)
}

// echoRunner installs engine.Echo as the App's runner. It replaces the retired
// --dry-run flag: the flag is gone from the CLI, but the property these tests
// assert -- which argv a command would issue -- is unchanged, so the seam that
// used to be a user-facing mode is now a test-only one.
func echoRunner(a *App) { a.NewRunner = func(*App) engine.Runner { return engine.Echo{W: os.Stdout} } }

// smokeAdminPass is a distinctive admin password used only by the standalone test
// env, so TestSecretsNeverEchoed can prove it never reaches stdout.
const smokeAdminPass = "SMOKE-PW-do-not-log-1234"

// writeStandaloneEnv writes a minimal single-broker (redundancy: no) env to a temp
// file and returns its path. The HA sample sets tls.serverSecret with absent cert
// files and defines all three nodes, which makes the secret-bearing prep steps fail
// and the HA-only config/verify steps poll or exercise failover -- unsuitable for a
// clean run over the echo seam. This env has no TLS and no nodes, so those steps
// self-skip.
func writeStandaloneEnv(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "standalone.yaml")
	content := "redundancy: no\n" +
		"image:\n" +
		"  repo: solace-pubsub-standard\n" +
		"  tag: \"10.10.1.128\"\n" +
		"admin:\n" +
		"  pass: " + smokeAdminPass + "\n" +
		"kubernetes:\n" +
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

// runStandalone runs a k8s command over the echo seam against the standalone env.
func runStandalone(t *testing.T, path string, args ...string) (string, error) {
	t.Helper()
	full := append(append([]string{}, args...), "--env", path)
	return runRootWith(t, full, echoRunner)
}

// runCtr runs a container (docker/podman) command over the echo seam against the
// given env path. It is the container-facing name for the same run that
// runStandalone performs, kept distinct for readability at container call sites
// (against both the HA sample and a container-standalone env).
func runCtr(t *testing.T, path string, args ...string) (string, error) {
	t.Helper()
	return runStandalone(t, path, args...)
}

// writeCtrStandaloneEnv writes a minimal single-broker (redundancy: no) env that is
// valid for docker and podman: only nodes.primary.name is mandatory (data dir and
// network mode default). It carries no TLS, domainCerts, or productKeys, so the
// config steps that gate on those self-skip and none of them reach a poll loop --
// keeping every run over the echo seam fast and deterministic. The k8s-shaped
// writeStandaloneEnv cannot be reused: it has no nodes: block, which fails
// container validation (nodes.primary.name is required).
//
// An env file must declare its platform section -- even an empty one -- or
// resolvePlatform refuses it ("declares no platform section"). This fixture
// declares BOTH docker: {} and podman: {} so the one file still serves either
// platform; every caller is therefore ambiguous and must pass --platform.
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
		"    name: pri-host\n" +
		"docker: {}\n" +
		"podman: {}\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write container standalone env: %v", err)
	}
	return path
}

// loadDirect writes yamlBody to a temp file and loads it for platform p, for
// tests that need a *config.Config to build an App directly (bypassing App.load
// and its runner-selection entirely) so a fake Runner other than engine.Echo can
// be attached -- see opRunner below.
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
// opK8sDeployAll/opK8sPrepAll/opK8sRemoveAll/opCtrVerifyRedundancy etc. are
// unexported handlers that take only an *App (or an *App plus a role arg), so a
// test in this package can call them directly rather than through cobra/runRoot.
// Doing so is what makes two things possible that engine.Echo cannot provide: (1)
// targeted fault injection -- Echo's Run/Output methods never return an error, so
// the ordering property "step N fails, step N+1 never runs" has no double to
// exercise it without this, mirroring internal/container's capRunner/failOn
// (transport_test.go); and (2) canned Output content -- Echo always returns (nil,
// nil), which makes a poll-based HA state machine (config-sync leader,
// redundancy) exhaust its real PollInterval x PollAttempts budget (broker.New's 2s
// x 60 defaults) before timing out, since the CLI layer has no seam to shorten it.
// Supplying a healthy transcript lets the state machine succeed on its first check
// instead.
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
// opK8sDeployAll/opK8sRemoveAll's repeated, textually-identical steps (every
// `apply -f -` / `delete ... --ignore-not-found` call looks the same; only its
// position in the sequence identifies which step it belongs to).
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
// branches (confirmDelete/confirmLayer/confirmRestart) that a real terminal would
// otherwise gate on the test process's own, environment-dependent stdin, or
// installing echoRunner to capture the argv a command would issue instead of
// running it for real. configure may be nil, in which case this is exactly
// runRoot.
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

// runStatusStderr runs `status broker` over the echo seam with the given flags and
// returns what reached stderr, where the resolved-env-file line is echoed.
func runStatusStderr(t *testing.T, args ...string) (string, error) {
	t.Helper()
	var runErr error
	errOut := captureStderr(t, func() {
		_, runErr = runRootWith(t, append([]string{"status", "broker", "--platform", "kubernetes"}, args...), echoRunner)
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

// TestTreeStructure covers the one flat tree: the platform is resolved rather than
// typed as the first word of a command (platform.go), so every command lives
// directly off root (or off a verb group) regardless of which platform it applies
// to. The sample below is representative, not exhaustive: a top-level leaf, a
// group's child, a couple of multi-level paths, and at least one command from
// each applicability class (shared, kubernetes-only, container-only).
func TestTreeStructure(t *testing.T) {
	root := newRootCmd(&App{})

	var paths []string
	collectPaths(root, &paths)
	have := make(map[string]bool, len(paths))
	for _, p := range paths {
		have[p] = true
	}
	wantLeaves := []string{
		"solace-util check deploy",             // group's child, shared
		"solace-util deploy broker",            // group's child, shared
		"solace-util prepare namespace",        // group's child, kubernetes-only
		"solace-util remove operator",          // group's child, kubernetes-only
		"solace-util generate operator",        // group's child, kubernetes-only
		"solace-util config apply server-cert", // three-level path, shared
		"solace-util config leader",            // two-level path, shared
		"solace-util copy from",
		"solace-util copy into",
		"solace-util status broker",
		"solace-util convert",
		"solace-util auto-complete",
		"solace-util auto-complete powershell",
		"solace-util version",
	}
	for _, want := range wantLeaves {
		if !have[want] {
			t.Errorf("command path %q missing from tree", want)
		}
	}
}

// TestEveryRunnableCommandIsWired pins the wiring that replaced the two
// PersistentPreRunE hooks: a command that runs something must carry the shared
// pre-run (which resolves the platform and loads the env file) and the
// --allow-command flag that goes with executing. Missing either is invisible
// until that one command is run, so it is checked structurally instead. The verb
// groups (check, deploy, config, remove, ...) carry no RunE at all -- they print
// help and act on nothing -- so this walk never touches them.
func TestEveryRunnableCommandIsWired(t *testing.T) {
	root := newRootCmd(&App{})

	var walk func(c *cobra.Command)
	walk = func(c *cobra.Command) {
		path := c.CommandPath()
		// A verb group carries a RunE only to print help or reject an unknown noun
		// (group(), commands.go) -- it reaches no external command, so it must NOT
		// carry the pre-run or --allow-command.
		excluded := c.Annotations[groupAnnotation] == "true" ||
			path == "solace-util convert" ||
			path == "solace-util version" ||
			path == "solace-util auto-complete" ||
			strings.HasPrefix(path, "solace-util auto-complete ")
		if c.RunE != nil && !excluded {
			if c.PreRunE == nil {
				t.Errorf("%s: has RunE but no PreRunE (missing wireExec)", path)
			}
			if c.Flags().Lookup("allow-command") == nil {
				t.Errorf("%s: has RunE but no --allow-command flag (missing wireExec)", path)
			}
		}
		for _, sub := range c.Commands() {
			walk(sub)
		}
	}
	walk(root)

	if root.PersistentFlags().Lookup("allow-command") != nil {
		t.Error("root carries --allow-command as a persistent flag; it must be per-command (wireExec), never inherited")
	}
	convert := findCmd(t, root, "convert")
	if convert.Flags().Lookup("allow-command") != nil {
		t.Error("convert has an --allow-command flag; it loads no env file and executes nothing")
	}
}

// TestGroupCommandsPrintHelpAndDoNothing pins the no-implicit-actions rule
// (commands.go): a verb that owns objects has no RunE, so running it bare prints
// its own help and touches nothing. Proof that it touches nothing is that it
// succeeds with no --env at all -- a runnable leaf would fail resolving the
// missing default env.yaml, but a bare group never reaches PreRunE.
func TestGroupCommandsPrintHelpAndDoNothing(t *testing.T) {
	for _, name := range []string{
		"check", "smoke", "prepare", "deploy", "config", "start", "stop",
		"restart", "status", "logs", "copy", "generate", "remove",
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := runRoot(t, []string{name}); err != nil {
				t.Errorf("%s (bare) err = %v, want nil (prints help, does not act)", name, err)
			}
		})
	}
	// config's own object groups (apply/delete/disable) behave the same way.
	for _, path := range [][]string{{"config", "apply"}, {"config", "delete"}, {"config", "disable"}} {
		t.Run(strings.Join(path, " "), func(t *testing.T) {
			if _, err := runRoot(t, path); err != nil {
				t.Errorf("%v (bare) err = %v, want nil (prints help, does not act)", path, err)
			}
		})
	}
}

// TestFlagsRegistered covers flag registration on the flat tree. Each command is
// one static shape on every platform (platform.go), so a platform-scoped flag
// like --restart is registered on `deploy broker` unconditionally and only its
// applicability (flagOnlyOn) narrows by platform.
func TestFlagsRegistered(t *testing.T) {
	root := newRootCmd(&App{})
	cases := []struct {
		path  []string
		flags []string
	}{
		{[]string{"deploy", "broker"}, []string{"restart"}},
		{[]string{"deploy", "all"}, []string{"restart"}},
		{[]string{"remove", "broker"}, []string{"delete-data", "no-prompt"}},
		{[]string{"remove", "operator"}, []string{"delete-crd", "no-prompt"}},
		{[]string{"remove", "all"}, []string{"delete-data", "no-prompt"}},
		{[]string{"diagnostics"}, []string{"days"}},
		{[]string{"cli"}, []string{"input", "pod"}},
		{[]string{"copy", "from"}, []string{"pod"}},
		{[]string{"copy", "into"}, []string{"pod", "dir"}},
		{[]string{"status", "broker"}, []string{"all", "detail"}},
		{[]string{"status", "operator"}, []string{"detail"}},
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
	// --help short-circuits before PreRunE, so no env is needed.
	cases := [][]string{
		{"--help"},
		{"deploy", "--help"},
		{"config", "--help"},
		{"status", "--help"},
	}
	for _, args := range cases {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			if _, err := runRoot(t, args); err != nil {
				t.Errorf("help %q err = %v, want nil", args, err)
			}
		})
	}
}

// TestGenerateWired covers `generate`'s named-artifact leaves: none of them
// contact the cluster or the container engine, so a plain runRoot (no echo seam)
// is enough.
func TestGenerateWired(t *testing.T) {
	cases := []struct {
		name   string
		args   []string
		prefix string
	}{
		{"kubernetes generate broker", []string{"generate", "broker", "--platform", "kubernetes"}, "apiVersion:"},
		{"docker generate broker", []string{"generate", "broker", "--platform", "docker"}, "services:"},
		{"docker generate secrets", []string{"generate", "secrets", "--platform", "docker"}, "# docker secrets"},
		{"podman generate broker", []string{"generate", "broker", "--platform", "podman"}, "[Unit]"},
		{"podman generate secrets", []string{"generate", "secrets", "--platform", "podman"}, "# podman secrets"},
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

// TestCtrWiredDryRun drives every container command that is safe to run against
// the HA sample env over the echo seam: each reaches its real handler and returns
// no error, with the expected "+ <runtime> ..." (or systemctl/mkdir/chown) echo
// landing on stdout. Poll-driven steps (config leader / smoke redundancy, which
// fail over or wait) and secret-bearing prep are covered by the guard, standalone,
// and error tests instead, so nothing here blocks on a poll loop. The sample is
// docker compose mode and rootful podman, so status/remove take the compose path
// and podman systemctl carries no --user.
func TestCtrWiredDryRun(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want string
	}{
		{"docker check deploy", []string{"check", "deploy", "--platform", "docker"}, "+ docker version"},
		{"podman check deploy", []string{"check", "deploy", "--platform", "podman"}, "+ podman version"},
		{"docker status broker", []string{"status", "broker", "--platform", "docker"}, "+ docker ps"},
		{"podman status broker", []string{"status", "broker", "--platform", "podman"}, "+ podman ps"},
		{"docker copy from", []string{"copy", "from", "a.log", "--platform", "docker"}, "+ docker cp"},
		{"docker copy into", []string{"copy", "into", "a.cli", "--dir", "/tmp", "--platform", "docker"}, "+ docker cp"},
		{"docker logs broker", []string{"logs", "broker", "--platform", "docker"}, "+ docker logs -f"},
		{"docker cli", []string{"cli", "--platform", "docker"}, "+ docker exec -it"},
		{"docker shell", []string{"shell", "--platform", "docker"}, "+ docker exec -it"},
		{"docker prepare all", []string{"prepare", "all", "--platform", "docker"}, "+ mkdir -p"},
		{"docker prepare host", []string{"prepare", "host", "--platform", "docker"}, "+ chown"},
		{"docker deploy broker primary", []string{"deploy", "broker", "primary", "--platform", "docker"}, "+ docker compose"},
		{"docker deploy all primary", []string{"deploy", "all", "primary", "--platform", "docker"}, "+ docker compose"},
		{"podman deploy broker primary", []string{"deploy", "broker", "primary", "--platform", "podman"}, "+ systemctl daemon-reload"},
		{"podman deploy all primary", []string{"deploy", "all", "primary", "--platform", "podman"}, "+ systemctl daemon-reload"},
		{"docker remove broker", []string{"remove", "broker", "--no-prompt", "--platform", "docker"}, "+ docker compose"},
		{"podman remove broker", []string{"remove", "broker", "--no-prompt", "--platform", "podman"}, "+ systemctl"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out, err := runCtr(t, sampleEnv, tc.args...)
			if err != nil {
				t.Fatalf("%s err = %v, want nil", tc.name, err)
			}
			if !strings.Contains(out, tc.want) {
				t.Errorf("%s stdout = %q, want it to contain %q", tc.name, out, tc.want)
			}
		})
	}
}

// TestCtrRoleGuards covers the fail-loud / self-skip role guards on the two
// node-local HA state machines (config leader, smoke redundancy). None reach a
// poll loop: the HA cases are rejected before polling (wrong node, unknown host,
// or bad role) and the standalone cases return nil via skipIfStandalone, so every
// case resolves immediately over the echo seam.
func TestCtrRoleGuards(t *testing.T) {
	ha := sampleEnv
	standalone := writeCtrStandaloneEnv(t)
	cases := []struct {
		name    string
		env     string
		args    []string
		wantErr string // "" -> expect nil (self-skip path)
	}{
		{"config leader on monitor", ha, []string{"config", "leader", "monitor", "--platform", "docker"}, "must run on the primary node"},
		{"config leader on backup", ha, []string{"config", "leader", "backup", "--platform", "podman"}, "this host is the backup node"},
		{"smoke redundancy on monitor", ha, []string{"smoke", "redundancy", "monitor", "--platform", "docker"}, "cannot run on the monitor node"},
		{"smoke redundancy unknown host", ha, []string{"smoke", "redundancy", "--platform", "docker"}, "cannot determine node role"},
		{"config leader bad role", ha, []string{"config", "leader", "bogus", "--platform", "docker"}, "invalid node role"},
		{"config leader standalone skip", standalone, []string{"config", "leader", "--platform", "docker"}, ""},
		{"smoke redundancy standalone skip", standalone, []string{"smoke", "redundancy", "--platform", "podman"}, ""},
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
// container-standalone env over the echo seam: the VPN/user hardening and cli
// --input echo their exec commands, while the cert/key-gated steps self-skip
// (none configured). config leader is excluded -- it is HA-only and covered by
// TestCtrRoleGuards. None of these steps polls.
func TestCtrConfigDryRun(t *testing.T) {
	path := writeCtrStandaloneEnv(t)
	cases := []struct {
		name string
		args []string
	}{
		{"config disable default-vpn", []string{"config", "disable", "default-vpn", "--platform", "docker"}},
		{"config disable default-users", []string{"config", "disable", "default-users", "--platform", "docker"}},
		{"config apply domain-certs (skip)", []string{"config", "apply", "domain-certs", "--platform", "docker"}},
		{"cli --input runs a script", []string{"cli", "--input", "setup.cli", "--platform", "docker"}},
		{"config delete domain-certs (skip)", []string{"config", "delete", "domain-certs", "--platform", "docker"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := runCtr(t, path, tc.args...); err != nil {
				t.Fatalf("%s (container-standalone) err = %v, want nil", tc.name, err)
			}
		})
	}
}

// TestCtrExecCLIPathSeparator covers opCtrExecCLI's used-as-is branch: a file
// argument containing a path separator is not joined under the cliScripts folder.
// The bare-filename (join) branch is covered by TestCtrConfigDryRun.
func TestCtrExecCLIPathSeparator(t *testing.T) {
	path := writeCtrStandaloneEnv(t)
	if _, err := runCtr(t, path, "cli", "--input", "sub/dir/x.cli", "--platform", "docker"); err != nil {
		t.Fatalf("cli --input with a path arg err = %v, want nil", err)
	}
}

// TestCtrErrorPaths covers the actionable failures of the container config/check
// steps on a container-standalone env over the echo seam: the cert and
// product-key steps demand configuration that is absent, and a login over the
// echo runner cannot succeed against a non-existent broker. None polls.
func TestCtrErrorPaths(t *testing.T) {
	path := writeCtrStandaloneEnv(t)
	cases := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{"config apply server-cert (no tls)", []string{"config", "apply", "server-cert", "--platform", "docker"}, "must both be set"},
		{"config apply product-keys (none)", []string{"config", "apply", "product-keys", "--platform", "docker"}, "no product keys configured"},
		{"check semp-login (echo runner)", []string{"check", "semp-login", "--platform", "docker"}, "SEMP login failed"},
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

// TestCtrDiagnosticsDryRun is isolated because Diagnostics does an
// os.MkdirAll(diagDir) side-effect; t.Chdir(t.TempDir()) keeps the created dir out
// of the package directory. The container-standalone env's path is absolute, so it
// survives the chdir. Over the echo seam it echoes the node-local gather/download
// sequence without polling.
func TestCtrDiagnosticsDryRun(t *testing.T) {
	path := writeCtrStandaloneEnv(t)
	t.Chdir(t.TempDir())
	out, err := runCtr(t, path, "diagnostics", "--platform", "docker")
	if err != nil {
		t.Fatalf("diagnostics err = %v, want nil", err)
	}
	if !strings.Contains(out, "+ docker") {
		t.Errorf("diagnostics stdout = %q, want a '+ docker ...' echo", out)
	}
}

// TestCtrRoleArgCount pins that the role-taking commands reject a second
// positional (cobra.MaximumNArgs(1)). Arg validation runs before PreRunE, so no
// env is loaded.
func TestCtrRoleArgCount(t *testing.T) {
	cases := [][]string{
		{"config", "leader", "primary", "extra", "--platform", "docker"},
		{"smoke", "redundancy", "primary", "extra", "--platform", "docker"},
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
// short-circuits before PreRunE, so no env is needed). runRoot discards
// cobra's help output, so the assertion is purely that Execute returns no error.
func TestCtrRoleHelp(t *testing.T) {
	cases := [][]string{
		{"config", "leader", "--help", "--platform", "docker"},
		{"smoke", "redundancy", "--help", "--platform", "podman"},
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
// sample env over the echo seam: each reaches its real handler and returns no
// error. wantEcho commands shell out to kubectl (so a `+ kubectl ...` line lands
// on stdout); the skip-path commands (no configured labels / domain certs) return
// cleanly without touching the runner. Steps that need a live cluster to make
// sense on the HA sample (config leader -> redundancy poll, smoke redundancy ->
// failover, server-cert/secrets -> absent cert files) are exercised in the
// standalone and error tests instead.
func TestK8sWiredDryRun(t *testing.T) {
	cases := []struct {
		name     string
		args     []string
		wantEcho bool
	}{
		{"check deploy", []string{"check", "deploy"}, true},
		{"status broker", []string{"status", "broker"}, true},
		{"status broker --all", []string{"status", "broker", "--all"}, true},
		{"status broker --detail", []string{"status", "broker", "--detail"}, true},
		{"logs broker", []string{"logs", "broker"}, true},
		{"cli", []string{"cli"}, true},
		{"shell", []string{"shell"}, true},
		{"start broker", []string{"start", "broker"}, true},
		{"stop broker", []string{"stop", "broker"}, true},
		// restart deletes pods, so it takes the same --no-prompt gate remove does.
		{"restart broker (all)", []string{"restart", "broker", "--no-prompt"}, true},
		{"restart broker backup", []string{"restart", "broker", "backup", "--no-prompt"}, true},
		// roleWord's remaining two cases (backup is covered above): a swapped case
		// would misname which pod the prompt is about to bounce, so both must
		// actually be reached, not assumed from backup's.
		{"restart broker monitor", []string{"restart", "broker", "monitor", "--no-prompt"}, true},
		{"restart broker primary explicit", []string{"restart", "broker", "primary", "--no-prompt"}, true},
		{"deploy broker", []string{"deploy", "broker"}, true},
		{"deploy operator", []string{"deploy", "operator"}, true},
		{"prepare namespace", []string{"prepare", "namespace"}, true},
		{"prepare labels", []string{"prepare", "labels"}, false}, // no placement labels configured
		{"restart operator", []string{"restart", "operator"}, true},
		{"status operator", []string{"status", "operator"}, true},
		{"status operator --detail", []string{"status", "operator", "--detail"}, true},
		{"logs operator", []string{"logs", "operator"}, true},
		{"config disable default-vpn", []string{"config", "disable", "default-vpn"}, true},
		{"config disable default-users", []string{"config", "disable", "default-users"}, true},
		{"config apply domain-certs", []string{"config", "apply", "domain-certs"}, false}, // none configured
		{"cli --input --pod", []string{"cli", "--input", "setup.cli", "--pod", "p"}, true},
		{"copy from", []string{"copy", "from", "somefile", "--pod", "p"}, true},
		{"copy into", []string{"copy", "into", "somefile", "--pod", "p"}, true},
		// Every remove confirms now, secrets and namespace included, so they carry
		// --no-prompt here for the same reason broker/all/operator do.
		{"remove secrets", []string{"remove", "secrets", "--no-prompt"}, true},
		{"remove namespace", []string{"remove", "namespace", "--no-prompt"}, true},
		{"config delete domain-certs", []string{"config", "delete", "domain-certs"}, false}, // none configured
		// remove broker/all/operator carry --no-prompt so no confirm helper reads os.Stdin.
		{"remove broker", []string{"remove", "broker", "--no-prompt"}, true},
		{"remove all", []string{"remove", "all", "--no-prompt"}, true},
		{"remove operator", []string{"remove", "operator", "--no-prompt"}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			args := append([]string{}, tc.args...)
			args = append(args, "--platform", "kubernetes")
			out, err := runRootWith(t, withEnv(args...), echoRunner)
			if err != nil {
				t.Fatalf("%s err = %v, want nil", tc.name, err)
			}
			if tc.wantEcho && !strings.Contains(out, "+ kubectl") {
				t.Errorf("%s stdout = %q, want a '+ kubectl ...' echo", tc.name, out)
			}
			if !tc.wantEcho && strings.Contains(out, "+ kubectl") {
				t.Errorf("%s stdout = %q, want no kubectl echo (skip path)", tc.name, out)
			}
		})
	}
}

// TestK8sStandaloneDryRun covers the commands whose behavior branches on redundancy:
// on a standalone env the HA-only steps self-skip (config leader / smoke redundancy)
// and the secret-bearing prepare steps have no TLS to guard, so config/prepare/deploy
// all run clean over the echo seam.
func TestK8sStandaloneDryRun(t *testing.T) {
	path := writeStandaloneEnv(t)
	cases := []struct {
		name     string
		args     []string
		wantEcho bool
	}{
		{"config leader (skipped)", []string{"config", "leader", "--platform", "kubernetes"}, false},
		{"smoke redundancy (skipped)", []string{"smoke", "redundancy", "--platform", "kubernetes"}, false},
		{"prepare secrets", []string{"prepare", "secrets", "--platform", "kubernetes"}, true},
		{"prepare all", []string{"prepare", "all", "--platform", "kubernetes"}, true},
		{"deploy all", []string{"deploy", "all", "--platform", "kubernetes"}, true},
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

// TestDeployAllDoesNotApplyOperator pins the deliberate behavior change from the
// old `up`: the operator is cluster-scoped and shared between brokers, so
// `deploy all` no longer installs it -- only `deploy operator` does. A standalone
// env keeps this off the HA leader-assertion poll, which the echo seam cannot
// satisfy.
func TestDeployAllDoesNotApplyOperator(t *testing.T) {
	path := writeStandaloneEnv(t)
	out, err := runStandalone(t, path, "deploy", "all", "--platform", "kubernetes")
	if err != nil {
		t.Fatalf("deploy all err = %v, want nil", err)
	}
	// The marker is the CRD permission probe OperatorApply issues before anything
	// else. The operator's DEPLOYMENT NAME is not usable here: `check deploy` reads
	// it back to report whether the operator is installed, so it appears in the
	// echo of a run that installed nothing.
	if strings.Contains(out, "customresourcedefinitions") {
		t.Errorf("deploy all applied the operator bundle:\n%s", out)
	}
}

// TestCheckDeployWarnsWhenOperatorAbsent covers opK8sCheck's operator probe:
// `check deploy` is read-only, so a missing operator is reported as a warning
// rather than failing the check itself. That warning is what stands in for the
// operator install `deploy all` no longer performs.
//
// It cannot go through the echo seam: Echo's Output never returns an error, so the
// probe would always read as "installed". This drives opK8sCheck directly with the
// fault-injecting opRunner instead, failing exactly the CRD lookup and canning the
// StorageClass answers the check needs to get that far.
func TestCheckDeployWarnsWhenOperatorAbsent(t *testing.T) {
	cfg := loadDirect(t, "redundancy: no\n"+
		"image:\n  repo: solace-pubsub-standard\n  tag: \"10.10.1.128\"\n"+
		"admin:\n  pass: "+smokeAdminPass+"\n"+
		"kubernetes:\n  name: dev-broker\n  namespace: solace\n"+
		"  storage:\n    class: standard\n    msgNode: 30Gi\n", config.K8s)

	isCRDLookup := func(c opCall) bool {
		for i, a := range c.args {
			if a == "crd" && i+1 < len(c.args) {
				return true
			}
		}
		return false
	}
	rr := &opRunner{
		fail: func(c opCall) error {
			if isCRDLookup(c) {
				return fmt.Errorf("the server doesn't have a resource type \"crd\"")
			}
			return nil
		},
		output: func(c opCall) []byte {
			// The StorageClass probe reads one custom column at a time; answer both
			// with the values CheckStorageClass demands so it passes and the run
			// reaches the operator probe under test.
			for _, a := range c.args {
				if strings.Contains(a, "volumeBindingMode") {
					return []byte("WaitForFirstConsumer\n")
				}
				if strings.Contains(a, "allowVolumeExpansion") {
					return []byte("true\n")
				}
			}
			return nil
		},
	}
	a := &App{Cfg: cfg, Platform: config.K8s, Runner: rr}

	var err error
	errOut := captureStderr(t, func() { err = opK8sCheck(a) })
	if err != nil {
		t.Fatalf("check deploy err = %v, want nil: a missing operator warns, it does not fail the check", err)
	}
	if !strings.Contains(errOut, "does not look installed") {
		t.Errorf("check deploy stderr = %q, want a warning about the missing operator", errOut)
	}
}

// TestStartStopRestartBroker covers the day-2 start/stop/restart verbs on both
// platform families: Kubernetes scales the statefulset(s), containers start/stop
// the container in place.
func TestStartStopRestartBroker(t *testing.T) {
	t.Run("kubernetes", func(t *testing.T) {
		for _, args := range [][]string{{"start", "broker"}, {"stop", "broker"}} {
			out, err := runRootWith(t, append(withEnv(args...), "--platform", "kubernetes"), echoRunner)
			if err != nil {
				t.Fatalf("%v err = %v, want nil", args, err)
			}
			if !strings.Contains(out, "+ kubectl") {
				t.Errorf("%v stdout = %q, want a kubectl echo", args, out)
			}
		}
		out, err := runRootWith(t, append(withEnv("restart", "broker", "--no-prompt"), "--platform", "kubernetes"), echoRunner)
		if err != nil {
			t.Fatalf("restart broker err = %v, want nil", err)
		}
		if !strings.Contains(out, "+ kubectl") {
			t.Errorf("restart broker stdout = %q, want a kubectl echo", out)
		}
	})
	t.Run("docker", func(t *testing.T) {
		path := writeCtrStandaloneEnv(t)
		for _, args := range [][]string{{"start", "broker"}, {"stop", "broker"}, {"restart", "broker"}} {
			full := append(append([]string{}, args...), "--env", path, "--platform", "docker")
			out, err := runRootWith(t, full, echoRunner)
			if err != nil {
				t.Fatalf("%v err = %v, want nil", args, err)
			}
			if !strings.Contains(out, "+ docker") {
				t.Errorf("%v stdout = %q, want a docker echo", args, out)
			}
		}
	})
}

// TestCLICommand covers `cli`'s two shapes: bare, it opens an interactive session;
// with --input, it uploads and runs a script instead. Both are the same command
// now, distinguished by a flag rather than by being separate subcommands.
func TestCLICommand(t *testing.T) {
	t.Run("bare cli opens a session", func(t *testing.T) {
		out, err := runRootWith(t, append(withEnv("cli"), "--platform", "kubernetes"), echoRunner)
		if err != nil {
			t.Fatalf("cli err = %v, want nil", err)
		}
		if !strings.Contains(out, "+ kubectl") {
			t.Errorf("cli stdout = %q, want a kubectl exec echo", out)
		}
	})
	t.Run("--input runs a script", func(t *testing.T) {
		out, err := runRootWith(t, append(withEnv("cli", "--input", "setup.cli"), "--platform", "kubernetes"), echoRunner)
		if err != nil {
			t.Fatalf("cli --input err = %v, want nil", err)
		}
		if !strings.Contains(out, "+ kubectl") {
			t.Errorf("cli --input stdout = %q, want a kubectl exec echo", out)
		}
	})
}

// TestStatusBrokerFlags covers how --all and --detail compose on `status broker`:
// they widen the report along independent axes (every broker in the cluster vs.
// this env file's one; the static description vs. the running summary) rather
// than one replacing the other.
func TestStatusBrokerFlags(t *testing.T) {
	t.Run("kubernetes --all lists every broker", func(t *testing.T) {
		out, err := runRootWith(t, append(withEnv("status", "broker", "--all"), "--platform", "kubernetes"), echoRunner)
		if err != nil {
			t.Fatalf("status broker --all err = %v, want nil", err)
		}
		if !strings.Contains(out, "+ kubectl") {
			t.Errorf("status broker --all stdout = %q, want a kubectl echo", out)
		}
	})
	t.Run("kubernetes --detail adds the static description", func(t *testing.T) {
		out, err := runRootWith(t, append(withEnv("status", "broker", "--detail"), "--platform", "kubernetes"), echoRunner)
		if err != nil {
			t.Fatalf("status broker --detail err = %v, want nil", err)
		}
		if !strings.Contains(out, "+ kubectl") {
			t.Errorf("status broker --detail stdout = %q, want a kubectl echo", out)
		}
	})
	t.Run("container --detail adds the inspection", func(t *testing.T) {
		path := writeCtrStandaloneEnv(t)
		out, err := runRootWith(t, []string{"status", "broker", "--detail", "--env", path, "--platform", "docker"}, echoRunner)
		if err != nil {
			t.Fatalf("status broker --detail (docker) err = %v, want nil", err)
		}
		if !strings.Contains(out, "+ docker") {
			t.Errorf("status broker --detail (docker) stdout = %q, want a docker echo", out)
		}
	})
}

// TestRemoveBrokerLayerContract covers the retained-layer contract on `remove
// broker`: persistent data is kept unless asked for by name, and --no-prompt alone
// (which only answers the delete-the-broker question) must not also answer the
// delete-the-data question.
func TestRemoveBrokerLayerContract(t *testing.T) {
	path := writeStandaloneEnv(t)
	run := func(configure func(*App), args ...string) string {
		t.Helper()
		full := append(append([]string{}, args...), "--env", path, "--platform", "kubernetes")
		return captureStderr(t, func() {
			_, err := runRootWith(t, full, configure)
			if err != nil {
				t.Fatalf("%v err = %v, want nil", args, err)
			}
		})
	}
	t.Run("--no-prompt keeps data", func(t *testing.T) {
		errOut := run(echoRunner, "remove", "broker", "--no-prompt")
		if !strings.Contains(errOut, "PVCs kept") {
			t.Errorf("remove broker --no-prompt stderr = %q, want data kept", errOut)
		}
	})
	t.Run("--delete-data deletes it", func(t *testing.T) {
		errOut := run(echoRunner, "remove", "broker", "--no-prompt", "--delete-data")
		if !strings.Contains(errOut, "PVCs deleted") {
			t.Errorf("remove broker --delete-data stderr = %q, want data deleted", errOut)
		}
	})
	t.Run("non-interactive keeps data", func(t *testing.T) {
		errOut := run(func(a *App) {
			a.Interactive = func() bool { return false }
			echoRunner(a)
		}, "remove", "broker", "--no-prompt")
		if !strings.Contains(errOut, "PVCs kept") {
			t.Errorf("remove broker --no-prompt stderr = %q, want data kept by default", errOut)
		}
	})
}

// TestRemoveOperatorLayerContract mirrors TestRemoveBrokerLayerContract for the
// operator's CRDs: kept by default (their removal cascades to every broker in the
// cluster), deleted only when named.
func TestRemoveOperatorLayerContract(t *testing.T) {
	path := writeStandaloneEnv(t)
	run := func(args ...string) string {
		t.Helper()
		full := append(append([]string{}, args...), "--env", path, "--platform", "kubernetes")
		return captureStderr(t, func() {
			_, err := runRootWith(t, full, echoRunner)
			if err != nil {
				t.Fatalf("%v err = %v, want nil", args, err)
			}
		})
	}
	t.Run("kept by default", func(t *testing.T) {
		errOut := run("remove", "operator", "--no-prompt")
		if !strings.Contains(errOut, "CRDs kept") {
			t.Errorf("remove operator stderr = %q, want CRDs kept", errOut)
		}
	})
	t.Run("--delete-crd deletes them", func(t *testing.T) {
		errOut := run("remove", "operator", "--no-prompt", "--delete-crd")
		if !strings.Contains(errOut, "CRDs deleted") {
			t.Errorf("remove operator --delete-crd stderr = %q, want CRDs deleted", errOut)
		}
	})
}

// TestRemoveFlagsCompose pins that the two flags answer DIFFERENT questions and so
// must combine rather than conflict: --delete-data says what to do with the data,
// --no-prompt says not to ask about any of it, and a fully unattended removal that
// also drops the data needs both. They were briefly mutually exclusive, which made
// exactly that case impossible to express.
func TestRemoveFlagsCompose(t *testing.T) {
	path := writeStandaloneEnv(t)
	errOut := captureStderr(t, func() {
		_, err := runRootWith(t, []string{"remove", "broker", "--delete-data", "--no-prompt",
			"--env", path, "--platform", "kubernetes"}, func(a *App) {
			a.Interactive = func() bool { return false }
			echoRunner(a)
		})
		if err != nil {
			t.Fatalf("remove broker --delete-data --no-prompt err = %v, want nil", err)
		}
	})
	if !strings.Contains(errOut, "PVCs deleted") {
		t.Errorf("stderr = %q, want the data deleted with nothing asked", errOut)
	}
}

func TestConfirmFlagShortcuts(t *testing.T) {
	if !confirmDelete(&App{noPrompt: true}, "broker x") {
		t.Error("confirmDelete with --no-prompt = false, want true")
	}
	if confirmLayer(&App{noPrompt: true}, layerData) {
		t.Error("confirmLayer with --no-prompt = true, want false (kept)")
	}
	if !confirmLayer(&App{deleteLayer: true}, layerData) {
		t.Error("confirmLayer with --delete-data = false, want true")
	}
}

// TestConfirmNonTTY covers the unattended branches. Pointing os.Stdin at a pipe (not a
// character device) makes isTTY deterministically false, so confirmDelete refuses
// without --no-prompt and confirmLayer keeps the retained layer -- with no prompt
// read, on any host.
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
		t.Error("confirmDelete non-TTY without --no-prompt = true, want false")
	}
	if confirmLayer(&App{}, layerData) {
		t.Error("confirmLayer non-TTY = true, want false (kept)")
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

// TestPromptYes pins the strict layer-deletion confirmation: only an exact
// (trimmed, case-insensitive) "yes" accepts; a bare "y" is not enough.
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
		_, err := runRoot(t, []string{"status", "broker", "--env", "/no/such/file.yaml", "--platform", "kubernetes"})
		if err == nil || !strings.Contains(err.Error(), "not found: looked for") {
			t.Fatalf("bad --env err = %v, want a not-found error", err)
		}
	})
	t.Run("bad container role", func(t *testing.T) {
		_, err := runRoot(t, withEnv("deploy", "broker", "bogus", "--platform", "docker"))
		if err == nil || !strings.Contains(err.Error(), "invalid node role") {
			t.Fatalf("docker deploy broker bogus err = %v, want 'invalid node role'", err)
		}
	})
	t.Run("bad container generate role", func(t *testing.T) {
		_, err := runRoot(t, withEnv("generate", "broker", "bogus", "--platform", "docker"))
		if err == nil || !strings.Contains(err.Error(), "invalid node role") {
			t.Fatalf("docker generate broker bogus err = %v, want 'invalid node role'", err)
		}
	})
	t.Run("bad container deploy-all role", func(t *testing.T) {
		// ParseRole runs in RunE before any host operation, so the bogus role is
		// rejected without the (real) Check/PrepHost/Deploy ever executing.
		_, err := runRoot(t, withEnv("deploy", "all", "bogus", "--platform", "docker"))
		if err == nil || !strings.Contains(err.Error(), "invalid node role") {
			t.Fatalf("docker deploy all bogus err = %v, want 'invalid node role'", err)
		}
	})
	t.Run("bad k8s role leaf", func(t *testing.T) {
		_, err := runRoot(t, withEnv("logs", "broker", "bogus", "--platform", "kubernetes"))
		if err == nil || !strings.Contains(err.Error(), "invalid node role") {
			t.Fatalf("k8s logs broker bogus err = %v, want 'invalid node role'", err)
		}
	})
	t.Run("unknown generate target", func(t *testing.T) {
		// The refusal is group()'s own, not cobra's: cobra would print help and
		// exit 0 for an unknown word on a verb that owns objects.
		_, err := runRoot(t, withEnv("generate", "bogus", "--platform", "kubernetes"))
		if err == nil || !strings.Contains(err.Error(), "bogus") {
			t.Fatalf("generate bogus err = %v, want a refusal naming the unknown word", err)
		}
	})
}

// TestK8sGenSecretsWired covers the renderable Secret manifests via `generate
// secrets`. It uses the standalone env rather than the HA sample, whose
// tls.serverSecret points at cert files that do not exist in a checkout.
func TestK8sGenSecretsWired(t *testing.T) {
	path := writeStandaloneEnv(t)
	out, err := runRoot(t, []string{"generate", "secrets", "--env", path, "--platform", "kubernetes"})
	if err != nil {
		t.Fatalf("generate secrets err = %v, want nil", err)
	}
	if !strings.HasPrefix(out, "apiVersion: v1") {
		t.Errorf("generate secrets should render Secret manifests, got %q", firstLine(out))
	}
	// The manifests carry the value base64-encoded, so the raw password must not
	// appear -- but the rendering is still secret-bearing by design.
	if !strings.Contains(out, "kind: Secret") {
		t.Errorf("generate secrets output is not a Secret manifest:\n%s", out)
	}
}

// TestGenSecretsRefusesEmptyValue: the printed script tells the operator to run
// it, so it must not be printable when running it would create an empty secret --
// `generate secrets` refuses on the same precondition `deploy broker` does. The HA
// sample with its PSK cleared is exactly the pre-`prepare host` state.
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
		_, err := runRoot(t, []string{"generate", "secrets", "--env", path, "--platform", platform})
		if err == nil || !strings.Contains(err.Error(), "nodes.psk") {
			t.Errorf("%s generate secrets with an empty PSK err = %v, want it to name nodes.psk", platform, err)
		}
	}
	// The deploy artifact only references secrets by name, so it stays renderable.
	if _, err := runRoot(t, []string{"generate", "broker", "--env", path, "--platform", "docker"}); err != nil {
		t.Errorf("generate broker should not need the secret values: %v", err)
	}
}

// TestGenNeverLeaksSecrets is the end-to-end guard for the secret
// externalization: the deploy artifacts a user prints, shares, or commits must
// reference the admin password by name, while `generate secrets` is the one
// rendering allowed to carry it.
func TestGenNeverLeaksSecrets(t *testing.T) {
	path := writeCtrStandaloneEnv(t)
	for _, platform := range []string{"docker", "podman"} {
		for _, args := range [][]string{{"generate", "broker"}} {
			out, err := runRoot(t, append(append([]string{}, args...), "--env", path, "--platform", platform))
			if err != nil {
				t.Fatalf("%s %v: %v", platform, args, err)
			}
			if strings.Contains(out, smokeAdminPass) {
				t.Errorf("%s %v leaked the admin password:\n%s", platform, args, out)
			}
		}
		out, err := runRoot(t, []string{"generate", "secrets", "--env", path, "--platform", platform})
		if err != nil {
			t.Fatalf("%s generate secrets: %v", platform, err)
		}
		if !strings.Contains(out, smokeAdminPass) {
			t.Errorf("%s generate secrets must carry the value it creates the secret from:\n%s", platform, out)
		}
	}
}

// TestConfigStepsDoNotLeakSecrets drives each config apply/disable step against a
// container fixture carrying every optional value (server cert/key, a domain CA, a
// product key) and asserts none of them prints the private key material to
// stdout. `config` no longer aggregates these into one run-everything step (there
// is no re-runnable ordering to assume), so each is exercised through its own
// command instead of one combined call -- replacing the old direct-call coverage
// of the now-deleted opCtrConfigAll.
func TestConfigStepsDoNotLeakSecrets(t *testing.T) {
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
	path := filepath.Join(dir, "cfgsteps.yaml")
	content := "redundancy: no\n" +
		"image:\n  repo: solace-pubsub-standard\n  tag: \"10.10.1.128\"\n" +
		"admin:\n  pass: " + smokeAdminPass + "\n" +
		"tls:\n  cert: " + certPath + "\n  certKey: " + keyPath + "\n" +
		"nodes:\n  primary:\n    name: pri-host\n" +
		"docker: {}\n" +
		"broker:\n" +
		"  domainCerts:\n    folder: " + filepath.ToSlash(dir) + "\n    files:\n      myca: myca.pem\n" +
		"  productKeys:\n    - KEY-1\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write env: %v", err)
	}

	for _, args := range [][]string{
		{"config", "apply", "server-cert"},
		{"config", "apply", "domain-certs"},
		{"config", "apply", "product-keys"},
		{"config", "disable", "default-vpn"},
		{"config", "disable", "default-users"},
	} {
		full := append(append([]string{}, args...), "--platform", "docker")
		out, err := runCtr(t, path, full...)
		if err != nil {
			t.Fatalf("%v err = %v, want nil", args, err)
		}
		if strings.Contains(out, keyMaterial) {
			t.Errorf("%v leaked the server-certificate private key to stdout", args)
		}
	}
}

// writeK8sDeployAllEnv builds a minimal, valid k8s env with redundancy
// overridable, so opK8sDeployAll's HA-only final Leader() branch can be
// exercised (writeStandaloneEnv is fixed at redundancy: no).
func writeK8sDeployAllEnv(t *testing.T, redundancy string) *config.Config {
	t.Helper()
	yamlBody := "redundancy: " + redundancy + "\n" +
		"image:\n  repo: solace-pubsub-standard\n  tag: \"10.10.1.128\"\n" +
		"admin:\n  pass: " + smokeAdminPass + "\n" +
		"kubernetes:\n" +
		"  name: dev-broker\n" +
		"  namespace: solace\n" +
		"  adminSecret: solace-admin-secret\n" +
		"  updateStrategy: automatedRolling\n" +
		"  storage:\n    class: standard\n    msgNode: 30Gi\n"
	return loadDirect(t, yamlBody, config.K8s)
}

// k8sDeployAllOutputHook supplies the canned Output content opK8sDeployAll's
// happy path needs once a fake (non-Echo) Runner is in play:
// k8s.Cluster.isDryRun() is a concrete type assertion on engine.Echo, so a fake
// Runner takes CheckStorageClass's real validation branch (unlike engine.Echo,
// which skips it) -- it needs a WaitForFirstConsumer/true StorageClass answer to
// pass, and Leader (HA only) needs a healthy `show redundancy` transcript to
// avoid its real poll budget.
func k8sDeployAllOutputHook(c opCall) []byte {
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

// TestOpK8sDeployAllAssertsLeaderOnHA covers opK8sDeployAll's final branch: on an
// HA config, `deploy all` must assert the config-sync leader as its last step, not
// just deploy the broker and stop. It is unreachable via runRoot/engine.Echo --
// Echo's fixed empty output never satisfies Leader's poll, which would otherwise
// run for broker.New's real 2s x 60 budget -- so this drives opK8sDeployAll
// directly over a fake Runner.
func TestOpK8sDeployAllAssertsLeaderOnHA(t *testing.T) {
	cfg := writeK8sDeployAllEnv(t, "yes")
	rr := &opRunner{output: k8sDeployAllOutputHook}
	a := &App{Cfg: cfg, Platform: config.K8s, Runner: rr}
	var deployErr error
	captureStdout(t, func() { deployErr = opK8sDeployAll(a) })
	if deployErr != nil {
		t.Fatalf("opK8sDeployAll (HA) err = %v, want nil", deployErr)
	}
	if !rr.hasCall("assert-leader") {
		t.Error("opK8sDeployAll did not assert the config-sync leader after deploying an HA broker")
	}
}

// TestOpK8sDeployAllAborts covers opK8sDeployAll's four error-return arms (Check,
// CreateNamespace, CreateSecrets, DeployBroker), each in a separate sub-test that
// fails exactly that step and asserts no later step's command was issued. Unlike
// the retired opK8sUp, there is no operator-apply step: the operator is
// cluster-scoped and installed on its own (`deploy operator`).
func TestOpK8sDeployAllAborts(t *testing.T) {
	t.Run("check fails -> nothing else runs", func(t *testing.T) {
		cfg := writeK8sDeployAllEnv(t, "no")
		rr := &opRunner{fail: opFailOn("version"), output: k8sDeployAllOutputHook}
		a := &App{Cfg: cfg, Platform: config.K8s, Runner: rr}
		var err error
		captureStdout(t, func() { err = opK8sDeployAll(a) })
		if err == nil {
			t.Fatal("opK8sDeployAll = nil, want the injected Check failure to abort")
		}
		if rr.hasCall("apply") {
			t.Error("opK8sDeployAll issued an apply command after Check failed")
		}
	})

	steps := []string{"create-namespace", "create-secrets", "deploy-broker"}
	for i, step := range steps {
		n := i + 1
		t.Run(step+" fails -> the next step never runs", func(t *testing.T) {
			cfg := writeK8sDeployAllEnv(t, "no")
			rr := &opRunner{fail: opFailOnCount("apply", n), output: k8sDeployAllOutputHook}
			a := &App{Cfg: cfg, Platform: config.K8s, Runner: rr}
			var err error
			captureStdout(t, func() { err = opK8sDeployAll(a) })
			if err == nil {
				t.Fatalf("opK8sDeployAll = nil, want the injected %s failure to abort", step)
			}
			if got := rr.callCount("apply"); got != n {
				t.Errorf("opK8sDeployAll issued %d apply command(s) after %s failed, want exactly %d (no later step ran)", got, step, n)
			}
		})
	}
}

// TestPrepLabelsIsInteractiveOnly pins the one command in the tree that cannot be
// scripted, and why. The env file names the label each broker role wants; which
// MACHINE carries it is chosen from a prompt, with no flag to express it. So a
// non-interactive run is refused up front rather than failing deep in the picker on
// an unreadable stdin -- and with nothing configured there is no question to ask,
// so that case stays a no-op even without a terminal.
func TestPrepLabelsIsInteractiveOnly(t *testing.T) {
	t.Run("refused without a terminal when labels are configured", func(t *testing.T) {
		cfg := loadDirect(t, "redundancy: no\n"+
			"image:\n  repo: solace-pubsub-standard\n  tag: \"10.10.1.128\"\n"+
			"admin:\n  pass: "+smokeAdminPass+"\n"+
			"kubernetes:\n  name: dev-broker\n  namespace: solace\n"+
			"  storage:\n    class: standard\n    msgNode: 30Gi\n"+
			"  placement:\n    labelsPrimary: [\"solace-node: primary\"]\n", config.K8s)
		rr := &opRunner{}
		a := &App{Cfg: cfg, Platform: config.K8s, Runner: rr,
			Interactive: func() bool { return false }}
		err := opK8sPrepLabels(a)
		if err == nil || !strings.Contains(err.Error(), "needs a terminal") {
			t.Fatalf("prepare labels err = %v, want a refusal naming the missing terminal", err)
		}
		if len(rr.calls) != 0 {
			t.Errorf("prepare labels touched the cluster before refusing: %+v", rr.calls)
		}
	})
	t.Run("no-op without a terminal when nothing is configured", func(t *testing.T) {
		cfg := writeK8sDeployAllEnv(t, "no")
		rr := &opRunner{}
		a := &App{Cfg: cfg, Platform: config.K8s, Runner: rr,
			Interactive: func() bool { return false }}
		var err error
		captureStderr(t, func() { err = opK8sPrepLabels(a) })
		if err != nil {
			t.Fatalf("prepare labels with no labels configured err = %v, want nil", err)
		}
		if len(rr.calls) != 0 {
			t.Errorf("prepare labels should touch nothing when unconfigured: %+v", rr.calls)
		}
	})
}

// TestDeployAllNeverLabelsNodes: labelling is interactive, so it is out of the
// scripted path entirely. `deploy all` used to run it when placement was configured
// and stdin happened to be a terminal, which made the same command interactive or
// not depending on where it ran.
func TestDeployAllNeverLabelsNodes(t *testing.T) {
	cfg := loadDirect(t, "redundancy: no\n"+
		"image:\n  repo: solace-pubsub-standard\n  tag: \"10.10.1.128\"\n"+
		"admin:\n  pass: "+smokeAdminPass+"\n"+
		"kubernetes:\n  name: dev-broker\n  namespace: solace\n"+
		"  storage:\n    class: standard\n    msgNode: 30Gi\n"+
		"  placement:\n    labelsPrimary: [\"solace-node: primary\"]\n", config.K8s)
	rr := &opRunner{output: func(c opCall) []byte {
		for _, a := range c.args {
			if strings.Contains(a, "volumeBindingMode") {
				return []byte("WaitForFirstConsumer\n")
			}
			if strings.Contains(a, "allowVolumeExpansion") {
				return []byte("true\n")
			}
		}
		return nil
	}}
	a := &App{Cfg: cfg, Platform: config.K8s, Runner: rr,
		Interactive: func() bool { return true }}
	var err error
	captureStdout(t, func() { err = opK8sDeployAll(a) })
	if err != nil {
		t.Fatalf("deploy all err = %v, want nil", err)
	}
	for _, c := range rr.calls {
		if len(c.args) > 1 && c.args[0] == "label" {
			t.Errorf("deploy all labelled a node: %v", c.args)
		}
	}
}

// TestOpK8sPrepAllAborts covers both of opK8sPrepAll's error-return arms
// (CreateNamespace, CreateSecrets): the same abort-ordering property as
// opK8sDeployAll. Those two ARE the whole sequence -- the operator is installed by
// its own command, and node labelling is interactive so it is not in `all` at all.
func TestOpK8sPrepAllAborts(t *testing.T) {
	steps := []string{"create-namespace", "create-secrets"}
	for i, step := range steps {
		n := i + 1
		t.Run(step+" fails -> the next step never runs", func(t *testing.T) {
			cfg := writeK8sDeployAllEnv(t, "no")
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

// TestOpK8sRemoveAllAborts covers opK8sRemoveAll's two error-return arms
// (DeleteBroker, DeleteSecrets): it must not remove the namespace out from under a
// broker- or secrets-deletion that actually failed, leaving orphaned state -- a
// real correctness property of the teardown order, not just error forwarding.
func TestOpK8sRemoveAllAborts(t *testing.T) {
	steps := []struct {
		name string
		n    int
	}{
		{"delete-broker", 1},
		{"delete-secrets", 2},
	}
	for _, s := range steps {
		t.Run(s.name+" fails -> delete-namespace never runs", func(t *testing.T) {
			cfg := writeK8sDeployAllEnv(t, "no")
			rr := &opRunner{fail: opFailOnCount("delete", s.n)}
			a := &App{Cfg: cfg, Platform: config.K8s, Runner: rr, noPrompt: true}
			var err error
			captureStdout(t, func() { err = opK8sRemoveAll(a) })
			if err == nil {
				t.Fatalf("opK8sRemoveAll = nil, want the injected %s failure to abort", s.name)
			}
			if got := rr.callCount("delete"); got != s.n {
				t.Errorf("opK8sRemoveAll issued %d delete command(s) after %s failed, want exactly %d (delete-namespace never ran)", got, s.name, s.n)
			}
		})
	}
}

// TestOpCtrVerifyRedundancyRunsRedundancyLocal covers opCtrVerifyRedundancy's
// actual failover exercise arm (RedundancyLocal) rather than only the skip/reject
// arms already covered by TestCtrRoleGuards. It sets this host as the primary and
// supplies a canned `show redundancy` transcript over a fake Runner, since
// engine.Echo's fixed empty output would otherwise send RedundancyLocal into its
// real poll loop (broker.New's 2s x 60 budget, which the CLI has no seam to
// shorten).
func TestOpCtrVerifyRedundancyRunsRedundancyLocal(t *testing.T) {
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
	captureStdout(t, func() { err = opCtrVerifyRedundancy(a, "") })
	if err == nil || !strings.Contains(err.Error(), "redundancy configuration/status is not healthy") {
		t.Fatalf("opCtrVerifyRedundancy (primary, active-but-unhealthy) err = %v, want the redundancy-unhealthy error", err)
	}
}

// TestK8sSmokeRedundancyUnhealthy covers opK8sVerifyRedundancy's error-return: on
// the HA sample over the echo seam, engine.Echo's empty `show redundancy` output
// makes primaryRedundancyUp false, so Redundancy fails on its first check (no poll).
func TestK8sSmokeRedundancyUnhealthy(t *testing.T) {
	_, err := runRootWith(t, withEnv("smoke", "redundancy", "--platform", "kubernetes"), echoRunner)
	if err == nil || !strings.Contains(err.Error(), "redundancy configuration/status is not healthy") {
		t.Fatalf("k8s smoke redundancy (HA sample) err = %v, want the redundancy-unhealthy error", err)
	}
}

// TestK8sConfigDeleteDomainCertsConfigured covers domainCANames' loop body: every
// other test's domainCerts.files map is empty, so the map-to-slice conversion
// feeding RemoveDomainCerts is trivially correct by vacuity. This configures one
// CA and asserts `config delete domain-certs` actually issues a kubectl exec
// rather than self-skipping.
func TestK8sConfigDeleteDomainCertsConfigured(t *testing.T) {
	path := filepath.Join(t.TempDir(), "domaincerts.yaml")
	content := "redundancy: no\n" +
		"image:\n  repo: solace-pubsub-standard\n  tag: \"10.10.1.128\"\n" +
		"admin:\n  pass: " + smokeAdminPass + "\n" +
		"kubernetes:\n" +
		"  name: dev-broker\n" +
		"  namespace: solace\n" +
		"  adminSecret: solace-admin-secret\n" +
		"  updateStrategy: automatedRolling\n" +
		"  storage:\n    class: standard\n    msgNode: 30Gi\n" +
		"broker:\n" +
		"  domainCerts:\n    folder: certs\n    files:\n      myca: myca.pem\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write env: %v", err)
	}
	out, err := runRootWith(t, []string{"config", "delete", "domain-certs", "--env", path, "--platform", "kubernetes"}, echoRunner)
	if err != nil {
		t.Fatalf("config delete domain-certs (configured) err = %v, want nil", err)
	}
	if !strings.Contains(out, "+ kubectl") {
		t.Errorf("config delete domain-certs (configured) stdout = %q, want a '+ kubectl ...' echo", out)
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
	out, err := runCtr(t, dst, "status", "broker", "--platform", "docker")
	if err != nil {
		t.Fatalf("docker status broker against the converted env err = %v, want nil", err)
	}
	if !strings.Contains(out, "+ docker") {
		t.Errorf("converted env did not drive a real command:\n%s", out)
	}
}

func TestConvertErrorPaths(t *testing.T) {
	src := writeBashEnv(t)
	// convert reads the same --platform every other command does, so the rejection
	// is config.ParsePlatform's own and names the platform rather than the flag --
	// one word, one meaning, whichever command it was typed on.
	t.Run("bad platform", func(t *testing.T) {
		_, err := runRoot(t, []string{"convert", src, "--platform", "bogus"})
		if err == nil || !strings.Contains(err.Error(), "invalid platform") {
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

// TestVersionPrintsStampedValue: `version` reports whatever the linker (the
// dev scripts' -X flag) set the package var to, verbatim -- the contract the
// release binaries depend on to match the git tag.
func TestVersionPrintsStampedValue(t *testing.T) {
	old := version
	version = "v9.9.9-test"
	t.Cleanup(func() { version = old })

	out, err := runRoot(t, []string{"version"})
	if err != nil {
		t.Fatalf("version: %v", err)
	}
	if !strings.Contains(out, "v9.9.9-test") {
		t.Errorf("output = %q, want it to contain the stamped version", out)
	}
}

// TestVersionDefaultsToDev: an unstamped build -- this package's own `go test`,
// or a plain `go build .` -- reports "dev".
func TestVersionDefaultsToDev(t *testing.T) {
	out, err := runRoot(t, []string{"version"})
	if err != nil {
		t.Fatalf("version: %v", err)
	}
	if !strings.HasPrefix(out, "solace-util dev ") {
		t.Errorf("output = %q, want it to start with %q", out, "solace-util dev ")
	}
}

// TestVersionIncludesToolchainAndPlatform: support triage needs the Go
// toolchain and OS/arch that built the binary alongside the tag.
func TestVersionIncludesToolchainAndPlatform(t *testing.T) {
	out, err := runRoot(t, []string{"version"})
	if err != nil {
		t.Fatalf("version: %v", err)
	}
	for _, want := range []string{runtime.Version(), runtime.GOOS, runtime.GOARCH} {
		if !strings.Contains(out, want) {
			t.Errorf("output = %q, want it to contain %q", out, want)
		}
	}
}

// TestVersionRejectsArgs: NoArgs is enforced like every other bare leaf.
func TestVersionRejectsArgs(t *testing.T) {
	if _, err := runRoot(t, []string{"version", "extra"}); err == nil {
		t.Error("version with an argument: want an error, got nil")
	}
}

// TestBashEnvGivenToEnvFlag is the other half of the migration story: pointing
// -e at a legacy bash file must say it is not YAML and name the converter.
func TestBashEnvGivenToEnvFlag(t *testing.T) {
	src := writeBashEnv(t)
	_, err := runRoot(t, []string{"status", "broker", "-e", src, "--platform", "kubernetes"})
	if err == nil {
		t.Fatal("a bash env file should not load")
	}
	for _, want := range []string{"not valid YAML", "this looks like a legacy bash env file", "solace-util convert"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not mention %q", err.Error(), want)
		}
	}
}

func TestExecute(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"solace-util", "--help"}
	var err error
	captureStdout(t, func() { err = Execute() })
	if err != nil {
		t.Fatalf("Execute(--help) err = %v, want nil", err)
	}
}

// TestK8sConfirmDeclined covers the confirm-declined branch of every removal: a
// non-interactive run without --no-prompt must issue zero cluster calls -- the
// actual safety default.
//
// secrets and namespace are here because they are the two that used to run with no
// confirmation at all. `remove namespace` deletes everything that happens to live
// in the namespace, not only what this env file put there, so an unattended run
// reaching kubectl is the single worst outcome in the tree.
func TestK8sConfirmDeclined(t *testing.T) {
	cases := []struct {
		name string
		args []string
	}{
		{"remove broker declined", []string{"remove", "broker", "--platform", "kubernetes"}},
		{"remove all declined", []string{"remove", "all", "--platform", "kubernetes"}},
		{"remove secrets declined", []string{"remove", "secrets", "--platform", "kubernetes"}},
		{"remove namespace declined", []string{"remove", "namespace", "--platform", "kubernetes"}},
		{"remove operator declined", []string{"remove", "operator", "--platform", "kubernetes"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out, err := runRootWith(t, withEnv(tc.args...), func(a *App) {
				a.Interactive = func() bool { return false }
				echoRunner(a)
			})
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
// non-interactive run (no --no-prompt) must bounce nothing, whether restarting every pod
// or a single one, and a bad role is rejected before any prompt is even possible.
func TestK8sRestartConfirmGate(t *testing.T) {
	t.Run("restart broker (all) declined (no --no-prompt)", func(t *testing.T) {
		out, err := runRootWith(t, withEnv("restart", "broker", "--platform", "kubernetes"), func(a *App) {
			a.Interactive = func() bool { return false }
			echoRunner(a)
		})
		if err != nil {
			t.Fatalf("restart broker declined err = %v, want nil", err)
		}
		if strings.Contains(out, "+ kubectl") {
			t.Errorf("restart broker declined stdout = %q, want no kubectl echo", out)
		}
	})
	t.Run("restart broker one role declined (no --no-prompt)", func(t *testing.T) {
		out, err := runRootWith(t, withEnv("restart", "broker", "backup", "--platform", "kubernetes"), func(a *App) {
			a.Interactive = func() bool { return false }
			echoRunner(a)
		})
		if err != nil {
			t.Fatalf("restart broker backup declined err = %v, want nil", err)
		}
		if strings.Contains(out, "+ kubectl") {
			t.Errorf("restart broker backup declined stdout = %q, want no kubectl echo", out)
		}
	})
	t.Run("bad role rejected before any prompt", func(t *testing.T) {
		_, err := runRoot(t, withEnv("restart", "broker", "bogus", "--platform", "kubernetes"))
		if err == nil || !strings.Contains(err.Error(), "invalid node role") {
			t.Fatalf("restart broker bogus err = %v, want 'invalid node role'", err)
		}
	})
}

// TestCtrConfirmDeclined covers opCtrDelete's confirm-declined branch, the
// container-side counterpart of TestK8sConfirmDeclined: a non-interactive removal
// (no --no-prompt) must issue zero runtime calls.
func TestCtrConfirmDeclined(t *testing.T) {
	path := writeCtrStandaloneEnv(t)
	out, err := runRootWith(t, []string{"remove", "broker", "--env", path, "--platform", "docker"}, func(a *App) {
		a.Interactive = func() bool { return false }
		echoRunner(a)
	})
	if err != nil {
		t.Fatalf("docker remove broker declined err = %v, want nil", err)
	}
	if strings.Contains(out, "+ docker") {
		t.Errorf("docker remove broker declined stdout = %q, want no docker echo", out)
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

// TestSecretsNeverEchoed is the S3 smoke check: a secret-bearing command over the
// echo seam must show its stdin as a byte count, never the secret value. `check
// semp-login` puts the admin credential on curl's stdin; the login itself fails
// under Echo (no broker), which is fine -- the assertion is purely that the
// password does not reach stdout.
func TestSecretsNeverEchoed(t *testing.T) {
	path := writeStandaloneEnv(t)
	out, _ := runStandalone(t, path, "check", "semp-login", "--platform", "kubernetes")
	if !strings.Contains(out, "bytes on stdin") {
		t.Errorf("check semp-login stdout = %q, want a 'bytes on stdin' redaction", out)
	}
	if strings.Contains(out, smokeAdminPass) {
		t.Error("check semp-login leaked the admin password to stdout")
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
		"kubernetes:\n" +
		"  name: dev-broker\n" +
		"  namespace: solace\n" +
		"  adminSecret: solace-admin-secret\n" +
		"  updateStrategy: automatedRolling\n" +
		"  storage:\n    class: standard\n    msgNode: 30Gi\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write env: %v", err)
	}
	_, err := runRoot(t, []string{"generate", "secrets", "--env", path, "--platform", "kubernetes"})
	if err == nil || !strings.Contains(err.Error(), "read tls.cert") {
		t.Fatalf("k8s generate secrets (missing cert file) err = %v, want it to wrap the tls.cert read failure", err)
	}
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}

// fakeBinaryOnPath writes an executable stub named base into a fresh directory, puts
// that directory at the front of PATH for this test, and returns the bare name plus the
// absolute path engine.Resolve will find. It makes the announcement assertions hermetic:
// no test host needs kubectl or docker installed, and the expected path is exact rather
// than "something absolute".
func fakeBinaryOnPath(t *testing.T, base string) (name, path string) {
	t.Helper()
	dir := t.TempDir()
	file, body, mode := base, "#!/bin/sh\nexit 0\n", os.FileMode(0o755)
	if runtime.GOOS == "windows" {
		// LookPath resolves a bare name through PATHEXT, which includes .BAT -- the
		// same assumption engine's TestResolveRefusesCurrentDirectory makes.
		file, body, mode = base+".bat", "@echo off\r\nexit /b 0\r\n", 0o666
	}
	path = filepath.Join(dir, file)
	if err := os.WriteFile(path, []byte(body), mode); err != nil {
		t.Fatalf("write fake binary: %v", err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	return base, path
}

// allowRuntime approves a fake binary for this config the only way the allowlist can be
// widened -- the operator's own --allow-command, here through its one entry point.
func allowRuntime(t *testing.T, cfg *config.Config, names ...string) {
	t.Helper()
	if err := cfg.AllowCommands(names); err != nil {
		t.Fatalf("AllowCommands(%v): %v", names, err)
	}
}

// TestAnnounceCommandsNamesResolvedBinaries covers the preamble that replaced the
// per-call `exec:` line: the binaries an env file chose are resolved and named ONCE, up
// front, so the location the allowlist cannot guarantee is still visible without
// repeating itself between report lines on every command.
func TestAnnounceCommandsNamesResolvedBinaries(t *testing.T) {
	t.Run("k8s names the cluster CLI", func(t *testing.T) {
		name, path := fakeBinaryOnPath(t, "solace-fake-kube")
		cfg, err := config.Load(writeStandaloneEnv(t), config.K8s)
		if err != nil {
			t.Fatalf("Load: %v", err)
		}
		cfg.K8s.Runtime = config.Command{name}
		allowRuntime(t, cfg, name)
		a := &App{Cfg: cfg, Platform: config.K8s}
		got := captureStderr(t, a.announceCommands)
		if want := "==> using " + name + ": " + path + "\n"; got != want {
			t.Errorf("announcement =\n%q\nwant\n%q", got, want)
		}
	})
	// docker.compose defaults to the runtime's own `compose` subcommand, so argv[0] is
	// one binary announced once -- naming it twice would read as two installs.
	t.Run("docker names one binary when compose is derived", func(t *testing.T) {
		name, path := fakeBinaryOnPath(t, "solace-fake-docker")
		cfg, err := config.Load(writeCtrStandaloneEnv(t), config.Docker)
		if err != nil {
			t.Fatalf("Load: %v", err)
		}
		cfg.Docker.Runtime = config.Command{name}
		cfg.Docker.Compose = config.Command{name, "compose"}
		allowRuntime(t, cfg, name)
		a := &App{Cfg: cfg, Platform: config.Docker}
		got := captureStderr(t, a.announceCommands)
		if want := "==> using " + name + ": " + path + "\n"; got != want {
			t.Errorf("announcement =\n%q\nwant\n%q", got, want)
		}
	})
	// A host carrying only the standalone compose v1 binary runs two different
	// binaries, and both locations matter.
	t.Run("docker names a standalone compose binary too", func(t *testing.T) {
		name, path := fakeBinaryOnPath(t, "solace-fake-docker")
		compose, composePath := fakeBinaryOnPath(t, "solace-fake-compose")
		cfg, err := config.Load(writeCtrStandaloneEnv(t), config.Docker)
		if err != nil {
			t.Fatalf("Load: %v", err)
		}
		cfg.Docker.Runtime = config.Command{name}
		cfg.Docker.Compose = config.Command{compose}
		allowRuntime(t, cfg, name, compose)
		a := &App{Cfg: cfg, Platform: config.Docker}
		got := captureStderr(t, a.announceCommands)
		want := "==> using " + name + ": " + path + "\n" +
			"==> using " + compose + ": " + composePath + "\n"
		if got != want {
			t.Errorf("announcement =\n%q\nwant\n%q", got, want)
		}
	})
	// A name that resolves nowhere is skipped in silence: this is a report, and turning
	// it into a failure would break a command whose runner the operator never reaches.
	// The first real execution still fails with engine.Resolve's own message.
	t.Run("an unresolvable binary is skipped silently", func(t *testing.T) {
		cfg, err := config.Load(writeStandaloneEnv(t), config.K8s)
		if err != nil {
			t.Fatalf("Load: %v", err)
		}
		cfg.K8s.Runtime = config.Command{"solace-absent-kube"}
		allowRuntime(t, cfg, "solace-absent-kube")
		a := &App{Cfg: cfg, Platform: config.K8s}
		if got := captureStderr(t, a.announceCommands); got != "" {
			t.Errorf("announcement = %q, want nothing for a binary that resolves nowhere", got)
		}
	})
}

// TestBinaryAnnouncementWiring drives the preamble through the real command tree. The
// stub is named `kubectl` so it is the schema default and needs no --allow-command --
// which also keeps the negative cases honest, since --allow-command is itself refused on
// a command that renders without executing.
//
// The negative half matters as much as the positive: `generate` is documented to need no
// kubectl/docker/podman installed at all, since it never executes anything, and
// resolving one there would contradict the promise.
func TestBinaryAnnouncementWiring(t *testing.T) {
	const marker = "==> using "

	t.Run("a real run announces before it works", func(t *testing.T) {
		_, path := fakeBinaryOnPath(t, "kubectl")
		env := writeStandaloneEnv(t)
		// The stub answers every call with empty stdout, so `check deploy` fails at
		// the storage-class assertion -- after the announcement, which is what is
		// asserted.
		got := captureStderr(t, func() {
			_, _ = runRoot(t, []string{"check", "deploy", "--env", env, "--platform", "kubernetes"})
		})
		if want := marker + "kubectl: " + path; !strings.Contains(got, want) {
			t.Errorf("stderr = %q, want it to carry %q", got, want)
		}
	})
	t.Run("nothing is announced where nothing executes", func(t *testing.T) {
		fakeBinaryOnPath(t, "kubectl")
		env := writeStandaloneEnv(t)
		for _, args := range [][]string{
			{"generate", "broker"},
			{"generate", "operator"},
		} {
			full := append(append([]string{}, args...), "--env", env, "--platform", "kubernetes")
			got := captureStderr(t, func() { _, _ = runRoot(t, full) })
			if strings.Contains(got, marker) {
				t.Errorf("%v announced a binary it never runs: %q", args, got)
			}
		}
	})
}

// TestVerboseFlagTracesEveryCommand: -v is the opt-in per-call trail that replaced the
// unconditional `exec:` line. It answers "what exactly did this run issue?", which the
// once-per-binary preamble deliberately does not.
func TestVerboseFlagTracesEveryCommand(t *testing.T) {
	fakeBinaryOnPath(t, "kubectl")
	env := writeStandaloneEnv(t)

	traced := captureStderr(t, func() {
		_, _ = runRoot(t, []string{"check", "deploy", "-v", "--env", env, "--platform", "kubernetes"})
	})
	if !strings.Contains(traced, "==> exec: ") || !strings.Contains(traced, "version -o json") {
		t.Errorf("-v stderr = %q, want a `==> exec: <path> version -o json` line", traced)
	}
	// The default run stays quiet per call: the preamble already named the binary.
	quiet := captureStderr(t, func() {
		_, _ = runRoot(t, []string{"check", "deploy", "--env", env, "--platform", "kubernetes"})
	})
	if strings.Contains(quiet, "==> exec: ") {
		t.Errorf("a run without -v traced its commands: %q", quiet)
	}
	// And it is a no-op with the test-only echo seam installed too, where Echo
	// already prints every command it would run -- passing both must still work
	// rather than fight.
	out, err := runRootWith(t, []string{"check", "deploy", "-v", "--env", env, "--platform", "kubernetes"}, echoRunner)
	if err != nil {
		t.Fatalf("echoRunner + -v: %v", err)
	}
	if !strings.Contains(out, "+ kubectl version -o json") {
		t.Errorf("echoRunner + -v stdout = %q, want the echoed command", out)
	}
}

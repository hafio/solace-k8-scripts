package k8s

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"solace/internal/broker"
	"solace/internal/config"
	"solace/internal/engine"
)

// --- capturing fake engine.Runner ------------------------------------------

type rrCall struct {
	method string // Run | RunInput | RunInteractive | Output | OutputInput
	name   string
	args   []string
	stdin  string
}

type recRunner struct {
	calls       []rrCall
	out         []byte   // returned by Output/OutputInput when outQueue is exhausted
	outQueue    [][]byte // consumed in order by successive Output calls (for multi-read ops)
	outErr      error    // error returned by Output/OutputInput (nil = success)
	outErrQueue []error  // popped per Output/OutputInput call; falls back to outErr once drained
	runErr      error    // error returned by Run (nil = success); for best-effort paths
	runErrQueue []error  // popped per Run call; falls back to runErr once drained
	runInputErr error    // error returned by RunInput (nil = success)

	// Cluster.Preflight runs `auth can-i` before every mutating operation, so the
	// double answers it out of band: the reply comes from these two fields rather
	// than from out/outQueue, which keeps every pre-existing scripted read aligned
	// with the call it was written for. Empty canI means "yes" -- the permitted
	// case, which is what almost every test is about. A test that exercises a
	// refusal sets canI="no"; one that exercises an unreachable server sets canIErr.
	canI    string
	canIErr error
}

// isCanI reports whether an argv is the Preflight probe. Matched on the token
// rather than a position so a configured kubernetes.runtime with leading arguments still
// resolves to the same answer.
func isCanI(args []string) bool {
	for _, a := range args {
		if a == "can-i" {
			return true
		}
	}
	return false
}

// canIAnswer is the scripted Preflight reply, defaulting to the permitted case.
func (r *recRunner) canIAnswer() ([]byte, error) {
	if r.canI == "" {
		return []byte("yes\n"), r.canIErr
	}
	return []byte(r.canI + "\n"), r.canIErr
}

// nextOut returns the next queued Output body, falling back to out once the queue
// is drained, so a test can script differing results for sequential kubectl reads.
func (r *recRunner) nextOut() []byte {
	if len(r.outQueue) > 0 {
		o := r.outQueue[0]
		r.outQueue = r.outQueue[1:]
		return o
	}
	return r.out
}

// nextRunErr returns the next queued Run error, falling back to runErr once the
// queue is drained, so a test can fail one specific call in a multi-step op (e.g.
// let a scale succeed but the rollout that follows it fail).
func (r *recRunner) nextRunErr() error {
	if len(r.runErrQueue) > 0 {
		e := r.runErrQueue[0]
		r.runErrQueue = r.runErrQueue[1:]
		return e
	}
	return r.runErr
}

// nextOutErr returns the next queued Output/OutputInput error, falling back to
// outErr once the queue is drained -- the Output-side mirror of nextRunErr, so a
// test can fail one specific read in a multi-step op (e.g. let the RBAC precheck
// succeed but the node listing that follows it fail).
func (r *recRunner) nextOutErr() error {
	if len(r.outErrQueue) > 0 {
		e := r.outErrQueue[0]
		r.outErrQueue = r.outErrQueue[1:]
		return e
	}
	return r.outErr
}

func (r *recRunner) Run(_ context.Context, name string, args ...string) error {
	r.calls = append(r.calls, rrCall{"Run", name, args, ""})
	return r.nextRunErr()
}
func (r *recRunner) RunInput(_ context.Context, in []byte, name string, args ...string) error {
	r.calls = append(r.calls, rrCall{"RunInput", name, args, string(in)})
	return r.runInputErr
}
// RunEnv exists to satisfy engine.Runner: nothing in the k8s package passes
// secrets through a child environment (every secret rides stdin), so it records
// the call like Run and the extra environment is deliberately not modelled.
func (r *recRunner) RunEnv(_ context.Context, _ []string, name string, args ...string) error {
	r.calls = append(r.calls, rrCall{"RunEnv", name, args, ""})
	return r.nextRunErr()
}
func (r *recRunner) RunInteractive(_ context.Context, name string, args ...string) error {
	r.calls = append(r.calls, rrCall{"RunInteractive", name, args, ""})
	return nil
}
func (r *recRunner) Output(_ context.Context, name string, args ...string) ([]byte, error) {
	r.calls = append(r.calls, rrCall{"Output", name, args, ""})
	if isCanI(args) {
		return r.canIAnswer()
	}
	return r.nextOut(), r.nextOutErr()
}
func (r *recRunner) OutputInput(_ context.Context, in []byte, name string, args ...string) ([]byte, error) {
	r.calls = append(r.calls, rrCall{"OutputInput", name, args, string(in)})
	return r.out, r.nextOutErr()
}

// afterPreflight asserts the FIRST recorded call is the read-only permission probe
// for verb/resource, and returns everything recorded after it. Every mutating
// Cluster operation must ask before it acts, so this does double duty: it pins that
// ordering, and it lets the call-shape assertions below keep counting only the work
// they are actually about. A test that expects no probe simply reads r.calls.
func (r *recRunner) afterPreflight(t *testing.T, verb, resource string) []rrCall {
	t.Helper()
	if len(r.calls) == 0 {
		t.Fatalf("no calls recorded: the read-only `auth can-i %s %s` probe must run before anything else", verb, resource)
	}
	first := r.calls[0]
	want := []string{"auth", "can-i", verb, resource, "-n", "solace"}
	if first.method != "Output" || !eqArgs(first.args, want) {
		t.Fatalf("first call = %+v, want the preflight probe Output %v", first, want)
	}
	return r.calls[1:]
}

func (r *recRunner) last() rrCall {
	if len(r.calls) == 0 {
		return rrCall{}
	}
	return r.calls[len(r.calls)-1]
}

func eqArgs(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// --- exec argv (Run / Output / OutputInput), no -c ever --------------------

func TestTransportExecArgs(t *testing.T) {
	cfg := haCfg()
	rr := &recRunner{}
	tr := NewTransport(rr, cfg)
	ctx := context.Background()

	cases := []struct {
		name string
		call func()
		want rrCall
	}{
		{
			"Run primary",
			func() { _ = tr.Run(ctx, config.Primary, "show", "version") },
			rrCall{"Run", "kubectl", []string{"exec", "-n", "solace", "dev-broker-pubsubplus-p-0", "--", "show", "version"}, ""},
		},
		{
			"Output backup",
			func() { _, _ = tr.Output(ctx, config.Backup, "hostname") },
			rrCall{"Output", "kubectl", []string{"exec", "-n", "solace", "dev-broker-pubsubplus-b-0", "--", "hostname"}, ""},
		},
		{
			"OutputInput monitor adds -i and rides stdin",
			func() { _, _ = tr.OutputInput(ctx, config.Monitor, []byte("user = \"a:b\"\n"), "curl", "-K", "-") },
			rrCall{"OutputInput", "kubectl", []string{"exec", "-i", "-n", "solace", "dev-broker-pubsubplus-m-0", "--", "curl", "-K", "-"}, "user = \"a:b\"\n"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tc.call()
			got := rr.last()
			if got.method != tc.want.method || got.name != tc.want.name || !eqArgs(got.args, tc.want.args) || got.stdin != tc.want.stdin {
				t.Errorf("%s\n got: %+v\nwant: %+v", tc.name, got, tc.want)
			}
			for _, a := range got.args {
				if a == "-c" {
					t.Error("broker pods are single-container: -c must never appear")
				}
			}
		})
	}
}

// --- Upload: body on stdin via `sh -c 'cat > <dest>'`, never in argv --------

func TestTransportUpload(t *testing.T) {
	cfg := haCfg()
	rr := &recRunner{}
	tr := NewTransport(rr, cfg)

	secret := "PRIVATE-KEY-MATERIAL"
	dest := "/usr/sw/jail/certs/tls-2026-07-31.crt.key"
	if err := tr.Upload(context.Background(), config.Primary, []byte(secret), dest); err != nil {
		t.Fatalf("Upload: %v", err)
	}
	got := rr.last()
	want := rrCall{
		"RunInput", "kubectl",
		[]string{"exec", "-i", "-n", "solace", "dev-broker-pubsubplus-p-0", "--", "sh", "-c", "cat > '" + dest + "'"},
		secret,
	}
	if got.method != want.method || got.name != want.name || !eqArgs(got.args, want.args) || got.stdin != want.stdin {
		t.Errorf("Upload\n got: %+v\nwant: %+v", got, want)
	}
	if strings.Contains(strings.Join(got.args, " "), secret) {
		t.Error("secret body leaked into the Upload argv")
	}
}

// TestTransportUploadQuotesDest guards the defensive single-quote escaping so a
// metacharacter in a path cannot break out of the `cat >` redirect.
func TestTransportUploadQuotesDest(t *testing.T) {
	rr := &recRunner{}
	tr := NewTransport(rr, haCfg())
	if err := tr.Upload(context.Background(), config.Primary, []byte("x"), "/tmp/a'b"); err != nil {
		t.Fatalf("Upload: %v", err)
	}
	shArg := rr.last().args[len(rr.last().args)-1]
	if shArg != `cat > '/tmp/a'\''b'` {
		t.Errorf("dest quoting = %q", shArg)
	}
}

// --- kubectl cp, both directions, with -n <ns> ------------------------------

func TestTransportCopy(t *testing.T) {
	cfg := haCfg()
	rr := &recRunner{}
	tr := NewTransport(rr, cfg)
	ctx := context.Background()

	if err := tr.UploadFile(ctx, config.Primary, "local/in.cli", "/usr/sw/jail/cliscripts/.x.cli"); err != nil {
		t.Fatalf("UploadFile: %v", err)
	}
	up := rr.last()
	wantUp := []string{"cp", "-n", "solace", "local/in.cli", "dev-broker-pubsubplus-p-0:/usr/sw/jail/cliscripts/.x.cli"}
	if up.method != "Run" || up.name != "kubectl" || !eqArgs(up.args, wantUp) {
		t.Errorf("UploadFile argv\n got: %+v\nwant cp %v", up, wantUp)
	}

	if err := tr.Download(ctx, config.Monitor, "/usr/sw/jail/logs/diag.tgz", "out/diag.tgz"); err != nil {
		t.Fatalf("Download: %v", err)
	}
	dn := rr.last()
	wantDn := []string{"cp", "-n", "solace", "dev-broker-pubsubplus-m-0:/usr/sw/jail/logs/diag.tgz", "out/diag.tgz"}
	if dn.method != "Run" || dn.name != "kubectl" || !eqArgs(dn.args, wantDn) {
		t.Errorf("Download argv\n got: %+v\nwant cp %v", dn, wantDn)
	}
}

// --- end-to-end: a real broker.Ops over engine.Echo never echoes the body ---

func TestTransportEchoHidesUploadBody(t *testing.T) {
	buf := &bytes.Buffer{}
	cfg := haCfg()
	tr := NewTransport(engine.Echo{W: buf}, cfg)
	o := broker.New(tr, cfg, nil)

	body := "SECRET-CLI-BODY\n"
	if _, err := o.RunCLI(context.Background(), config.Primary, "probe", body); err != nil {
		t.Fatalf("RunCLI over Echo: %v", err)
	}
	out := buf.String()
	if strings.Contains(out, "SECRET-CLI-BODY") {
		t.Errorf("Echo leaked the uploaded body:\n%s", out)
	}
	if !strings.Contains(out, "bytes on stdin") {
		t.Errorf("Echo should show the upload as a byte count:\n%s", out)
	}
	// The CLI exec itself is echoed as a normal command against the primary pod.
	if !strings.Contains(out, "dev-broker-pubsubplus-p-0 -- "+broker.CLIBinary+" -Apes .probe.cli") {
		t.Errorf("Echo missing the cli exec line:\n%s", out)
	}
}

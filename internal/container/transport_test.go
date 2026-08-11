package container

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"

	"solace/internal/broker"
	"solace/internal/config"
	"solace/internal/engine"
)

// --- capturing fake engine.Runner ------------------------------------------

type capCall struct {
	method string // Run | RunInput | RunInteractive | Output | OutputInput
	name   string
	args   []string
	stdin  string
}

type capRunner struct {
	calls  []capCall
	out    []byte
	outErr error
	// fail, when set, is consulted by the Run-family methods after recording the
	// call: a non-nil return is propagated as the command's error, so a test can
	// drive the manager's error-wrap branches for a specific command. nil (the
	// default) always succeeds, leaving existing tests unchanged.
	fail func(name string, args []string) error
}

func (r *capRunner) Run(_ context.Context, name string, args ...string) error {
	r.calls = append(r.calls, capCall{"Run", name, args, ""})
	return r.failFor(name, args)
}
func (r *capRunner) RunInput(_ context.Context, in []byte, name string, args ...string) error {
	r.calls = append(r.calls, capCall{"RunInput", name, args, string(in)})
	return r.failFor(name, args)
}
func (r *capRunner) RunInteractive(_ context.Context, name string, args ...string) error {
	r.calls = append(r.calls, capCall{"RunInteractive", name, args, ""})
	return r.failFor(name, args)
}
func (r *capRunner) failFor(name string, args []string) error {
	if r.fail == nil {
		return nil
	}
	return r.fail(name, args)
}
func (r *capRunner) Output(_ context.Context, name string, args ...string) ([]byte, error) {
	r.calls = append(r.calls, capCall{"Output", name, args, ""})
	return r.out, r.outErr
}
func (r *capRunner) OutputInput(_ context.Context, in []byte, name string, args ...string) ([]byte, error) {
	r.calls = append(r.calls, capCall{"OutputInput", name, args, string(in)})
	return r.out, r.outErr
}
func (r *capRunner) last() capCall {
	if len(r.calls) == 0 {
		return capCall{}
	}
	return r.calls[len(r.calls)-1]
}

// failOn builds a capRunner.fail hook that errors when the command name or any of
// its args contains substr, so a test can target one command (e.g. "daemon-reload",
// "chown", "compose") and assert the manager wraps its failure.
func failOn(substr string) func(name string, args []string) error {
	return func(name string, args []string) error {
		if strings.Contains(name, substr) {
			return fmt.Errorf("injected failure for %q", name)
		}
		for _, a := range args {
			if strings.Contains(a, substr) {
				return fmt.Errorf("injected failure for %q", substr)
			}
		}
		return nil
	}
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

func dockerCfg() *config.Config {
	c := &config.Config{}
	c.Docker.Runtime = config.Command{"docker"}
	c.Docker.Container.Name = "solace"
	return c
}

func podmanCfg() *config.Config {
	c := &config.Config{}
	c.Podman.Runtime = config.Command{"podman"}
	c.Podman.Container.Name = "sol-pod"
	return c
}

// --- exec argv: `<runtime> exec [-i] <name> argv...`, never a `--` -----------

func TestTransportExecArgs(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		name    string
		cfg     *config.Config
		p       config.Platform
		call    func(tr broker.Transport)
		want    capCall
	}{
		{
			"docker Run has no -- and no -i",
			dockerCfg(), config.Docker,
			func(tr broker.Transport) { _ = tr.Run(ctx, config.Primary, "show", "version") },
			capCall{"Run", "docker", []string{"exec", "solace", "show", "version"}, ""},
		},
		{
			"docker Output ignores role (node-local)",
			dockerCfg(), config.Docker,
			func(tr broker.Transport) { _, _ = tr.Output(ctx, config.Backup, "hostname") },
			capCall{"Output", "docker", []string{"exec", "solace", "hostname"}, ""},
		},
		{
			"podman OutputInput adds -i and rides stdin",
			podmanCfg(), config.Podman,
			func(tr broker.Transport) {
				_, _ = tr.OutputInput(ctx, config.Monitor, []byte("user = \"a:b\"\n"), "curl", "-K", "-")
			},
			capCall{"OutputInput", "podman", []string{"exec", "-i", "sol-pod", "curl", "-K", "-"}, "user = \"a:b\"\n"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rr := &capRunner{}
			tr := NewTransport(rr, tc.cfg, tc.p)
			tc.call(tr)
			got := rr.last()
			if got.method != tc.want.method || got.name != tc.want.name || !eqArgs(got.args, tc.want.args) || got.stdin != tc.want.stdin {
				t.Errorf("%s\n got: %+v\nwant: %+v", tc.name, got, tc.want)
			}
			for _, a := range got.args {
				if a == "--" {
					t.Error("`--` must never appear in a container exec (docker exec rejects it)")
				}
			}
		})
	}
}

// --- Upload: body on stdin via `sh -c 'cat > <dest>'`, never in argv ---------

func TestTransportUpload(t *testing.T) {
	rr := &capRunner{}
	tr := NewTransport(rr, podmanCfg(), config.Podman)

	secret := "PRIVATE-KEY-MATERIAL"
	dest := "/usr/sw/jail/certs/tls-2026-08-03.crt.key"
	if err := tr.Upload(context.Background(), config.Primary, []byte(secret), dest); err != nil {
		t.Fatalf("Upload: %v", err)
	}
	got := rr.last()
	want := capCall{
		"RunInput", "podman",
		[]string{"exec", "-i", "sol-pod", "sh", "-c", "cat > '" + dest + "'"},
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
	rr := &capRunner{}
	tr := NewTransport(rr, dockerCfg(), config.Docker)
	if err := tr.Upload(context.Background(), config.Primary, []byte("x"), "/tmp/a'b"); err != nil {
		t.Fatalf("Upload: %v", err)
	}
	shArg := rr.last().args[len(rr.last().args)-1]
	if shArg != `cat > '/tmp/a'\''b'` {
		t.Errorf("dest quoting = %q", shArg)
	}
}

// --- `<runtime> cp`, both directions, container-name-prefixed ----------------

func TestTransportCopy(t *testing.T) {
	rr := &capRunner{}
	tr := NewTransport(rr, dockerCfg(), config.Docker)
	ctx := context.Background()

	if err := tr.UploadFile(ctx, config.Primary, "local/in.cli", "/usr/sw/jail/cliscripts/.x.cli"); err != nil {
		t.Fatalf("UploadFile: %v", err)
	}
	up := rr.last()
	wantUp := []string{"cp", "local/in.cli", "solace:/usr/sw/jail/cliscripts/.x.cli"}
	if up.method != "Run" || up.name != "docker" || !eqArgs(up.args, wantUp) {
		t.Errorf("UploadFile argv\n got: %+v\nwant cp %v", up, wantUp)
	}

	if err := tr.Download(ctx, config.Monitor, "/usr/sw/jail/logs/diag.tgz", "out/diag.tgz"); err != nil {
		t.Fatalf("Download: %v", err)
	}
	dn := rr.last()
	wantDn := []string{"cp", "solace:/usr/sw/jail/logs/diag.tgz", "out/diag.tgz"}
	if dn.method != "Run" || dn.name != "docker" || !eqArgs(dn.args, wantDn) {
		t.Errorf("Download argv\n got: %+v\nwant cp %v", dn, wantDn)
	}
}

// --- end-to-end: a real broker.Ops over engine.Echo never echoes the body ----

func TestTransportEchoHidesUploadBody(t *testing.T) {
	buf := &bytes.Buffer{}
	cfg := podmanCfg()
	tr := NewTransport(engine.Echo{W: buf}, cfg, config.Podman)
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
	// The CLI exec is echoed as a normal command against the container (no `--`).
	if !strings.Contains(out, "sol-pod "+broker.CLIBinary+" -Apes .probe.cli") {
		t.Errorf("Echo missing the cli exec line:\n%s", out)
	}
}

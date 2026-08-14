package engine

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
)

// TestHelperProcess is not a real test: it is re-invoked as the child process
// by the Exec tests via the standard os/exec helper-process pattern. It only
// runs when GO_WANT_HELPER_PROCESS=1, so under a normal `go test` run it is a
// no-op. Everything after the "--" separator in os.Args is the fake command.
func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	// Find the "--" separator and take the args after it.
	args := os.Args
	for i, a := range args {
		if a == "--" {
			args = args[i+1:]
			break
		}
	}
	if len(args) == 0 {
		os.Exit(0)
	}
	switch args[0] {
	case "echo":
		fmt.Fprint(os.Stdout, strings.Join(args[1:], " "))
	case "cat":
		_, _ = io.Copy(os.Stdout, os.Stdin)
	case "env":
		// Prints the value of the variable named by the next arg, which is how the
		// RunEnv test proves the child actually received it.
		if len(args) > 1 {
			fmt.Fprint(os.Stdout, os.Getenv(args[1]))
		}
	case "fail":
		os.Exit(3)
	default:
		os.Exit(0)
	}
	os.Exit(0)
}

// helperCommand builds the re-invocation of the test binary as a fake external
// command. Exec inherits os.Environ(), so callers set GO_WANT_HELPER_PROCESS
// via t.Setenv before running.
func helperCommand(extra ...string) (string, []string) {
	name := os.Args[0]
	args := append([]string{"-test.run=TestHelperProcess", "--"}, extra...)
	return name, args
}

// captureStdout swaps os.Stdout for a pipe while fn runs and returns whatever
// fn (or a child process streaming to os.Stdout) wrote. os.Stdout is always
// restored. Not safe with t.Parallel.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	os.Stdout = w
	defer func() { os.Stdout = old }()

	done := make(chan []byte, 1)
	errc := make(chan error, 1)
	go func() {
		b, e := io.ReadAll(r)
		errc <- e
		done <- b
	}()

	fn()

	os.Stdout = old
	if err := w.Close(); err != nil {
		t.Fatalf("close pipe writer: %v", err)
	}
	if e := <-errc; e != nil {
		t.Fatalf("read pipe: %v", e)
	}
	out := <-done
	if err := r.Close(); err != nil {
		t.Fatalf("close pipe reader: %v", err)
	}
	return string(out)
}

func TestQuoteTok(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"empty", "", "''"},
		{"plain", "kubectl", "kubectl"},
		{"plainDashes", "--dry-run=client", "--dry-run=client"},
		{"spaces", "hello world", "'hello world'"},
		{"tab", "a\tb", "'a\tb'"},
		{"newline", "a\nb", "'a\nb'"},
		{"singleQuote", "it's", `'it'\''s'`},
		{"dollar", "$HOME", "'$HOME'"},
		{"pipe", "a|b", "'a|b'"},
		{"semicolon", "a;b", "'a;b'"},
		{"redirect", "a>b", "'a>b'"},
		{"glob", "*.yaml", "'*.yaml'"},
		{"brackets", "a[0]", "'a[0]'"},
		{"braces", "a{b}", "'a{b}'"},
		{"parens", "a(b)", "'a(b)'"},
		{"doubleQuote", `a"b`, `'a"b'`},
		{"ampersand", "a&b", "'a&b'"},
		{"question", "a?b", "'a?b'"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := quoteTok(tt.in); got != tt.want {
				t.Errorf("quoteTok(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestQuote(t *testing.T) {
	tests := []struct {
		name string
		cmd  string
		args []string
		want string
	}{
		{"nameOnly", "kubectl", nil, "kubectl"},
		{"nameAndArgs", "kubectl", []string{"get", "pods"}, "kubectl get pods"},
		{"quotedArg", "echo", []string{"hello world"}, "echo 'hello world'"},
		{"emptyArg", "echo", []string{""}, "echo ''"},
		{"quoteInArg", "echo", []string{"it's"}, `echo 'it'\''s'`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Quote(tt.cmd, tt.args...); got != tt.want {
				t.Errorf("Quote(%q, %q) = %q, want %q", tt.cmd, tt.args, got, tt.want)
			}
		})
	}
}

func TestEchoRun(t *testing.T) {
	var buf bytes.Buffer
	e := Echo{W: &buf}
	if err := e.Run(context.Background(), "kubectl", "get", "pods"); err != nil {
		t.Fatalf("Run: unexpected err %v", err)
	}
	want := "+ kubectl get pods\n"
	if got := buf.String(); got != want {
		t.Errorf("Run wrote %q, want %q", got, want)
	}
}

func TestEchoRunInteractive(t *testing.T) {
	var buf bytes.Buffer
	e := Echo{W: &buf}
	if err := e.RunInteractive(context.Background(), "bash", "-c", "echo hi"); err != nil {
		t.Fatalf("RunInteractive: unexpected err %v", err)
	}
	want := "+ bash -c 'echo hi'\n"
	if got := buf.String(); got != want {
		t.Errorf("RunInteractive wrote %q, want %q", got, want)
	}
}

func TestEchoRunInput(t *testing.T) {
	var buf bytes.Buffer
	e := Echo{W: &buf}
	in := []byte("apiVersion: v1")
	if err := e.RunInput(context.Background(), in, "kubectl", "apply", "-f", "-"); err != nil {
		t.Fatalf("RunInput: unexpected err %v", err)
	}
	want := fmt.Sprintf("+ kubectl apply -f -  <<< (%d bytes on stdin)\n", len(in))
	if got := buf.String(); got != want {
		t.Errorf("RunInput wrote %q, want %q", got, want)
	}
}

// TestEchoRunEnv is the dry-run half of the secret-carrying path: the variable
// names must be visible (they are what an operator checks) and no value may be.
// The command still leads the line, so "+ <cmd>" stays greppable.
func TestEchoRunEnv(t *testing.T) {
	var buf bytes.Buffer
	e := Echo{W: &buf}
	env := []string{"SOLACE_ADMIN_PASSWORD=hunter2", "SOLACE_REDUNDANCY_PSK=psk-value"}
	if err := e.RunEnv(context.Background(), env, "docker", "compose", "up", "-d"); err != nil {
		t.Fatalf("RunEnv: unexpected err %v", err)
	}
	want := "+ docker compose up -d  <<< (env: SOLACE_ADMIN_PASSWORD=*** SOLACE_REDUNDANCY_PSK=***)\n"
	if got := buf.String(); got != want {
		t.Errorf("RunEnv wrote %q, want %q", got, want)
	}
}

// TestEchoRunEnvNoEnv covers the empty-environment arm: with nothing to annotate
// the line is exactly what Run would print, rather than a dangling "(env: )".
func TestEchoRunEnvNoEnv(t *testing.T) {
	var buf bytes.Buffer
	e := Echo{W: &buf}
	if err := e.RunEnv(context.Background(), nil, "docker", "compose", "ps"); err != nil {
		t.Fatalf("RunEnv: unexpected err %v", err)
	}
	want := "+ docker compose ps\n"
	if got := buf.String(); got != want {
		t.Errorf("RunEnv wrote %q, want %q", got, want)
	}
}

func TestMaskEnv(t *testing.T) {
	tests := []struct {
		name string
		in   []string
		want string
	}{
		{"none", nil, ""},
		{"one", []string{"A=b"}, "A=***"},
		{"two", []string{"A=b", "C=d"}, "A=*** C=***"},
		{"valueWithEquals", []string{"A=b=c"}, "A=***"},
		{"emptyValue", []string{"A="}, "A=***"},
		{"oddName", []string{"A B=c"}, "'A B'=***"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MaskEnv(tt.in); got != tt.want {
				t.Errorf("MaskEnv(%v) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestEchoOutput(t *testing.T) {
	var buf bytes.Buffer
	e := Echo{W: &buf}
	out, err := e.Output(context.Background(), "helm", "version")
	if err != nil {
		t.Fatalf("Output: unexpected err %v", err)
	}
	if out != nil {
		t.Errorf("Output returned %v, want nil", out)
	}
	want := "+ helm version\n"
	if got := buf.String(); got != want {
		t.Errorf("Output wrote %q, want %q", got, want)
	}
}

// TestEchoDefaultWriter exercises Echo.w()'s default branch: a zero-value Echo
// (nil W) writes to os.Stdout.
func TestEchoDefaultWriter(t *testing.T) {
	got := captureStdout(t, func() {
		if err := (Echo{}).Run(context.Background(), "kubectl", "get", "pods"); err != nil {
			t.Errorf("Run: unexpected err %v", err)
		}
	})
	want := "+ kubectl get pods\n"
	if got != want {
		t.Errorf("Echo{}.Run wrote %q, want %q", got, want)
	}
}

func TestExecOutput(t *testing.T) {
	t.Setenv("GO_WANT_HELPER_PROCESS", "1")

	name, args := helperCommand("echo", "hello")
	out, err := Exec{}.Output(context.Background(), name, args...)
	if err != nil {
		t.Fatalf("Output(echo): unexpected err %v", err)
	}
	if !strings.Contains(string(out), "hello") {
		t.Errorf("Output(echo) = %q, want it to contain %q", out, "hello")
	}
}

func TestExecOutputFail(t *testing.T) {
	t.Setenv("GO_WANT_HELPER_PROCESS", "1")

	name, args := helperCommand("fail")
	out, err := Exec{}.Output(context.Background(), name, args...)
	if err == nil {
		t.Fatalf("Output(fail): expected err, got nil (out=%q)", out)
	}
	if !strings.Contains(err.Error(), name) {
		t.Errorf("Output(fail) err = %q, want it to contain binary name %q", err, name)
	}
}

func TestExecRun(t *testing.T) {
	t.Setenv("GO_WANT_HELPER_PROCESS", "1")

	// Success path: default case in the helper exits 0.
	nameOK, argsOK := helperCommand("echo", "streamed")
	var errOK error
	_ = captureStdout(t, func() {
		errOK = Exec{}.Run(context.Background(), nameOK, argsOK...)
	})
	if errOK != nil {
		t.Fatalf("Run(echo): unexpected err %v", errOK)
	}

	// Failure path: helper exits 3.
	nameFail, argsFail := helperCommand("fail")
	var errFail error
	_ = captureStdout(t, func() {
		errFail = Exec{}.Run(context.Background(), nameFail, argsFail...)
	})
	if errFail == nil {
		t.Fatalf("Run(fail): expected err, got nil")
	}
	if !strings.Contains(errFail.Error(), nameFail) {
		t.Errorf("Run(fail) err = %q, want it to contain binary name %q", errFail, nameFail)
	}
}

func TestExecRunInput(t *testing.T) {
	t.Setenv("GO_WANT_HELPER_PROCESS", "1")

	name, args := helperCommand("cat")
	in := []byte("meow")
	var err error
	got := captureStdout(t, func() {
		err = Exec{}.RunInput(context.Background(), in, name, args...)
	})
	if err != nil {
		t.Fatalf("RunInput(cat): unexpected err %v", err)
	}
	if !strings.Contains(got, "meow") {
		t.Errorf("RunInput(cat) streamed %q, want it to contain %q", got, "meow")
	}

	// Failure path: proves RunInput wraps a child failure the same way
	// Run/RunEnv/RunInteractive do (name: err), not just on success.
	nameFail, argsFail := helperCommand("fail")
	var errFail error
	_ = captureStdout(t, func() {
		errFail = Exec{}.RunInput(context.Background(), []byte("meow"), nameFail, argsFail...)
	})
	if errFail == nil {
		t.Fatal("RunInput(fail): expected err, got nil")
	}
	if !strings.Contains(errFail.Error(), nameFail) {
		t.Errorf("RunInput(fail) err = %q, want it to contain binary name %q", errFail, nameFail)
	}
}

// TestExecRunEnv proves both halves of RunEnv's contract: the child receives the
// extra variable, and it still inherits this process's environment (the helper
// only runs at all because GO_WANT_HELPER_PROCESS was inherited).
func TestExecRunEnv(t *testing.T) {
	t.Setenv("GO_WANT_HELPER_PROCESS", "1")

	name, args := helperCommand("env", "SOLACE_TEST_SECRET")
	var err error
	got := captureStdout(t, func() {
		err = Exec{}.RunEnv(context.Background(), []string{"SOLACE_TEST_SECRET=from-parent"}, name, args...)
	})
	if err != nil {
		t.Fatalf("RunEnv: unexpected err %v", err)
	}
	if !strings.Contains(got, "from-parent") {
		t.Errorf("RunEnv child printed %q, want it to contain the injected value", got)
	}

	nameFail, argsFail := helperCommand("fail")
	var errFail error
	_ = captureStdout(t, func() {
		errFail = Exec{}.RunEnv(context.Background(), []string{"A=b"}, nameFail, argsFail...)
	})
	if errFail == nil {
		t.Fatal("RunEnv(fail): expected err, got nil")
	}
	if !strings.Contains(errFail.Error(), nameFail) {
		t.Errorf("RunEnv(fail) err = %q, want it to name the binary %q", errFail, nameFail)
	}
}

func TestExecRunInteractive(t *testing.T) {
	t.Setenv("GO_WANT_HELPER_PROCESS", "1")

	name, args := helperCommand() // no extra args -> helper exits 0
	var err error
	_ = captureStdout(t, func() {
		err = Exec{}.RunInteractive(context.Background(), name, args...)
	})
	if err != nil {
		t.Fatalf("RunInteractive: unexpected err %v", err)
	}

	// Failure path: proves RunInteractive wraps a child failure the same way
	// Run/RunInput/RunEnv do (name: err); this backs `exec -it` / shell
	// sessions, so a silent-failure regression here would hide a broken shell.
	nameFail, argsFail := helperCommand("fail")
	var errFail error
	_ = captureStdout(t, func() {
		errFail = Exec{}.RunInteractive(context.Background(), nameFail, argsFail...)
	})
	if errFail == nil {
		t.Fatal("RunInteractive(fail): expected err, got nil")
	}
	if !strings.Contains(errFail.Error(), nameFail) {
		t.Errorf("RunInteractive(fail) err = %q, want it to contain binary name %q", errFail, nameFail)
	}
}

// TestExecOutputInput proves OutputInput wires stdin from `in` into the child
// and captures stdout into the returned buffer -- not streamed to the real
// stdout -- since this is the curl -K - path with no other way to get the
// response body back to the caller.
func TestExecOutputInput(t *testing.T) {
	t.Setenv("GO_WANT_HELPER_PROCESS", "1")

	name, args := helperCommand("cat")
	in := []byte("secret-body")
	out, err := Exec{}.OutputInput(context.Background(), in, name, args...)
	if err != nil {
		t.Fatalf("OutputInput(cat): unexpected err %v", err)
	}
	if !strings.Contains(string(out), "secret-body") {
		t.Errorf("OutputInput(cat) = %q, want it to contain %q", out, "secret-body")
	}
}

// TestExecOutputInputFail mirrors TestExecOutputFail: OutputInput must wrap a
// child failure the same way its stdin-less sibling Output does.
func TestExecOutputInputFail(t *testing.T) {
	t.Setenv("GO_WANT_HELPER_PROCESS", "1")

	name, args := helperCommand("fail")
	out, err := Exec{}.OutputInput(context.Background(), []byte("secret-body"), name, args...)
	if err == nil {
		t.Fatalf("OutputInput(fail): expected err, got nil (out=%q)", out)
	}
	if !strings.Contains(err.Error(), name) {
		t.Errorf("OutputInput(fail) err = %q, want it to contain binary name %q", err, name)
	}
}

// TestEchoOutputInput is the dry-run half of OutputInput, the same
// credential-bearing call RunInput backs (curl -K - with a password on
// stdin): the byte count must be visible but the stdin body must never reach
// dry-run output, which is printed, logged and pasted into tickets (S3).
func TestEchoOutputInput(t *testing.T) {
	var buf bytes.Buffer
	e := Echo{W: &buf}
	in := []byte("password=hunter2")
	out, err := e.OutputInput(context.Background(), in, "curl", "-K", "-")
	if err != nil {
		t.Fatalf("OutputInput: unexpected err %v", err)
	}
	if out != nil {
		t.Errorf("OutputInput returned %v, want nil", out)
	}
	want := fmt.Sprintf("+ curl -K -  <<< (%d bytes on stdin)\n", len(in))
	if got := buf.String(); got != want {
		t.Errorf("OutputInput wrote %q, want %q", got, want)
	}
	if strings.Contains(buf.String(), "hunter2") {
		t.Errorf("OutputInput leaked stdin body into dry-run output: %q", buf.String())
	}
}

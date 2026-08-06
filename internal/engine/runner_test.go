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
}

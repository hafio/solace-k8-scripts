// Package engine abstracts external command execution (kubectl, helm, docker,
// podman, systemctl, openssl) so the rest of the tool can execute commands,
// echo them (--dry-run), or stub them in tests without touching os/exec.
package engine

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

// Runner runs external commands. Implementations: Exec (real), Echo (--dry-run),
// and test fakes. Every path the tool shells out through goes via a Runner so
// behaviour is uniform and testable.
type Runner interface {
	// Run executes name+args, streaming stdout/stderr to this process's stdout/stderr.
	Run(ctx context.Context, name string, args ...string) error
	// RunInput is Run with stdin fed from in (used for `kubectl apply -f -`).
	RunInput(ctx context.Context, in []byte, name string, args ...string) error
	// RunInteractive wires this process's stdio through to the child, for
	// interactive sessions (`exec -it`, a Solace CLI, a shell).
	RunInteractive(ctx context.Context, name string, args ...string) error
	// Output executes name+args and returns captured stdout; stderr is streamed.
	Output(ctx context.Context, name string, args ...string) ([]byte, error)
	// OutputInput is Output with stdin fed from in: it returns captured stdout
	// while streaming stderr (used for `curl -K -` with credentials on stdin).
	OutputInput(ctx context.Context, in []byte, name string, args ...string) ([]byte, error)
}

// Exec is the real Runner backed by os/exec.
type Exec struct{}

func (Exec) Run(ctx context.Context, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	return nil
}

func (Exec) RunInput(ctx context.Context, in []byte, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdin = bytes.NewReader(in)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	return nil
}

func (Exec) RunInteractive(ctx context.Context, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	return nil
}

func (Exec) Output(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return out.Bytes(), fmt.Errorf("%s: %w", name, err)
	}
	return out.Bytes(), nil
}

func (Exec) OutputInput(ctx context.Context, in []byte, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdin = bytes.NewReader(in)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return out.Bytes(), fmt.Errorf("%s: %w", name, err)
	}
	return out.Bytes(), nil
}

// Echo prints the command it would run instead of running it (--dry-run).
// Output returns nothing so callers that parse output degrade gracefully.
type Echo struct{ W io.Writer }

func (e Echo) w() io.Writer {
	if e.W != nil {
		return e.W
	}
	return os.Stdout
}

func (e Echo) Run(_ context.Context, name string, args ...string) error {
	fmt.Fprintln(e.w(), "+ "+Quote(name, args...))
	return nil
}

func (e Echo) RunInput(_ context.Context, in []byte, name string, args ...string) error {
	fmt.Fprintf(e.w(), "+ %s  <<< (%d bytes on stdin)\n", Quote(name, args...), len(in))
	return nil
}

func (e Echo) RunInteractive(_ context.Context, name string, args ...string) error {
	fmt.Fprintln(e.w(), "+ "+Quote(name, args...))
	return nil
}

func (e Echo) Output(_ context.Context, name string, args ...string) ([]byte, error) {
	fmt.Fprintln(e.w(), "+ "+Quote(name, args...))
	return nil, nil
}

func (e Echo) OutputInput(_ context.Context, in []byte, name string, args ...string) ([]byte, error) {
	fmt.Fprintf(e.w(), "+ %s  <<< (%d bytes on stdin)\n", Quote(name, args...), len(in))
	return nil, nil
}

// Quote renders a command line for display, single-quoting any token that
// contains whitespace or shell-significant characters. Display only.
func Quote(name string, args ...string) string {
	var b strings.Builder
	b.WriteString(quoteTok(name))
	for _, a := range args {
		b.WriteByte(' ')
		b.WriteString(quoteTok(a))
	}
	return b.String()
}

func quoteTok(s string) string {
	if s == "" {
		return "''"
	}
	if strings.ContainsAny(s, " \t\n\"'$&|;<>()*?[]{}") {
		return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
	}
	return s
}

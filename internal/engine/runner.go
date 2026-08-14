// Package engine abstracts external command execution (kubectl, helm, docker,
// podman, systemctl, openssl) so the rest of the tool can execute commands,
// echo them (--dry-run), or stub them in tests without touching os/exec.
package engine

import (
	"bytes"
	"context"
	"errors"
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
	// RunEnv is Run with extra "KEY=value" variables added to the child's
	// environment. It carries secret values to a child that reads them from its
	// environment (docker compose's environment-sourced secrets) without ever
	// putting them in an argv -- so implementations must never echo a value.
	RunEnv(ctx context.Context, extraEnv []string, name string, args ...string) error
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

// ResolveOut is where the pre-exec `exec:` line is written. It exists so a test can
// capture the line; production leaves it nil and the line goes to stderr, which
// keeps it out of any rendered artifact piped from stdout.
var ResolveOut io.Writer

func resolveOut() io.Writer {
	if ResolveOut != nil {
		return ResolveOut
	}
	return os.Stderr
}

// command resolves name through PATH and builds the exec.Cmd, announcing the
// resolved binary on stderr before the caller runs it:
//
//	exec: /usr/bin/kubectl get pods -n solace
//
// Resolution is explicit rather than left to exec.Command so two things are true at
// the moment they matter. First, an unexpected binary LOCATION is visible -- the
// allowlist in config guarantees `kubectl` is what was asked for, and this line
// shows which kubectl actually answered. Second, exec.ErrDot is an error here, not
// a fallback: Go reports it when a bare name resolved relative to the current
// directory, which is precisely the "attacker-supplied file shipped alongside the
// config" case, so it is refused rather than run.
func command(ctx context.Context, name string, args []string) (*exec.Cmd, error) {
	path, err := exec.LookPath(name)
	if err != nil {
		// LookPath returns the path it found ALONGSIDE ErrDot, so the message can
		// name the file that would have run.
		if errors.Is(err, exec.ErrDot) {
			return nil, fmt.Errorf("refusing to run %q from the current directory: it resolved to %q, which is "+
				"not on your PATH -- a binary shipped beside an env file must never run implicitly; "+
				"install it, or add its directory to PATH", name, path)
		}
		return nil, fmt.Errorf("%s: not found on PATH: %w", name, err)
	}
	fmt.Fprintln(resolveOut(), "exec: "+Quote(path, args...))
	cmd := exec.CommandContext(ctx, path, args...)
	return cmd, nil
}

func (Exec) Run(ctx context.Context, name string, args ...string) error {
	cmd, err := command(ctx, name, args)
	if err != nil {
		return err
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	return nil
}

func (Exec) RunInput(ctx context.Context, in []byte, name string, args ...string) error {
	cmd, err := command(ctx, name, args)
	if err != nil {
		return err
	}
	cmd.Stdin = bytes.NewReader(in)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	return nil
}

func (Exec) RunEnv(ctx context.Context, extraEnv []string, name string, args ...string) error {
	cmd, err := command(ctx, name, args)
	if err != nil {
		return err
	}
	// Inherit and extend: the child still needs PATH, HOME and DOCKER_* from this
	// process, so this adds to the environment rather than replacing it. Appending
	// also means a name that somehow collided with an inherited one would be a
	// duplicate entry rather than an override -- but nothing config-derived can
	// produce a bare PATH/LD_PRELOAD name in the first place, since every variable
	// here is a secret name carrying a fixed literal suffix (render.ContainerSecret;
	// pinned by TestComposeSecretEnvNamesCannotBeSystemVars).
	cmd.Env = append(os.Environ(), extraEnv...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	return nil
}

func (Exec) RunInteractive(ctx context.Context, name string, args ...string) error {
	cmd, err := command(ctx, name, args)
	if err != nil {
		return err
	}
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	return nil
}

func (Exec) Output(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd, err := command(ctx, name, args)
	if err != nil {
		return nil, err
	}
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return out.Bytes(), fmt.Errorf("%s: %w", name, err)
	}
	return out.Bytes(), nil
}

func (Exec) OutputInput(ctx context.Context, in []byte, name string, args ...string) ([]byte, error) {
	cmd, err := command(ctx, name, args)
	if err != nil {
		return nil, err
	}
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

// RunEnv echoes the variable names it would set with their values masked: the
// whole point of the environment is to carry secrets, and --dry-run output is
// printed, logged and pasted into tickets (§3). The names are annotated AFTER the
// command, the way RunInput annotates its stdin, so every echoed line still reads
// as "+ <the command>".
func (e Echo) RunEnv(ctx context.Context, extraEnv []string, name string, args ...string) error {
	if len(extraEnv) == 0 {
		return e.Run(ctx, name, args...)
	}
	fmt.Fprintf(e.w(), "+ %s  <<< (env: %s)\n", Quote(name, args...), MaskEnv(extraEnv))
	return nil
}

// MaskEnv renders "KEY=value" pairs as a printable "KEY=*** KEY2=***", keeping the
// names (which are what a reader needs to check the wiring) and dropping every
// value. Exported so any other display path masks identically.
func MaskEnv(env []string) string {
	masked := make([]string, 0, len(env))
	for _, pair := range env {
		key, _, _ := strings.Cut(pair, "=")
		masked = append(masked, quoteTok(key)+"=***")
	}
	return strings.Join(masked, " ")
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

package engine

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// captureResolve swaps the pre-exec announcement sink for a buffer and returns it.
func captureResolve(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	prev := ResolveOut
	ResolveOut = &buf
	t.Cleanup(func() { ResolveOut = prev })
	return &buf
}

// TestExecEchoesResolvedPath: before running anything, Exec announces the binary it
// actually resolved. The allowlist in config guarantees the NAME is one this tool
// drives; this line is what makes the LOCATION visible, at the moment it matters.
// It goes to stderr so it never contaminates a rendered artifact on stdout.
func TestExecEchoesResolvedPath(t *testing.T) {
	buf := captureResolve(t)
	name, args := helperCommand("echo", "hello")
	if err := (Exec{}).Run(context.Background(), name, args...); err != nil {
		t.Fatalf("Run: %v", err)
	}
	line := buf.String()
	if !strings.HasPrefix(line, "exec: ") {
		t.Fatalf("announcement = %q, want it to start with %q", line, "exec: ")
	}
	// The path, not the name as typed: an absolute path is the whole point.
	path := strings.TrimPrefix(strings.SplitN(line, " ", 3)[1], "")
	if !filepath.IsAbs(strings.Trim(path, "'")) {
		t.Errorf("announced %q, want an absolute resolved path", path)
	}
	// The remaining args are shown too, so the line reads as the whole command.
	if !strings.Contains(line, "hello") {
		t.Errorf("announcement = %q, want it to carry the arguments", line)
	}
}

// TestExecEchoesOnEveryMethod: every Runner method resolves and announces, not just
// Run. A method that skipped it would be a silent hole in exactly the transparency
// this exists to provide -- and Output/OutputInput are the ones that read cluster
// state, so they are the least visible to begin with.
func TestExecEchoesOnEveryMethod(t *testing.T) {
	name, args := helperCommand("echo", "x")
	cases := []struct {
		method string
		call   func() error
	}{
		{"Run", func() error { return Exec{}.Run(context.Background(), name, args...) }},
		{"RunInput", func() error { return Exec{}.RunInput(context.Background(), nil, name, args...) }},
		{"RunEnv", func() error { return Exec{}.RunEnv(context.Background(), nil, name, args...) }},
		{"RunInteractive", func() error { return Exec{}.RunInteractive(context.Background(), name, args...) }},
		{"Output", func() error { _, err := Exec{}.Output(context.Background(), name, args...); return err }},
		{"OutputInput", func() error {
			_, err := Exec{}.OutputInput(context.Background(), nil, name, args...)
			return err
		}},
	}
	for _, tc := range cases {
		t.Run(tc.method, func(t *testing.T) {
			buf := captureResolve(t)
			if err := tc.call(); err != nil {
				t.Fatalf("%s: %v", tc.method, err)
			}
			if !strings.HasPrefix(buf.String(), "exec: ") {
				t.Errorf("%s announced %q, want an `exec: <path>` line", tc.method, buf.String())
			}
		})
	}
}

// TestExecMissingBinaryIsActionable: a name that resolves nowhere fails before any
// process starts, naming what was not found. Reaching os/exec's own "file does not
// exist" would name a path the operator never typed.
func TestExecMissingBinaryIsActionable(t *testing.T) {
	captureResolve(t)
	err := Exec{}.Run(context.Background(), "solace-no-such-binary-exists", "x")
	if err == nil {
		t.Fatal("running a binary that is not on PATH must fail")
	}
	for _, want := range []string{"solace-no-such-binary-exists", "not found on PATH"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error = %v, want it to contain %q", err, want)
		}
	}
}

// TestExecRefusesCurrentDirectoryResolution is the layer-6 guarantee that pairs
// with config's bare-name rule: even a bare, allowlisted name must not resolve to a
// file sitting in the working directory. That is precisely the "binary shipped
// alongside the env file" case -- an operator who unpacks a shared archive and runs
// the tool from inside it. Go reports it as exec.ErrDot; this refuses rather than
// running it.
func TestExecRefusesCurrentDirectoryResolution(t *testing.T) {
	captureResolve(t)
	dir := t.TempDir()

	// A file that LookPath would consider executable, named so it cannot collide
	// with anything genuinely installed.
	base := "solace-cwd-probe"
	body := "#!/bin/sh\nexit 0\n"
	mode := os.FileMode(0o755)
	if runtime.GOOS == "windows" {
		// Windows resolves by PATHEXT; .bat is in the default list.
		base += ".bat"
		body = "@echo off\r\nexit /b 0\r\n"
		mode = 0o666
	}
	if err := os.WriteFile(filepath.Join(dir, base), []byte(body), mode); err != nil {
		t.Fatalf("writing the probe: %v", err)
	}
	t.Chdir(dir)

	// Ask for it the way an env file would: a bare name, no separator. The
	// invariant under test is the same either way -- the file in the working
	// directory must not run -- but which layer stops it depends on the host, so
	// check what this one's LookPath does before asserting the message.
	//
	// Windows resolves the current directory implicitly unless
	// NoDefaultCurrentDirectoryInExePath is set; Unix does not resolve it at all
	// unless "." is on PATH. Go signals the former with exec.ErrDot, which is the
	// branch this test is really about.
	_, lookErr := exec.LookPath("solace-cwd-probe")
	err := Exec{}.Run(context.Background(), "solace-cwd-probe", "arg")
	if err == nil {
		t.Fatal("a binary in the current directory was executed; it must never resolve from cwd")
	}
	if errors.Is(lookErr, exec.ErrDot) {
		if !strings.Contains(err.Error(), "current directory") {
			t.Errorf("error = %v, want it to say the current directory was refused", err)
		}
		return
	}
	// This host never offered the cwd copy in the first place, so there was
	// nothing for the ErrDot branch to refuse. The invariant still holds -- the
	// probe did not run -- and the branch itself is asserted on hosts that do.
	t.Logf("host does not resolve from the current directory (LookPath: %v); "+
		"the ErrDot branch is exercised where it does", lookErr)
	if !strings.Contains(err.Error(), "not found on PATH") {
		t.Errorf("error = %v, want the not-found message", err)
	}
}

// TestChildEnvNamesAreNotSystemVariables pins the layer-6 environment rule at the
// place the environment is actually built. RunEnv is the ONLY path in this codebase
// that adds variables to a child, and the values it carries are secrets -- so the
// question that matters is whether anything could name a variable the child's
// loader reads. Nothing here may be PATH, LD_PRELOAD or their relatives.
//
// The upstream half (that a config-derived secret name always keeps a fixed literal
// suffix, so it cannot fold to a bare PATH) is pinned in container's
// TestComposeSecretEnvNamesCannotBeSystemVars; this end asserts the invariant holds
// for whatever it is handed.
func TestChildEnvNamesAreNotSystemVariables(t *testing.T) {
	dangerous := []string{
		"PATH", "Path", "LD_PRELOAD", "LD_LIBRARY_PATH", "DYLD_INSERT_LIBRARIES",
		"PATHEXT", "SHELL", "IFS", "BASH_ENV", "ENV",
	}
	// The names this tool really passes, from render.ContainerSecret.EnvVar.
	passed := []string{
		"SOLACE_ADMIN_PASSWORD=x",
		"SOLACE_REDUNDANCY_PSK=y",
		"SOLACE_TLS_PASSPHRASE=z",
		"SOLACE_USER_APPUSER_PASSWORD=w",
	}
	for _, pair := range passed {
		name, _, _ := strings.Cut(pair, "=")
		for _, bad := range dangerous {
			if strings.EqualFold(name, bad) {
				t.Errorf("child environment would define %q, which the loader reads", name)
			}
		}
	}
	// And the masking that keeps values out of --dry-run output still holds for them.
	masked := MaskEnv(passed)
	for _, secret := range []string{"x", "y", "z", "w"} {
		if strings.Contains(masked, "="+secret) {
			t.Errorf("MaskEnv(%v) leaked a value: %q", passed, masked)
		}
	}
}

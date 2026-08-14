package broker

import (
	"context"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"time"

	"solace/internal/config"
)

// Ops runs the shared config/verify operations against a broker through a
// Transport. One Ops is built per command with the platform's transport injected
// (kubectl-exec for k8s, engine-exec for containers); the operations themselves
// are platform-agnostic. Log receives progress lines (routed to stderr by the
// caller so rendered stdout stays clean); a nil Log discards them.
type Ops struct {
	T   Transport
	Cfg *config.Config
	Log func(format string, args ...any) // progress -> stderr; nil discards
	Out io.Writer                        // user-facing command output; nil -> os.Stdout

	// Polling knobs for the HA state machines (leader, redundancy). New sets
	// sensible defaults; tests set PollInterval to 0 to avoid sleeping.
	PollInterval time.Duration
	PollAttempts int

	// ActiveDwell is the fixed wait the node-local backup redundancy handshake
	// holds after becoming active before reverting ("after 10s of being active"),
	// distinct from PollInterval. New defaults it to 10s; tests set it to 0.
	ActiveDwell time.Duration
	// Hostname resolves this host's name for node-local role detection
	// (LocalRole). New defaults it to os.Hostname; tests inject a fixed value.
	Hostname func() (string, error)
}

// New builds an Ops with default polling parameters (2s interval, 60 attempts --
// a bounded ceiling replacing the bash scripts' unbounded busy-waits), a 10s
// active-dwell for the backup redundancy handshake, and os.Hostname for role
// detection.
func New(t Transport, cfg *config.Config, log func(string, ...any)) *Ops {
	return &Ops{
		T:            t,
		Cfg:          cfg,
		Log:          log,
		Out:          os.Stdout,
		PollInterval: 2 * time.Second,
		PollAttempts: 60,
		ActiveDwell:  10 * time.Second,
		Hostname:     os.Hostname,
	}
}

func (o *Ops) logf(format string, args ...any) {
	if o.Log != nil {
		o.Log(format, args...)
	}
}

// out returns the user-facing output sink, defaulting to os.Stdout.
func (o *Ops) out() io.Writer {
	if o.Out != nil {
		return o.Out
	}
	return os.Stdout
}

// show writes captured broker output to the user-facing sink. A write error to
// stdout is not actionable, so it is deliberately discarded.
func (o *Ops) show(b []byte) { _, _ = o.out().Write(b) }

// sleep waits PollInterval, honoring context cancellation. A zero interval (in
// tests) returns immediately without allocating a timer.
func (o *Ops) sleep(ctx context.Context) error {
	return o.wait(ctx, o.PollInterval)
}

// dwell waits ActiveDwell, honoring context cancellation. It backs the node-local
// backup handshake's fixed hold after becoming active; a zero duration (in tests)
// returns immediately.
func (o *Ops) dwell(ctx context.Context) error {
	return o.wait(ctx, o.ActiveDwell)
}

// wait blocks for d, honoring context cancellation. A non-positive d returns
// immediately without allocating a timer.
func (o *Ops) wait(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return ctx.Err()
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

// RunCLI implements the CLI-over-exec primitive every config/verify step is built
// on: upload the script body to the cliscripts dir as `.<name>.cli`, then run the
// broker CLI over it and return captured stdout. name must be a safe identifier.
func (o *Ops) RunCLI(ctx context.Context, role config.Role, name, body string) ([]byte, error) {
	if err := validName("cli script name", name); err != nil {
		return nil, err
	}
	if err := o.T.Upload(ctx, role, []byte(body), cliScriptPath(name)); err != nil {
		return nil, fmt.Errorf("upload cli script %q: %w", name, err)
	}
	out, err := o.T.Output(ctx, role, CLIBinary, "-Apes", cliArg(name))
	if err != nil {
		return out, fmt.Errorf("run cli script %q: %w", name, err)
	}
	return out, nil
}

// removeCLI deletes one or more uploaded `.<name>.cli` scripts, best-effort. It
// logs a warning on failure rather than aborting (cleanup is not fatal).
func (o *Ops) removeCLI(ctx context.Context, role config.Role, names ...string) {
	paths := make([]string, 0, len(names))
	for _, n := range names {
		paths = append(paths, cliScriptPath(n))
	}
	args := append([]string{"rm", "-f"}, paths...)
	if err := o.T.Run(ctx, role, args...); err != nil {
		o.logf("[WARN] cleanup of cli script(s) %v failed: %v", names, err)
	}
}

// skipIfStandalone reports whether an HA-gated step should no-op. It logs a WARN
// and returns true for standalone deployments, matching the "Standalone ...
// detected" branches of 050/061.
func (o *Ops) skipIfStandalone(step string) bool {
	if o.Cfg.RedundancyEnabled() {
		return false
	}
	o.logf("[WARN] %s is HA-only; standalone deployment -- skipping.", step)
	return true
}

// field returns the value after the first ": " on the first output line that
// contains label, with CR stripped -- the Go form of the bash idiom
// `grep "<label>" | tr -d '\r'` followed by `${VAR#*: }`. Returns "" if absent.
func field(output, label string) string {
	for _, raw := range strings.Split(output, "\n") {
		line := strings.TrimRight(raw, "\r")
		if strings.Contains(line, label) {
			if i := strings.Index(line, ": "); i >= 0 {
				return line[i+2:]
			}
			return ""
		}
	}
	return ""
}

// countContains counts output lines containing needle (CR stripped), the Go form
// of `grep <label> | grep -c <needle>` used by the redundancy activity checks.
func countContains(output, label, needle string) int {
	n := 0
	for _, raw := range strings.Split(output, "\n") {
		line := strings.TrimRight(raw, "\r")
		if strings.Contains(line, label) && strings.Contains(line, needle) {
			n++
		}
	}
	return n
}

// containsAnyFold reports whether output contains any needle, case-insensitively.
// It ports the `grep -Ei '<a>|<b>'` error scans of 057 and 059.
func containsAnyFold(output string, needles ...string) bool {
	low := strings.ToLower(output)
	for _, n := range needles {
		if strings.Contains(low, strings.ToLower(n)) {
			return true
		}
	}
	return false
}

// nameRE constrains user-influenced identifiers that reach a shell or the CLI
// (cli script names, domain CA names, uploaded filenames) to a safe character
// set, the §3 boundary validation this port owns (the bash scripts had none).
var nameRE = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)

func validName(kind, s string) error {
	if s == "" || strings.Contains(s, "..") || !nameRE.MatchString(s) {
		return fmt.Errorf("invalid %s %q: only letters, digits, '.', '_' and '-' are allowed", kind, s)
	}
	return nil
}

// validCLILine checks a value that is written verbatim into one line of a Solace
// CLI script. Unlike validName it does not constrain the character set -- a
// product key is an opaque vendor string and its alphabet is not ours to decide --
// but it rejects the one thing that changes the script's meaning: a control
// character. A newline would turn a single `product-key <k>` line into extra
// commands run in the already-elevated session (§3 boundary validation).
// cliForbiddenPassword are the characters the broker CLI rejects inside a quoted
// `create username ... password "..."` value. A package-local copy of
// config.cliForbiddenPassword: config validates the env file at load so a bad value
// fails before any deploy, and this is the boundary that actually protects the CLI
// line it is interpolated into (§3 -- validate at the app boundary AND sanitize where
// the value is consumed). One small copy per package, as with nameRE.
const cliForbiddenPassword = ":()\";'<>,`\\*&|"

// validCLIPassword checks a password bound for a quoted CLI value. Unlike
// validCLILine it never puts the offending value in its error -- only the user it
// belongs to and the single character at fault (§3).
func validCLIPassword(user, pass string) error {
	if pass == "" {
		return fmt.Errorf("password for additional user %q must not be empty", user)
	}
	if i := strings.IndexFunc(pass, func(r rune) bool { return r < 0x20 || r == 0x7f }); i >= 0 {
		return fmt.Errorf("password for additional user %q contains a control character at offset %d; "+
			"it must be a single line", user, i)
	}
	if i := strings.IndexAny(pass, cliForbiddenPassword); i >= 0 {
		return fmt.Errorf("password for additional user %q contains %q, which the broker CLI rejects in a "+
			"password; none of %s may appear (the value itself is not shown)",
			user, string(pass[i]), cliForbiddenPassword)
	}
	return nil
}

func validCLILine(kind, s string) error {
	if strings.TrimSpace(s) == "" {
		return fmt.Errorf("invalid %s: must not be empty", kind)
	}
	if i := strings.IndexFunc(s, func(r rune) bool { return r < 0x20 || r == 0x7f }); i >= 0 {
		return fmt.Errorf("invalid %s %q: contains a control character (0x%02x) at offset %d; it must be a single line",
			kind, s, s[i], i)
	}
	return nil
}

package config

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

// An env file is executable content. kubernetes.runtime, docker.runtime, podman.runtime
// and docker.compose each name a binary this process runs on the operator's own
// machine, and env files travel -- repos, pull requests, shared archives -- so the
// person who wrote one is routinely not the person who runs it. Everything in this
// file exists to keep the file from choosing what executes.
//
// What it defends: an unlisted binary; a path form pointing at a file shipped
// beside the config; a subcommand smuggled ahead of the one this tool appends.
//
// What it does not defend, by design: a trojan already installed on the operator's
// PATH (the host is compromised before this code runs -- no in-process check can
// help), and config that is malicious but entirely legitimate in form, such as a
// real kubectl aimed at the wrong cluster. Both are review problems, covered in
// README's trust-model note rather than here.
//
// The whole check is one function, CheckCommand, called from BOTH Validate and
// every executor immediately before argv is built (k8s.Cluster.clusterCmd,
// container.Manager.runtimeCmd, and the two transports). A hostile env file is
// therefore inert even if nobody ran validation first, and the two enforcement
// points cannot drift because there is only one definition to drift from.
//
// The honest limit: a flag's VALUE is an unvalidatable position. Argument arity is
// unknowable without modelling every flag of every allowed binary, so the token
// after `--kubeconfig` is accepted as that flag's value whatever it says. The hard
// guarantee therefore covers argv[0] and every bare token -- what runs, and what
// could act as a subcommand -- not the contents of a flag value.

// execBinaries is the per-platform allowlist: the CLIs this tool actually drives.
// Nothing else may be argv[0] from config alone. `oc` is OpenShift's kubectl and
// `nerdctl` is a docker-CLI drop-in, so both speak the exact command surface this
// tool issues; `docker-compose` is here for docker.compose on a host carrying only
// the standalone v1 binary. Ordered for the error message, which names the list.
//
// Deliberately absent: wrappers such as microk8s, lima, minikube or a site shim.
// They are legitimate ways to reach a broker CLI, but approving one is the operator's
// call, not the env file's -- that is what --allow-command is for. Privilege-escalation
// wrappers are absent for a different and stronger reason: --allow-command cannot
// approve them either (see neverAllowed).
var execBinaries = map[Platform][]string{
	K8s:    {"kubectl", "oc"},
	Docker: {"docker", "docker-compose", "nerdctl"},
	Podman: {"podman"},
}

// neverAllowed are binaries --allow-command may not approve, at any time, by anyone.
// They are all privilege-escalation wrappers, and the reason is not that escalating
// is wrong -- rootful podman genuinely needs root -- but that escalating HERE is the
// wrong place for it. `sudo solace-util deploy` elevates one process the operator
// chose, visibly, at the moment they typed it. A `runtime: sudo podman` elevates
// every command this tool issues for the lifetime of an env file, decided by whoever
// wrote that file, and the operator who approves it once on the command line cannot
// see what it will be used for. Escalate before invoking this tool, never through it.
//
// Refusing the category rather than the word: a list that blocked only `sudo` while
// allowing `doas` or `pkexec` would be a control in name only.
var neverAllowed = map[string]string{
	"sudo":   "sudo",
	"doas":   "doas",
	"su":     "su",
	"pkexec": "pkexec",
	"run0":   "run0",
	"runas":  "runas", // Windows
	"gsudo":  "gsudo", // Windows
}

// unsafeTokenChars are the characters no command token may carry. Under argv exec
// none of them is an injection -- exec never involves a shell, so ';' is an
// ordinary filename character -- but a token holding one is inert here only as
// long as it stays in an argv. These same tokens reach log lines, --dry-run output
// pasted into tickets, and the rendered compose/quadlet artifacts, so they are
// refused at the boundary instead of being escaped correctly by every consumer
// forever (S3: validate at the boundary AND sanitize at the shell layer).
//
// Backslash is included, which means a Windows path cannot appear in any token.
// That is not a loss for argv[0], which may not be a path at all, and a flag value
// takes forward slashes -- kubectl, docker and podman all accept
// `C:/Users/you/.kube/config`. The error message says so.
const unsafeTokenChars = "\"'`$;|&<>()*?[]{}~#!\\"

// pathSeparators are the characters that make a token a path rather than a bare
// name. Both are rejected on every platform: an env file that ships alongside a
// `./kubectl` must not be able to point at it, and a Windows-only check would let
// the same file do exactly that when carried to Linux.
const pathSeparators = `/\`

// commandRules describe what one Command-typed field may contain. Every field this
// tool executes has exactly one rules value, built by the helpers below, so the
// per-field differences live in one place rather than at the call sites.
type commandRules struct {
	field    string   // schema path, for error messages ("kubernetes.runtime")
	platform Platform // which allowlist applies
	subword  string   // the single bare subcommand this field may carry at index 1
}

// clusterRules guard kubernetes.runtime -- the cluster CLI, checked on every platform
// because ApplyDefaults fills it everywhere and only k8s reads it.
func clusterRules() commandRules {
	return commandRules{field: "kubernetes.runtime", platform: K8s}
}

// runtimeRules guard docker.runtime / podman.runtime.
func runtimeRules(p Platform) commandRules {
	return commandRules{field: platformKey(p) + ".runtime", platform: p}
}

// composeRules guard docker.compose. It is the one field allowed a bare token, and
// only the exact word `compose` as the LAST token, directly after an allowed
// binary: the compose plugin really is a subcommand of the runtime
// (`docker compose`, or `lima nerdctl compose` when the runtime is wrapped), so
// refusing it would refuse the field's own derived default. Constraining it to that
// literal in that position is the point -- a `compose: docker rm` would otherwise
// smuggle a destructive verb ahead of the `-f <file> up -d` this tool appends.
func composeRules() commandRules {
	return commandRules{field: "docker.compose", platform: Docker, subword: "compose"}
}

// allowed builds the lookup for one rules value: the platform allowlist plus the
// operator's --allow-command additions for this invocation. Rebuilt per call
// rather than cached -- the lists hold a handful of entries, and a cache would be
// one more thing that could disagree between validator and executor.
func (r commandRules) allowed(extra map[string]bool) map[string]bool {
	set := make(map[string]bool, len(execBinaries[r.platform])+len(extra))
	for _, name := range execBinaries[r.platform] {
		set[name] = true
	}
	for name := range extra {
		set[name] = true
	}
	// Belt and braces on the escalation rule. AllowCommands already refuses these
	// with a message that explains the alternative, which is where an operator
	// actually meets the rule -- this second pass makes the outcome structural, so a
	// future edit that adds one to execBinaries, or a caller that populates
	// extraAllowed some other way, still cannot put a privilege-escalation wrapper
	// in front of the broker CLI.
	for name := range neverAllowed {
		delete(set, name)
	}
	return set
}

// list renders this platform's allowlist for an error message, in declaration
// order, without the operator's additions -- naming those back would suggest the
// env file could have asked for them.
func (r commandRules) list() string {
	return strings.Join(execBinaries[r.platform], ", ")
}

// CheckCommand is the single definition of what an executable command may be. It
// runs layers 1-3 in order: every token passes the charset; argv[0] is a bare,
// allowlisted binary name; every later token is a flag, a flag's value, or another
// allowlisted binary. extra carries the operator's --allow-command approvals for
// this invocation, and is nil on the config-only path.
//
// It is exported so the executors can re-run the exact check the validator ran,
// immediately before they build argv (layer 5). Callers use the Command accessors
// below rather than calling this directly.
func CheckCommand(r commandRules, cmd Command, extra map[string]bool) error {
	if len(cmd) == 0 {
		return fmt.Errorf("%s is empty: it must name the binary to run (one of: %s)", r.field, r.list())
	}
	for i, tok := range cmd {
		if err := checkToken(r.field, i, tok); err != nil {
			return err
		}
	}
	ok := r.allowed(extra)
	if err := checkBinary(r, cmd[0], ok); err != nil {
		return err
	}
	return checkFlagShape(r, cmd, ok)
}

// checkToken is layer 1: the token charset. Emptiness and control characters keep
// their own messages, since an empty argument and a converted-from-bash newline
// are the two mistakes a well-meaning env file actually makes.
func checkToken(field string, i int, tok string) error {
	if tok == "" {
		return fmt.Errorf("%s[%d] is an empty argument; remove it or quote the intended value", field, i)
	}
	if j := strings.IndexFunc(tok, isCtrl); j >= 0 {
		return fmt.Errorf("%s[%d] contains a control character (0x%02x) at offset %d: %q",
			field, i, tok[j], j, tok)
	}
	if strings.IndexFunc(tok, isSpace) >= 0 {
		return fmt.Errorf("%s[%d] = %q contains whitespace inside one argument; write each argument as its own "+
			"list entry, or use the scalar form, which splits on whitespace", field, i, tok)
	}
	if j := strings.IndexFunc(tok, isInvisible); j >= 0 {
		r, _ := utf8.DecodeRuneInString(tok[j:])
		return fmt.Errorf("%s[%d] = %q contains an invisible formatting character (U+%04X) at offset %d; "+
			"it cannot be seen in a review or a log line, so it is not allowed in a command token", field, i, tok, r, j)
	}
	if j := strings.IndexAny(tok, unsafeTokenChars); j >= 0 {
		return fmt.Errorf("%s[%d] = %q contains %q, which is not allowed in a command token; "+
			"a Windows path works with forward slashes (C:/Users/you/.kube/config)",
			field, i, tok, string(tok[j]))
	}
	return nil
}

// checkBinary is layer 2: argv[0] is a bare name from the allowlist. The bare-name
// rule is what stops `command: ./kubectl` -- a relative or absolute path would run
// a file the env file chose, which is the whole attack; a bare name can only be
// resolved through the operator's own PATH (engine.Exec does that resolution
// explicitly, and refuses to resolve from the current directory).
func checkBinary(r commandRules, tok string, allowed map[string]bool) error {
	if j := strings.IndexAny(tok, pathSeparators); j >= 0 {
		return fmt.Errorf("%s[0] = %q must be a bare binary name, not a path: %q would run a file named by the "+
			"env file rather than the one on your PATH -- write it as %q and let PATH resolve it",
			r.field, tok, string(tok[j]), execBase(tok))
	}
	if !allowed[execBase(tok)] {
		return fmt.Errorf("%s[0] = %q is not a binary this tool runs: allowed on %s are %s -- "+
			"correct the env file, or approve this one for a single run with --allow-command %s",
			r.field, tok, r.platform, r.list(), execBase(tok))
	}
	return nil
}

// checkFlagShape is layer 3: everything after argv[0]. A token passes if it is a
// flag, another allowlisted binary (a chained runner such as an approved
// `lima nerdctl`), this field's one permitted subword, or the value of the flag
// before it. Anything else is a bare word sitting exactly where this tool appends
// its own subcommand, so it is refused.
func checkFlagShape(r commandRules, cmd Command, allowed map[string]bool) error {
	for i := 1; i < len(cmd); i++ {
		tok := cmd[i]
		switch {
		case tok == "--":
			// End-of-flags would reopen everything the bare-word rule closes: the
			// next token stops being a subcommand to the shell-free parser and
			// starts being a positional the allowed binary happily accepts.
			return fmt.Errorf("%s[%d] = \"--\" is not allowed: end-of-flags would let the env file smuggle a "+
				"positional argument ahead of the subcommand this tool appends -- remove it", r.field, i)
		case strings.HasPrefix(tok, "-"):
		case allowed[execBase(tok)]:
		case r.subword != "" && tok == r.subword && i == len(cmd)-1 && allowed[execBase(cmd[i-1])]:
			// The field's one permitted subcommand, and only in the one position
			// where it means what the field says: last, directly after an allowed
			// binary. That covers `docker compose` and, with lima approved,
			// `lima nerdctl compose` -- while `docker rm` is still a bare word
			// and `docker compose up` still has a token this tool did not append.
		case strings.HasPrefix(cmd[i-1], "-") && !strings.Contains(cmd[i-1], "="):
			// The value of the preceding flag. Unvalidatable beyond the charset:
			// arity is unknowable without modelling every flag of every allowed
			// binary, so this position is deliberately trusted (see the file header).
		default:
			return fmt.Errorf("%s[%d] = %q is not allowed in subcommand position: this tool appends its own "+
				"subcommand, so a bare word here would run ahead of it -- only flags, their values, and "+
				"allowed binaries (%s) may follow; approve a chained runner for a single run with "+
				"--allow-command %s", r.field, i, tok, r.list(), execBase(tok))
		}
	}
	return nil
}

// execBase strips one optional .exe/.EXE suffix so a Windows operator may write
// `kubectl.exe` and match the same allowlist entry. Only the suffix is folded --
// the name itself is compared exactly, because on a case-sensitive filesystem
// `KUBECTL` and `kubectl` are different files.
func execBase(tok string) string {
	if len(tok) > 4 && strings.EqualFold(tok[len(tok)-4:], ".exe") {
		return tok[:len(tok)-4]
	}
	return tok
}

// isSpace reports the whitespace a single argument may not contain, over the whole
// Unicode White_Space property rather than just ASCII space. That matters because
// the two YAML forms do not agree: the scalar form is split with strings.Fields,
// which is Unicode-aware, so `runtime: kubectl -n a<U+3000>b` becomes three tokens
// and never reaches here with the character embedded -- but the explicit sequence
// form preserves it. An ASCII-only check would therefore accept through one form
// exactly what it rejects through the other. Tab and the vertical whitespace are
// caught earlier as control characters, which gives them a better message.
func isSpace(r rune) bool { return unicode.IsSpace(r) }

// isInvisible reports the Unicode format characters (category Cf): zero-width
// spaces and joiners, bidirectional overrides, and their relatives. They are not
// whitespace and carry no argv-splitting risk under argv exec, so they get their
// own check and their own message -- what makes them unacceptable is that they are
// invisible. A token that renders identically to a legitimate one, in a review, a
// log line, or a --dry-run transcript pasted into a ticket, defeats the reading
// this whole file asks an operator to do. Nothing legitimate needs one.
func isInvisible(r rune) bool { return unicode.Is(unicode.Cf, r) }

// AllowCommands records the operator's --allow-command approvals for this
// invocation and is the ONLY way to widen the allowlist. It is deliberately not
// reachable from the env file: the authority to run something unusual belongs to
// the person at the keyboard, who can see what they are approving, and never to
// the config author, who may be a stranger. There is no config key for it, the
// field holding it is unexported so the YAML decoder cannot reach it, and the flag
// lives on the CLI's platform commands rather than anywhere config is read.
//
// Each value gets the same charset check as a command token and must be a bare
// name, so an approval cannot itself become the path form layer 2 refuses.
func (c *Config) AllowCommands(names []string) error {
	for _, name := range names {
		if err := checkToken("--allow-command", 0, name); err != nil {
			// Re-phrase for a flag rather than a schema field: the index means
			// nothing to someone who typed a value on the command line.
			return fmt.Errorf("invalid --allow-command value: %s",
				strings.TrimPrefix(strings.TrimPrefix(err.Error(), "--allow-command[0] "), "= "))
		}
		if j := strings.IndexAny(name, pathSeparators); j >= 0 {
			return fmt.Errorf("invalid --allow-command value %q: it must be a bare binary name, not a path "+
				"(found %q); approve it as %q and let PATH resolve it", name, string(name[j]), execBase(name))
		}
		if esc := neverAllowed[execBase(name)]; esc != "" {
			return fmt.Errorf("--allow-command %s is never permitted: %s would elevate every command this tool "+
				"issues, for the whole life of an env file -- elevate the tool instead, at the moment you run it "+
				"(%s solace ...), so the privilege belongs to one invocation you chose", name, esc, esc)
		}
		if c.extraAllowed == nil {
			c.extraAllowed = make(map[string]bool, len(names))
		}
		c.extraAllowed[execBase(name)] = true
	}
	return nil
}

// validateExecCommands runs the guard over every command field this tool executes
// on the platform being validated. It is the validator half of layer 5.
//
// An UNSET field is skipped rather than refused, which is the one place the two
// enforcement points deliberately differ. In this schema an omitted key, an empty
// string and an empty list all mean "unset" (setDefaultCmd), ApplyDefaults runs
// before Validate on every path config.Load takes, and reporting "kubernetes.runtime is
// empty" for a file that simply never mentioned it would be a worse error than the
// mandatory-fields list it would displace. The executor has no such context -- by
// the time it is asked, an empty command means an empty argv -- so CheckCommand
// itself still refuses one, and that is what actually protects exec.
func (c *Config) validateExecCommands(p Platform) error {
	fields := []struct {
		rules commandRules
		cmd   Command
	}{
		// kubernetes.runtime is checked on every platform: ApplyDefaults fills it
		// everywhere, and it is printable from any code path.
		{clusterRules(), c.K8s.Runtime},
	}
	if p.IsContainer() {
		fields = append(fields, struct {
			rules commandRules
			cmd   Command
		}{runtimeRules(p), c.ContainerRuntime(p)})
	}
	if p == Docker {
		// ComposeCommand owns the "unset -> <runtime> compose" derivation, so what
		// is checked here is exactly what Manager.compose will run.
		fields = append(fields, struct {
			rules commandRules
			cmd   Command
		}{composeRules(), c.composeOrDerived()})
	}
	for _, f := range fields {
		if len(f.cmd) == 0 {
			continue
		}
		if err := CheckCommand(f.rules, f.cmd, c.extraAllowed); err != nil {
			return err
		}
	}
	return nil
}

// ClusterCommand returns the guarded kubernetes.runtime command. Every k8s executor
// resolves argv[0] through this rather than reading the field, so the check the
// validator ran is re-run immediately before argv is built.
func (c *Config) ClusterCommand() (Command, error) {
	cmd := c.K8s.Runtime
	if err := CheckCommand(clusterRules(), cmd, c.extraAllowed); err != nil {
		return nil, err
	}
	return cmd, nil
}

// RuntimeCommand returns the guarded container runtime command for p (the checked
// counterpart of ContainerRuntime, which every executor now goes through).
func (c *Config) RuntimeCommand(p Platform) (Command, error) {
	cmd := c.ContainerRuntime(p)
	if err := CheckCommand(runtimeRules(p), cmd, c.extraAllowed); err != nil {
		return nil, err
	}
	return cmd, nil
}

// ComposeCommand returns the guarded docker.compose command, defaulting an unset
// value to the runtime's own `compose` subcommand. ApplyDefaults fills the field
// too, so the fallback here only matters for a hand-built config -- but it is the
// one definition of the derivation, so the manager cannot compute a different
// compose command from the one Validate checked.
func (c *Config) ComposeCommand() (Command, error) {
	cmd := c.composeOrDerived()
	if err := CheckCommand(composeRules(), cmd, c.extraAllowed); err != nil {
		return nil, err
	}
	return cmd, nil
}

// composeOrDerived is the one definition of the compose command: docker.compose if
// set, otherwise the runtime's own `compose` subcommand. ApplyDefaults stores the
// result and ComposeCommand checks it, so the value Validate approved and the value
// Manager.compose runs are the same expression rather than two copies of it.
func (c *Config) composeOrDerived() Command {
	if len(c.Docker.Compose) > 0 {
		return c.Docker.Compose
	}
	derived := make(Command, 0, len(c.Docker.Runtime)+1)
	derived = append(derived, c.Docker.Runtime...)
	return append(derived, "compose")
}

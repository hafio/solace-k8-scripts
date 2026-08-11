package k8s

import (
	"context"
	"strings"

	"solace/internal/broker"
	"solace/internal/config"
	"solace/internal/engine"
)

// kubectlTransport implements broker.Transport by wrapping `kubectl exec/cp -n <ns>
// <pod> --` against the broker pod for a role. Broker pods are single-container, so
// no `-c` flag is used (050:30, 053:56, enter-solace-cli.sh:18). Every call routes
// through the injected engine.Runner, so --dry-run echoes the command without running
// it and tests capture the exact argv.
type kubectlTransport struct {
	r   engine.Runner
	cfg *config.Config
}

// NewTransport builds the Kubernetes broker.Transport over the given runner and
// config. The returned transport binds each role to pod <name>-pubsubplus-<role>-0
// in the broker namespace.
func NewTransport(r engine.Runner, cfg *config.Config) broker.Transport {
	return &kubectlTransport{r: r, cfg: cfg}
}

func (k *kubectlTransport) ns() string { return k.cfg.K8s.Namespace }

// cmd is the configured cluster CLI (k8s.runtime, default `kubectl`): argv[0]
// plus any leading arguments that precede every call's own.
func (k *kubectlTransport) cmd() config.Command { return k.cfg.K8s.Runtime }

// execArgs builds `exec [-i] -n <ns> <pod> -- argv...` for a role's pod. stdin adds
// the `-i` flag for commands that read stdin (Upload, OutputInput).
func (k *kubectlTransport) execArgs(role config.Role, stdin bool, argv []string) []string {
	args := []string{"exec"}
	if stdin {
		args = append(args, "-i")
	}
	args = append(args, "-n", k.ns(), podName(k.cfg, role), "--")
	return append(args, argv...)
}

func (k *kubectlTransport) Run(ctx context.Context, role config.Role, argv ...string) error {
	kc := k.cmd()
	return k.r.Run(ctx, kc.Name(), kc.Args(k.execArgs(role, false, argv)...)...)
}

func (k *kubectlTransport) Output(ctx context.Context, role config.Role, argv ...string) ([]byte, error) {
	kc := k.cmd()
	return k.r.Output(ctx, kc.Name(), kc.Args(k.execArgs(role, false, argv)...)...)
}

func (k *kubectlTransport) OutputInput(ctx context.Context, role config.Role, in []byte, argv ...string) ([]byte, error) {
	kc := k.cmd()
	return k.r.OutputInput(ctx, in, kc.Name(), kc.Args(k.execArgs(role, true, argv)...)...)
}

// Upload writes data to destPath inside the pod by piping it to `sh -c 'cat >
// <dest>'` on stdin -- the secret-safe path: the body rides RunInput's stdin (shown
// as a byte count under --dry-run), never an argv or a temp file. destPath is
// tool-generated and validName-checked upstream, so shSingleQuote is defensive.
func (k *kubectlTransport) Upload(ctx context.Context, role config.Role, data []byte, destPath string) error {
	shcmd := "cat > " + shSingleQuote(destPath)
	kc := k.cmd()
	argv := k.execArgs(role, true, []string{"sh", "-c", shcmd})
	return k.r.RunInput(ctx, data, kc.Name(), kc.Args(argv...)...)
}

// UploadFile copies a local file into the pod via
// `kubectl cp -n <ns> <local> <pod>:<dest>` (051:51, 069:196, copy scripts:53).
func (k *kubectlTransport) UploadFile(ctx context.Context, role config.Role, localPath, destPath string) error {
	kc := k.cmd()
	return k.r.Run(ctx, kc.Name(), kc.Args("cp", "-n", k.ns(), localPath, podName(k.cfg, role)+":"+destPath)...)
}

// Download copies remotePath from the pod to localPath via
// `kubectl cp -n <ns> <pod>:<remote> <local>` (051:53, 069:196, copy scripts:42).
func (k *kubectlTransport) Download(ctx context.Context, role config.Role, remotePath, localPath string) error {
	kc := k.cmd()
	return k.r.Run(ctx, kc.Name(), kc.Args("cp", "-n", k.ns(), podName(k.cfg, role)+":"+remotePath, localPath)...)
}

// shSingleQuote wraps s in single quotes for safe use inside `sh -c`, escaping any
// embedded single quote (' -> '\''). Defensive: the only value passed to it is a
// tool-generated, validName-checked destPath, but this guarantees no shell
// metacharacter in a path can break out of the `cat >` redirect (§3).
func shSingleQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

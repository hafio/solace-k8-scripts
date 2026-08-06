// Package container implements the docker/podman ("container") platform: a
// node-local broker.Transport over `<engine> exec/cp`, plus a host Manager that
// deploys and operates one broker container per host (the analog of k8s.Cluster).
// It ports the docker-podman/*.sh script family. Unlike k8s -- where one control
// point drives every pod in a namespace -- a container host runs a single broker,
// so the transport is node-local and ignores the role argument, and the HA
// coordination (leader, redundancy) is a per-host handshake (see broker's
// LeaderLocal/RedundancyLocal).
package container

import (
	"context"
	"strings"

	"solace/internal/broker"
	"solace/internal/config"
	"solace/internal/engine"
)

// containerTransport implements broker.Transport by wrapping `<runtime> exec/cp`
// against the single broker container on this host. Node-local: there is one
// container per host, so the role argument is ignored (the k8s transport maps it
// to a pod; here every call targets the same container). Every call routes through
// the injected engine.Runner, so --dry-run echoes the command without running it
// and tests capture the exact argv.
type containerTransport struct {
	r       engine.Runner
	runtime string // docker | podman (or an override / drop-in)
	name    string // container name
}

// NewTransport builds the container broker.Transport over the given runner,
// config, and platform. It captures the resolved runtime binary
// (cfg.ContainerRuntime(p)) and container name (cfg.ContainerBlock(p).Name) once;
// the role argument on every method is ignored (node-local, one container/host).
func NewTransport(r engine.Runner, cfg *config.Config, p config.Platform) broker.Transport {
	return &containerTransport{
		r:       r,
		runtime: cfg.ContainerRuntime(p),
		name:    cfg.ContainerBlock(p).Name,
	}
}

// execArgs builds `exec [-i] <name> argv...`. stdin adds `-i` for commands that
// read stdin (Upload, OutputInput). No `--` separator: docker `exec` rejects it,
// and it is unnecessary for both engines since the container name is positional.
func (c *containerTransport) execArgs(stdin bool, argv []string) []string {
	args := []string{"exec"}
	if stdin {
		args = append(args, "-i")
	}
	args = append(args, c.name)
	return append(args, argv...)
}

func (c *containerTransport) Run(ctx context.Context, _ config.Role, argv ...string) error {
	return c.r.Run(ctx, c.runtime, c.execArgs(false, argv)...)
}

func (c *containerTransport) Output(ctx context.Context, _ config.Role, argv ...string) ([]byte, error) {
	return c.r.Output(ctx, c.runtime, c.execArgs(false, argv)...)
}

func (c *containerTransport) OutputInput(ctx context.Context, _ config.Role, in []byte, argv ...string) ([]byte, error) {
	return c.r.OutputInput(ctx, in, c.runtime, c.execArgs(true, argv)...)
}

// Upload writes data to destPath inside the container by piping it to `sh -c 'cat
// > <dest>'` on stdin -- the secret-safe path: the body rides RunInput's stdin
// (shown as a byte count under --dry-run), never an argv or a temp file. destPath
// is tool-generated and validName-checked upstream, so shSingleQuote is defensive.
func (c *containerTransport) Upload(ctx context.Context, _ config.Role, data []byte, destPath string) error {
	shcmd := "cat > " + shSingleQuote(destPath)
	argv := c.execArgs(true, []string{"sh", "-c", shcmd})
	return c.r.RunInput(ctx, data, c.runtime, argv...)
}

// UploadFile copies a local file into the container via
// `<runtime> cp <local> <name>:<dest>`.
func (c *containerTransport) UploadFile(ctx context.Context, _ config.Role, localPath, destPath string) error {
	return c.r.Run(ctx, c.runtime, "cp", localPath, c.name+":"+destPath)
}

// Download copies remotePath from the container to localPath via
// `<runtime> cp <name>:<remote> <local>`.
func (c *containerTransport) Download(ctx context.Context, _ config.Role, remotePath, localPath string) error {
	return c.r.Run(ctx, c.runtime, "cp", c.name+":"+remotePath, localPath)
}

// shSingleQuote wraps s in single quotes for safe use inside `sh -c`, escaping any
// embedded single quote (' -> '\''). Defensive: the only value passed to it is a
// tool-generated, validName-checked destPath, but this guarantees no shell
// metacharacter in a path can break out of the `cat >` redirect (§3). A
// package-local copy of the k8s helper -- deliberately not over-DRY'd across
// packages, matching the house style of one small copy per transport.
func shSingleQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

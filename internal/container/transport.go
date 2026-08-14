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
	r    engine.Runner
	cfg  *config.Config
	p    config.Platform
	name string // container name
}

// NewTransport builds the container broker.Transport over the given runner,
// config, and platform. It captures the container name (cfg.ContainerBlock(p).Name)
// once; the role argument on every method is ignored (node-local, one
// container/host). The runtime command is deliberately NOT captured -- it is
// resolved through the execution guard on every call (runtime below).
func NewTransport(r engine.Runner, cfg *config.Config, p config.Platform) broker.Transport {
	return &containerTransport{
		r:    r,
		cfg:  cfg,
		p:    p,
		name: cfg.ContainerBlock(p).Name,
	}
}

// runtime is the guarded container runtime command (docker.runtime /
// podman.runtime). It re-runs config.CheckCommand on every call rather than
// caching the value at construction: the transport can be built from a Config that
// never went through config.Load, so this is the executor half of the guard's two
// enforcement points and must not trust what it was handed.
func (c *containerTransport) runtime() (config.Command, error) {
	return c.cfg.RuntimeCommand(c.p)
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
	rt, err := c.runtime()
	if err != nil {
		return err
	}
	return c.r.Run(ctx, rt.Name(), rt.Args(c.execArgs(false, argv)...)...)
}

func (c *containerTransport) Output(ctx context.Context, _ config.Role, argv ...string) ([]byte, error) {
	rt, err := c.runtime()
	if err != nil {
		return nil, err
	}
	return c.r.Output(ctx, rt.Name(), rt.Args(c.execArgs(false, argv)...)...)
}

func (c *containerTransport) OutputInput(ctx context.Context, _ config.Role, in []byte, argv ...string) ([]byte, error) {
	rt, err := c.runtime()
	if err != nil {
		return nil, err
	}
	return c.r.OutputInput(ctx, in, rt.Name(), rt.Args(c.execArgs(true, argv)...)...)
}

// Upload writes data to destPath inside the container by piping it to `sh -c 'cat
// > <dest>'` on stdin -- the secret-safe path: the body rides RunInput's stdin
// (shown as a byte count under --dry-run), never an argv or a temp file. destPath
// is tool-generated and validName-checked upstream, so shSingleQuote is defensive.
func (c *containerTransport) Upload(ctx context.Context, _ config.Role, data []byte, destPath string) error {
	rt, err := c.runtime()
	if err != nil {
		return err
	}
	shcmd := "cat > " + shSingleQuote(destPath)
	argv := c.execArgs(true, []string{"sh", "-c", shcmd})
	return c.r.RunInput(ctx, data, rt.Name(), rt.Args(argv...)...)
}

// UploadFile copies a local file into the container via
// `<runtime> cp <local> <name>:<dest>`.
func (c *containerTransport) UploadFile(ctx context.Context, _ config.Role, localPath, destPath string) error {
	rt, err := c.runtime()
	if err != nil {
		return err
	}
	return c.r.Run(ctx, rt.Name(), rt.Args("cp", localPath, c.name+":"+destPath)...)
}

// Download copies remotePath from the container to localPath via
// `<runtime> cp <name>:<remote> <local>`.
func (c *containerTransport) Download(ctx context.Context, _ config.Role, remotePath, localPath string) error {
	rt, err := c.runtime()
	if err != nil {
		return err
	}
	return c.r.Run(ctx, rt.Name(), rt.Args("cp", c.name+":"+remotePath, localPath)...)
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

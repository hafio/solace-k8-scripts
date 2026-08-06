// Package broker implements the post-deploy config and verify operations that
// run against a *running* Solace broker over the Solace CLI and SEMP. These are
// runtime operations -- they don't care whether the broker runs in a Kubernetes
// pod or a host container -- so they are written once here and reused by every
// platform. Only the exec transport differs: k8s wraps `kubectl exec/cp -n <ns>
// <pod> --`, containers wrap `<engine> exec/cp <container>`.
//
// It is the shared port of the numbered CLI scripts: 050 (assert-leader), 051
// (server-cert), 052 (domain-certs), 053 (disable-default-vpn), 054
// (disable-default-users), 057 (product-keys), 059 (exec-cli), 060 (login), 061
// (redundancy), 069 (diagnostics).
package broker

import (
	"context"

	"solace/internal/config"
)

// In-broker paths the PubSub+ image exposes. The CLI reads `-Apes .<name>.cli`
// relative to the cliscripts dir (its working directory), so RunCLI uploads the
// script to CLIScriptsDir and passes the leading-dot relative name to the binary.
const (
	JailRoot      = "/usr/sw/jail"
	CLIScriptsDir = JailRoot + "/cliscripts"
	CertsDir      = JailRoot + "/certs"
	CLIBinary     = "/usr/sw/loads/currentload/bin/cli"
)

// Transport runs commands and moves files in a broker node identified by role.
// The k8s transport maps a role to the pod <name>-pubsubplus-<role>-0 in the
// broker namespace; container transports are node-local -- one container per
// host -- and ignore the role. All methods must respect the injected engine
// Runner so --dry-run echoes commands and never runs them, and so secrets fed
// on stdin (Upload data, OutputInput) are never echoed verbatim.
type Transport interface {
	// Run executes argv in the node's broker container, streaming stdout+stderr.
	Run(ctx context.Context, role config.Role, argv ...string) error
	// Output executes argv and returns captured stdout; stderr is streamed.
	Output(ctx context.Context, role config.Role, argv ...string) ([]byte, error)
	// OutputInput is Output with stdin fed from in. Used for secret-bearing
	// commands (curl credentials) so the secret rides stdin, never the argv/log.
	OutputInput(ctx context.Context, role config.Role, in []byte, argv ...string) ([]byte, error)
	// Upload writes data to destPath inside the node, creating/truncating it.
	// data may hold secrets (private keys); implementations feed it on stdin.
	Upload(ctx context.Context, role config.Role, data []byte, destPath string) error
	// UploadFile copies a local file into destPath inside the node.
	UploadFile(ctx context.Context, role config.Role, localPath, destPath string) error
	// Download copies remotePath from the node to localPath on this host.
	Download(ctx context.Context, role config.Role, remotePath, localPath string) error
}

// cliScriptPath is the in-broker path RunCLI uploads a script to: a hidden file
// in the cliscripts dir, matching the `.<name>.cli` convention of the bash scripts.
func cliScriptPath(name string) string { return CLIScriptsDir + "/." + name + ".cli" }

// cliArg is the relative script name passed to the CLI binary (cwd = cliscripts).
func cliArg(name string) string { return "." + name + ".cli" }

// certPath is the in-broker path a certificate file is uploaded to.
func certPath(filename string) string { return CertsDir + "/" + filename }

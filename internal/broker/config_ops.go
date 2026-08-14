package broker

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"solace/internal/config"
)

// ServerCert loads the TLS server certificate into each of roles over the Solace
// CLI, porting the CLI branch of 051 (the $SOLBK_SVR_SECRET k8s-secret fast path
// is handled by the k8s platform, not here). It concatenates key + cert + CAs
// into the tls-<dt>.crt.key file the broker loads. The private key rides Upload's
// stdin, so it never appears in an argv or --dry-run echo (§3).
func (o *Ops) ServerCert(ctx context.Context, dt string, roles ...config.Role) error {
	if o.Cfg.TLS.Cert == "" || o.Cfg.TLS.CertKey == "" {
		return fmt.Errorf("tls.cert and tls.certKey must both be set to load a server certificate")
	}
	bundle, err := concatFiles(o.Cfg.TLS.CertKey, o.Cfg.TLS.Cert)
	if err != nil {
		return err
	}
	for _, ca := range o.Cfg.TLS.CAs {
		caBytes, err := os.ReadFile(ca)
		if err != nil {
			return fmt.Errorf("read tls CA %q: %w", ca, err)
		}
		bundle = append(bundle, caBytes...)
	}

	file := serverCertFile(dt)
	for _, role := range roles {
		o.logf("Loading server certificate into %q node...", role)
		if err := o.T.Upload(ctx, role, bundle, certPath(file)); err != nil {
			return fmt.Errorf("upload certificate into %q node: %w", role, err)
		}
		out, err := o.RunCLI(ctx, role, "apply-server-certs", serverCertScript(dt))
		if err != nil {
			return err
		}
		o.show(out)
	}
	return nil
}

// DomainCerts loads the given domain certificate authorities into the node,
// porting 052. files maps CA name -> certificate filename located under folder;
// each file is uploaded to the certs dir and referenced by the generated CLI.
func (o *Ops) DomainCerts(ctx context.Context, role config.Role, folder string, files map[string]string) error {
	if len(files) == 0 {
		o.logf("No domain certificate authorities configured -- skipping.")
		return nil
	}
	for _, ca := range sortedKeys(files) {
		file := files[ca]
		if err := validName("domain CA name", ca); err != nil {
			return err
		}
		if err := validName("domain certificate filename", file); err != nil {
			return err
		}
		if err := o.T.UploadFile(ctx, role, filepath.Join(folder, file), certPath(file)); err != nil {
			return fmt.Errorf("upload domain certificate %q: %w", file, err)
		}
	}
	out, err := o.RunCLI(ctx, role, "load-domain-certs", domainCertsScript(files))
	if err != nil {
		return err
	}
	o.show(out)
	return nil
}

// DisableDefaultVPN shuts down the default message-VPN and its default
// client-username on the node, porting 053, then shows the resulting VPN list.
func (o *Ops) DisableDefaultVPN(ctx context.Context, role config.Role) error {
	if _, err := o.RunCLI(ctx, role, "disable-default-vpn", disableDefaultVPNScript()); err != nil {
		return err
	}
	out, err := o.RunCLI(ctx, role, "show-vpn", showVPNScript())
	if err != nil {
		return err
	}
	o.show(out)
	o.removeCLI(ctx, role, "disable-default-vpn", "show-vpn")
	return nil
}

// DisableDefaultUsers shuts down the "default" client-username in every VPN on
// the node, porting 054: list the VPNs, parse their names, then shut each down.
func (o *Ops) DisableDefaultUsers(ctx context.Context, role config.Role) error {
	list, err := o.RunCLI(ctx, role, "show-vpn", showVPNBareScript())
	if err != nil {
		return err
	}
	vpns := parseVPNNames(string(list))
	if len(vpns) == 0 {
		o.logf("[WARN] no message-VPNs parsed from broker output -- nothing to disable.")
		return nil
	}
	out, err := o.RunCLI(ctx, role, "disable-default-usernames", disableDefaultUsersScript(vpns))
	if err != nil {
		return err
	}
	o.show(out)
	return nil
}

// ProductKeys applies keys to each of roles (Primary, plus Backup in HA),
// porting 057. It fails loud if the broker reports an error in the output.
func (o *Ops) ProductKeys(ctx context.Context, keys []string, roles ...config.Role) error {
	if len(keys) == 0 {
		return fmt.Errorf("no product keys configured")
	}
	// Each key is interpolated into a line of a CLI script that runs with admin
	// already enabled, so it is checked before anything is uploaded -- the sibling
	// DomainCerts does the same for CA names and filenames.
	for _, k := range keys {
		if err := validCLILine("product key", k); err != nil {
			return err
		}
	}
	var combined bytes.Buffer
	for _, role := range roles {
		o.logf("Applying product key(s) to %q node...", role)
		out, err := o.RunCLI(ctx, role, "product-keys", productKeysScript(keys))
		if err != nil {
			return err
		}
		combined.Write(out)
		o.show(out)
		o.removeCLI(ctx, role, "product-keys")
	}
	if containsAnyFold(combined.String(), "error", "fail") {
		return fmt.Errorf("product-key application reported an error (see output above)")
	}
	return nil
}

// AdditionalUsers creates the management (CLI) users listed in
// admin.additionalUsers, each with its password and global access level. It exists
// because the operator has no declarative route for them: extra data keys in the
// credentials Secret are ignored, and extraEnvVars/extraEnvVarsSecret would put the
// passwords in the pod's environment. Container platforms do not use this -- they
// create the users at boot from the mounted secret file and the access-level setting.
//
// Unlike every sibling op it does NOT show the CLI output: `cli -Apes` echoes the
// command lines it ran, which here contain the passwords. The output is inspected in
// memory and discarded. (The transport still streams the child's stderr, so a
// process-level failure message from the CLI binary itself can reach the terminal;
// the transcript, which is what carries the commands, goes to stdout and is captured.)
func (o *Ops) AdditionalUsers(ctx context.Context, role config.Role, users []config.AdditionalUser) error {
	if len(users) == 0 {
		return fmt.Errorf("no additional users configured")
	}
	// Every value is interpolated into a CLI line, so all of them are checked before
	// anything is uploaded -- the same order ProductKeys and DomainCerts use.
	for _, u := range users {
		if err := validName("additional user", u.Username); err != nil {
			return err
		}
		if err := validCLILine("additional user access level", u.AccessLevel); err != nil {
			return err
		}
		if err := validCLIPassword(u.Username, u.Password); err != nil {
			return err
		}
	}

	const script = "additional-users"
	o.logf("Creating %d additional CLI user(s) on the %q node...", len(users), role)
	// Deferred, not sequential: the uploaded body carries every password, so a failed
	// run must not leave it behind in the broker's cliscripts dir.
	defer o.removeCLI(ctx, role, script)
	out, err := o.RunCLI(ctx, role, script, additionalUsersScript(users))
	if err != nil {
		return err
	}
	// `create username` fails when the user already exists. That is reported rather
	// than reconciled: silently re-setting a password an operator may have rotated on
	// the broker is worse than refusing.
	if containsAnyFold(string(out), "error", "fail", "already exists", "invalid") {
		return fmt.Errorf("creating additional users on the %q node reported an error; the output is "+
			"withheld because it repeats the passwords. A user that already exists is the likeliest "+
			"cause -- delete it on the broker, or drop it from admin.additionalUsers", role)
	}
	o.logf("[ OK ] created %d additional CLI user(s).", len(users))
	return nil
}

// ExecCLI uploads a local Solace CLI script and runs it in the node, porting 059.
// The remote name is the file's basename, validated to keep it out of shell/CLI
// injection range. It warns (does not fail) when the output looks like an error,
// matching the bash behavior.
func (o *Ops) ExecCLI(ctx context.Context, role config.Role, localPath string) error {
	name := filepath.Base(localPath)
	if err := validName("cli script filename", name); err != nil {
		return err
	}
	if err := o.T.UploadFile(ctx, role, localPath, cliScriptPath(name)); err != nil {
		return fmt.Errorf("upload cli script %q: %w", name, err)
	}
	out, err := o.T.Output(ctx, role, CLIBinary, "-Apes", cliArg(name))
	o.show(out)
	o.removeCLI(ctx, role, name)
	if err != nil {
		return fmt.Errorf("run cli script %q: %w", name, err)
	}
	if containsAnyFold(string(out), "invalid", "error", "busy") {
		o.logf("[WARN] errors detected in CLI output.")
	}
	return nil
}

// RemoveDomainCerts deletes each domain certificate authority from the node,
// porting 150 (k8s teardown domain-certs). cas are the CA names to remove; they
// are sorted for deterministic output and validated to keep them out of CLI
// injection range (§3). Empty input is a no-op with a log line.
func (o *Ops) RemoveDomainCerts(ctx context.Context, role config.Role, cas []string) error {
	if len(cas) == 0 {
		o.logf("No domain certificate authorities configured -- nothing to remove.")
		return nil
	}
	sorted := append([]string(nil), cas...)
	sort.Strings(sorted)
	for _, ca := range sorted {
		if err := validName("domain CA name", ca); err != nil {
			return err
		}
	}
	out, err := o.RunCLI(ctx, role, "remove-domain-certs", removeDomainCertsScript(sorted))
	if err != nil {
		return err
	}
	o.show(out)
	o.removeCLI(ctx, role, "remove-domain-certs")
	return nil
}

// concatFiles reads paths in order and returns their concatenation.
func concatFiles(paths ...string) ([]byte, error) {
	var buf bytes.Buffer
	for _, p := range paths {
		b, err := os.ReadFile(p)
		if err != nil {
			return nil, fmt.Errorf("read %q: %w", p, err)
		}
		buf.Write(b)
	}
	return buf.Bytes(), nil
}

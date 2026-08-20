package k8s

import (
	"context"
	"fmt"
	"path"
	"strings"

	"solace/internal/config"
)

// rolloutTimeout bounds the readiness wait after scaling a broker StatefulSet up.
// The bash replicas-start busy-waited forever (replicas-start-broker.sh:19-21); this
// port waits via `kubectl rollout status --timeout` so a stuck pod fails loud instead
// of hanging.
const rolloutTimeout = "300s"

// Status prints the broker's pods, services and StatefulSets in the broker namespace,
// porting get-broker-status.sh:16-20. Each `get` streams straight through the runner,
// so --dry-run echoes the three commands.
func (c *Cluster) Status(ctx context.Context) error {
	if err := c.kubectl(ctx, "get", "pods", "-n", c.ns(), "-o", "wide"); err != nil {
		return err
	}
	if err := c.kubectl(ctx, "get", "svc", "-n", c.ns()); err != nil {
		return err
	}
	return c.kubectl(ctx, "get", "statefulset", "-n", c.ns())
}

// showAllSections drives ShowAll. Broker pods and StatefulSets carry the -pubsubplus-
// infix, which excludes the operator's own pod (pubsubplus-eventbroker-operator-*, no
// leading dash); services match the looser "pubsubplus" so the LB service
// <name>-pubsubplus (no trailing -role) is included too (show-all-brokers.sh:31,74,90).
type surveySection struct {
	title    string
	resource string
	wide     bool
	filter   string
}

// showAllSections is the RUNNING picture: what is deployed and serving. The
// operator leads because it is what everything else depends on, and it is matched
// on its own name rather than the -pubsubplus- infix the broker resources carry.
var showAllSections = []surveySection{
	{"OPERATOR", "deployments", true, operatorDeployment},
	{"BROKERS", "pubsubpluseventbrokers", false, ""},
	{"PODS", "pods", true, "-pubsubplus-"},
	{"SERVICES", "svc", false, "pubsubplus"},
	{"STATEFULSETS", "statefulsets", false, "-pubsubplus-"},
}

// showDetailSections is what --detail adds: the STATIC artifacts a broker is built
// from, which outlive any particular pod. They are listed separately because they
// answer a different question -- not "is it running" but "what is it made of", and
// a PVC left behind by a removed broker is exactly the kind of thing that only
// shows up when you go looking.
var showDetailSections = []surveySection{
	{"SECRETS", "secrets", false, "pubsubplus"},
	{"CONFIGMAPS", "configmaps", false, "pubsubplus"},
	{"PERSISTENT VOLUME CLAIMS", "pvc", false, "pubsubplus"},
}

// ShowAll lists broker pods, services and StatefulSets across every namespace,
// porting show-all-brokers.sh. It replaces the bash jq column-formatting with native
// `kubectl get -A` output filtered client-side to broker resources -- the plain table
// kubectl prints, minus the custom AGE/DISK math, which was flagged as a deliberate
// simplification. Filtering needs the output captured, so under --dry-run (Echo) the
// get is echoed and the filter finds nothing.
func (c *Cluster) ShowAll(ctx context.Context, detail bool) error {
	sections := showAllSections
	if detail {
		sections = append(append([]surveySection{}, sections...), showDetailSections...)
	}
	return c.survey(ctx, sections, true)
}

// Survey is ShowAll scoped to this env file's namespace: the same picture, of the
// one broker the config describes. `--all` is what widens it to the cluster.
func (c *Cluster) Survey(ctx context.Context, detail bool) error {
	sections := showAllSections
	if detail {
		sections = append(append([]surveySection{}, sections...), showDetailSections...)
	}
	return c.survey(ctx, sections, false)
}

// survey lists each section, either cluster-wide or in the broker's namespace, and
// filters the rows client-side to the ones belonging to a Solace deployment. A
// section that fails is reported and skipped rather than aborting the rest: this is
// a report, and one resource kind the context cannot list (RBAC, or a CRD that is
// not installed) must not hide the kinds it can.
func (c *Cluster) survey(ctx context.Context, sections []surveySection, allNamespaces bool) error {
	w := c.out()
	for _, s := range sections {
		args := []string{"get", s.resource}
		if allNamespaces {
			args = append(args, "--all-namespaces")
		} else {
			args = append(args, "-n", c.ns())
		}
		if s.wide {
			args = append(args, "-o", "wide")
		}
		fmt.Fprintf(w, "### %s ###\n", s.title)
		raw, err := c.output(ctx, args...)
		if err != nil {
			fmt.Fprintf(w, "  (could not list %s: %v)\n", s.resource, err)
			continue
		}
		fmt.Fprintln(w, filterLines(string(raw), s.filter))
	}
	return nil
}

// filterLines keeps the table header (first line) plus every data line containing
// substr, so the client-side broker filter preserves column titles. An empty capture
// (e.g. --dry-run, or no such resources) reports "(none)".
func filterLines(raw, substr string) string {
	lines := strings.Split(strings.TrimRight(raw, "\n"), "\n")
	if len(lines) == 0 || lines[0] == "" {
		return "  (none)"
	}
	kept := []string{lines[0]}
	for _, ln := range lines[1:] {
		if strings.Contains(ln, substr) {
			kept = append(kept, ln)
		}
	}
	if len(kept) == 1 {
		return kept[0] + "\n  (none matched)"
	}
	return strings.Join(kept, "\n")
}

// DescribeBroker describes a role's broker pod, porting desc-broker.sh:18.
func (c *Cluster) DescribeBroker(ctx context.Context, role config.Role) error {
	return c.kubectl(ctx, "describe", "pod", "-n", c.ns(), podName(c.Cfg, role))
}

// DescribeLB describes the broker's load-balancer Service, porting desc-lb.sh:16.
func (c *Cluster) DescribeLB(ctx context.Context) error {
	return c.kubectl(ctx, "describe", "service/"+lbServiceName(c.Cfg), "-n", c.ns())
}

// Logs streams a role's pod logs, porting logs-broker.sh:20. passthrough carries any
// extra `kubectl logs` args (-f, --tail=N, ...) straight through.
func (c *Cluster) Logs(ctx context.Context, role config.Role, passthrough []string) error {
	args := append([]string{"logs", "-n", c.ns(), "pod/" + podName(c.Cfg, role)}, passthrough...)
	return c.kubectl(ctx, args...)
}

// interactiveExec runs `kubectl exec -it -n <ns> <pod> -- argv...` with this process's
// stdio wired through, for the interactive sessions (Solace CLI, shell).
func (c *Cluster) interactiveExec(ctx context.Context, role config.Role, argv ...string) error {
	args := append([]string{"exec", "-it", "-n", c.ns(), podName(c.Cfg, role), "--"}, argv...)
	k, err := c.cmd()
	if err != nil {
		return err
	}
	return c.R.RunInteractive(ctx, k.Name(), k.Args(args...)...)
}

// CLI opens an interactive Solace CLI session on a role's pod, porting
// enter-solace-cli.sh:18 (`cli -A`, the bare in-pod launcher, not the full load path).
func (c *Cluster) CLI(ctx context.Context, role config.Role) error {
	return c.interactiveExec(ctx, role, "cli", "-A")
}

// Shell opens an interactive shell on a role's pod (no bash script equivalent; the
// operational analogue of CLI for host-level troubleshooting).
func (c *Cluster) Shell(ctx context.Context, role config.Role) error {
	return c.interactiveExec(ctx, role, "bash")
}

// CopyFrom copies files out of a role's pod into the current directory, porting
// copy-files-from-broker.sh:42 (each lands under its basename). It attempts every
// file and fails loud at the end if any copy failed, rather than aborting on the
// first -- so one bad path does not strand the rest.
func (c *Cluster) CopyFrom(ctx context.Context, role config.Role, files []string) error {
	if len(files) == 0 {
		return fmt.Errorf("no files specified to copy from the broker")
	}
	t := NewTransport(c.R, c.Cfg)
	var failed int
	for _, f := range files {
		local := path.Base(f)
		c.logf("copying %s from %s", f, roleName(role))
		if err := t.Download(ctx, role, f, local); err != nil {
			fmt.Fprintf(c.out(), "  [ERROR] %s: %v\n", f, err)
			failed++
			continue
		}
		fmt.Fprintf(c.out(), "  [ OK ] %s -> %s\n", f, local)
	}
	if failed > 0 {
		return fmt.Errorf("%d of %d file(s) failed to copy from the broker", failed, len(files))
	}
	return nil
}

// CopyInto copies local files into destDir inside a role's pod, porting
// copy-files-into-broker.sh:53 (default destDir "." is the pod's login directory).
// Like CopyFrom it attempts all files and fails loud at the end.
func (c *Cluster) CopyInto(ctx context.Context, role config.Role, files []string, destDir string) error {
	if len(files) == 0 {
		return fmt.Errorf("no files specified to copy into the broker")
	}
	if destDir == "" {
		destDir = "."
	}
	t := NewTransport(c.R, c.Cfg)
	var failed int
	for _, f := range files {
		c.logf("copying %s into %s:%s", f, roleName(role), destDir)
		if err := t.UploadFile(ctx, role, f, destDir); err != nil {
			fmt.Fprintf(c.out(), "  [ERROR] %s: %v\n", f, err)
			failed++
			continue
		}
		fmt.Fprintf(c.out(), "  [ OK ] %s -> %s:%s\n", f, roleName(role), destDir)
	}
	if failed > 0 {
		return fmt.Errorf("%d of %d file(s) failed to copy into the broker", failed, len(files))
	}
	return nil
}

// ReplicasStart scales each broker StatefulSet up to one replica and waits for it to
// become ready before moving on, porting replicas-start-broker.sh. It scales the roles
// in order (primary, then backup, then monitor in HA), so the primary is ready before
// the backup joins; unlike the bash original it also waits on the monitor, and the
// wait is bounded by rolloutTimeout instead of an unbounded busy-wait.
// RestartPod deletes a role's broker pod so the StatefulSet controller recreates it
// against the pod template the operator has already updated. This is the step
// kubernetes.updateStrategy=manualPodRestart requires after `deploy` changes image.tag:
// the operator updates the template and then waits for a human, so without this
// there was no in-tool way to finish an upgrade. --ignore-not-found makes a repeat
// call harmless, and the readiness wait is bounded like ReplicasStart's.
func (c *Cluster) RestartPod(ctx context.Context, role config.Role) error {
	pod := podName(c.Cfg, role)
	c.logf("deleting pod %s so the statefulset recreates it", pod)
	if err := c.kubectl(ctx, "delete", "pod", "-n", c.ns(), pod, "--ignore-not-found"); err != nil {
		return fmt.Errorf("deleting pod %s: %w", pod, err)
	}
	sts := stsName(c.Cfg, role)
	c.logf("waiting for %s to become ready", sts)
	if err := c.kubectl(ctx, "rollout", "status", "statefulset/"+sts, "-n", c.ns(), "--timeout="+rolloutTimeout); err != nil {
		return fmt.Errorf("%s did not become ready within %s after restarting %s: %w", sts, rolloutTimeout, pod, err)
	}
	return nil
}

// RestartRolling restarts every broker pod in RestartOrder, stopping at the first
// failure so a broken restart never cascades into the next role.
func (c *Cluster) RestartRolling(ctx context.Context) error {
	for _, role := range RestartOrder(c.Cfg) {
		if err := c.RestartPod(ctx, role); err != nil {
			return err
		}
	}
	return nil
}

func (c *Cluster) ReplicasStart(ctx context.Context) error {
	for _, role := range HARoles(c.Cfg) {
		sts := stsName(c.Cfg, role)
		c.logf("scaling %s up to 1 replica", sts)
		if err := c.kubectl(ctx, "scale", "statefulset", sts, "-n", c.ns(), "--replicas=1"); err != nil {
			return fmt.Errorf("scaling %s up: %w", sts, err)
		}
		c.logf("waiting for %s to become ready", sts)
		if err := c.kubectl(ctx, "rollout", "status", "statefulset/"+sts, "-n", c.ns(), "--timeout="+rolloutTimeout); err != nil {
			return fmt.Errorf("%s did not become ready within %s: %w", sts, rolloutTimeout, err)
		}
	}
	return nil
}

// ReplicasStop scales every broker StatefulSet down to zero replicas, porting
// replicas-stop-broker.sh:23-26. The bash script's y/n confirmation lives in the CLI
// layer; here the decision to stop has already been made.
func (c *Cluster) ReplicasStop(ctx context.Context) error {
	for _, role := range HARoles(c.Cfg) {
		sts := stsName(c.Cfg, role)
		c.logf("scaling %s down to 0 replicas", sts)
		if err := c.kubectl(ctx, "scale", "statefulset", sts, "-n", c.ns(), "--replicas=0"); err != nil {
			return fmt.Errorf("scaling %s down: %w", sts, err)
		}
	}
	return nil
}

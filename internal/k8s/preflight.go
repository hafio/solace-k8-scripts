package k8s

import (
	"context"
	"fmt"
	"strings"
)

// Preflight is the read-only probe every mutating operation runs first. It asks the
// cluster one question -- may this identity create this resource here -- using the
// same guarded argv prefix the real work will use, so what it proves is what the
// next command will actually do.
//
// It exists because the alternative failure mode is worse than a slow start: a
// deploy that writes .broker.yaml, applies half a manifest set, and then stops on
// an expired token leaves the operator to work out which half landed. Failing
// before the first byte is written keeps "nothing happened" a true statement.
//
// There is deliberately no skip flag. A probe that can be turned off is a probe
// that is off in exactly the scripted runs that most need it, and the only
// legitimate reason to skip it -- previewing without a cluster -- is already
// --dry-run, which reaches the branch below.
//
// It never logs anyone in. Authentication is the operator's business and their
// credential store's; a tool that offered to fix an auth failure would be teaching
// people to hand it credentials it has no business holding.
func (c *Cluster) Preflight(ctx context.Context, verb, resource string) error {
	if c.isDryRun() {
		// Echo the probe so a preview still shows it, then skip the assertion --
		// the Echo runner answers nothing, and there is no cluster to answer.
		if err := c.kubectl(ctx, "auth", "can-i", verb, resource, "-n", c.ns()); err != nil {
			return err
		}
		fmt.Fprintln(c.out(), "  permission     : skipped (dry-run)")
		return nil
	}

	out, err := c.output(ctx, "auth", "can-i", verb, resource, "-n", c.ns())
	answer := canIAnswer(out)
	switch {
	case err == nil && answer == "yes":
		return nil
	case answer == "no":
		// Reached the API server and got a real answer: this is RBAC, not auth.
		// kubectl already printed its own explanation to stderr; add the one line
		// that says what to ask for.
		return fmt.Errorf("not allowed to %s %s in namespace %q: ask a cluster admin for a role binding "+
			"granting %s on %s there, then re-run", verb, resource, c.ns(), verb, resource)
	case err != nil:
		// No usable answer: an expired token, no context, an unreachable API
		// server. kubectl's own message is on stderr and wrapped in here too.
		return fmt.Errorf("cannot check permission to %s %s in namespace %q: %w\n"+
			"  log in first (kubectl: `kubectl config use-context <ctx>`; OpenShift: `oc login <server>`), "+
			"or point k8s.runtime at the right profile", verb, resource, c.ns(), err)
	default:
		// Exit 0 with something other than "yes" -- a wrapper that swallowed the
		// answer, or a kubectl whose output shape changed. Refusing is the safe
		// direction: proceeding would mean assuming a permission nobody confirmed.
		return fmt.Errorf("could not read the answer to `auth can-i %s %s` (got %q); "+
			"if k8s.runtime wraps kubectl, make sure it passes stdout through unchanged", verb, resource, answer)
	}
}

// canIAnswer extracts the verdict from `auth can-i` output. The answer is the LAST
// non-empty line, not the whole trimmed body: kubectl prints advisory lines above it
// on stdout -- "Warning: resource 'x' is not namespace scoped" is the common one, and
// a cluster with deprecated APIs adds more. Comparing the whole output would turn
// every such cluster into the "could not read the answer" branch and block a deploy
// the operator is perfectly entitled to make.
func canIAnswer(out []byte) string {
	lines := strings.Split(string(out), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		if line := strings.TrimSpace(lines[i]); line != "" {
			return line
		}
	}
	return ""
}

// brokerResource is the CRD the operator reconciles -- the thing every broker
// deploy ultimately creates. Named in full (resource.group) so `auth can-i` cannot
// match a same-named resource in another group.
const brokerResource = "pubsubpluseventbrokers.pubsubplus.solace.com"

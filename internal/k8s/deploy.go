package k8s

import (
	"context"
	"fmt"
	"os"

	"solace/internal/render"
)

// brokerYAMLFile is where DeployBroker writes the rendered CR when keepYAML is set,
// mirroring the bash `.broker.yaml` the operator deploy leaves in the working
// directory (020:206-210).
const brokerYAMLFile = ".broker.yaml"

// DeployBroker renders the PubSubPlusEventBroker CR and applies it on stdin
// (020:211). When keepYAML is set it also writes the manifest to .broker.yaml in
// the current directory for inspection/version control (020:206-210). The CR
// carries only secret *names*, never secret values, so applying on stdin -- rather
// than the bash apply-from-file -- changes nothing about what lands on disk.
func (c *Cluster) DeployBroker(ctx context.Context, keepYAML bool) error {
	manifest := render.BrokerCR(c.Cfg)
	if keepYAML {
		if err := os.WriteFile(brokerYAMLFile, manifest, 0o644); err != nil {
			return fmt.Errorf("writing %s: %w", brokerYAMLFile, err)
		}
		c.logf("wrote broker manifest to %s", brokerYAMLFile)
	}
	c.logf("deploying broker %s in %s", c.Cfg.K8s.Name, c.ns())
	return c.apply(ctx, manifest)
}

// DeleteBroker removes the broker CR (120:55) via `delete -f - --ignore-not-found`
// of the rendered manifest, so a repeat teardown is a no-op. When purge is set it
// then best-effort deletes the per-role data PVCs (data-<name>-pubsubplus-<role>-0;
// 120:65-69) -- their errors are swallowed because the CR delete already released
// the pods, and a missing/renamed PVC must not fail the teardown.
//
// purge defaults to false at the call site (keep data by default), the deliberately
// safer inverse of legacy 120's purge-by-default. The confirm/flag logic lives in
// the CLI layer; here purge is just the decision already made.
func (c *Cluster) DeleteBroker(ctx context.Context, purge bool) error {
	c.logf("deleting broker %s in %s", c.Cfg.K8s.Name, c.ns())
	if err := c.deleteStdin(ctx, render.BrokerCR(c.Cfg)); err != nil {
		return err
	}
	if !purge {
		return nil
	}
	for _, role := range HARoles(c.Cfg) {
		pvc := pvcName(c.Cfg, role)
		c.logf("purging PVC %s", pvc)
		if err := c.kubectl(ctx, "delete", "pvc", pvc, "-n", c.ns(), "--ignore-not-found"); err != nil {
			// Best-effort: a released or already-absent PVC must not abort teardown.
			c.logf("  [WARN] could not delete PVC %s: %v", pvc, err)
		}
	}
	return nil
}

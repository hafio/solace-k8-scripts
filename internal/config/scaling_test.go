package config

import (
	"strconv"
	"strings"
	"testing"
)

// tierOrder is the ascending key order scalingTierList claims. Keeping it here
// rather than sorting the map lets TestScalingTierListMatchesTable catch both a
// list that drifts from the table and a tier added to one but not the other.
var tierOrder = []int{100, 1000, 10000, 100000, 200000}

func TestScalingTiers(t *testing.T) {
	cases := []struct {
		maxConnections int
		cpu, mem       string
	}{
		{100, "2", "3410Mi"},
		{1000, "2", "6898Mi"},
		{10000, "4", "12435Mi"},
		{100000, "8", "30925Mi"},
		{200000, "12", "52581Mi"},
	}
	if len(cases) != len(scalingTiers) {
		t.Fatalf("scalingTiers has %d entries, this test pins %d", len(scalingTiers), len(cases))
	}
	for _, tc := range cases {
		got, ok := tierFor(tc.maxConnections)
		if !ok {
			t.Errorf("tierFor(%d) reported no tier", tc.maxConnections)
			continue
		}
		if got.cpu != tc.cpu || got.mem != tc.mem {
			t.Errorf("tierFor(%d) = {cpu:%q mem:%q}, want {cpu:%q mem:%q}",
				tc.maxConnections, got.cpu, got.mem, tc.cpu, tc.mem)
		}
	}
}

// TestTierForRejectsOffTierValues covers the deliberate absence of rounding: a
// value between tiers has no published sizing, so it resolves to nothing rather
// than to the neighbour above or below it.
func TestTierForRejectsOffTierValues(t *testing.T) {
	for _, v := range []int{0, 1, 99, 101, 500, 999, 5000, 50000, 150000, 200001, -100} {
		if _, ok := tierFor(v); ok {
			t.Errorf("tierFor(%d) resolved a tier; only %s are tiers", v, scalingTierList)
		}
	}
}

func TestScalingTierListMatchesTable(t *testing.T) {
	if len(tierOrder) != len(scalingTiers) {
		t.Fatalf("scalingTiers has %d entries, tierOrder has %d", len(scalingTiers), len(tierOrder))
	}
	parts := make([]string, len(tierOrder))
	for i, v := range tierOrder {
		if i > 0 && v <= tierOrder[i-1] {
			t.Fatalf("tierOrder is not ascending at index %d", i)
		}
		if _, ok := scalingTiers[v]; !ok {
			t.Errorf("tierOrder names %d, which is not in scalingTiers", v)
		}
		parts[i] = strconv.Itoa(v)
	}
	if want := strings.Join(parts, ", "); scalingTierList != want {
		t.Errorf("scalingTierList = %q, want %q", scalingTierList, want)
	}
}

// TestContainerMem pins the one rewrite between the two memory spellings this
// schema carries: Kubernetes' Mi against the bare m docker and podman accept.
func TestContainerMem(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		{"3410Mi", "3410m"},
		{"6898Mi", "6898m"},
		{"12435Mi", "12435m"},
		{"30925Mi", "30925m"},
		{"52581Mi", "52581m"},
		{"4Gi", "4g"},
		{"512m", "512m"}, // already a container value: untouched
		{"", ""},
	} {
		if got := containerMem(tc.in); got != tc.want {
			t.Errorf("containerMem(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
	// Every tier's rewrite must satisfy the validator the same value would face
	// coming from an env file -- otherwise the default itself would be rejected.
	for _, v := range tierOrder {
		mem := containerMem(scalingTiers[v].mem)
		if !containerMemRE.MatchString(mem) {
			t.Errorf("tier %d default mem %q is not accepted by containerMemRE", v, mem)
		}
	}
}

func TestApplyScalingTierDefaultsK8s(t *testing.T) {
	c := &Config{}
	c.Scaling.MaxConnections = 10000
	c.ApplyDefaults(K8s)

	if c.Scaling.CPU != "4" {
		t.Errorf("Scaling.CPU = %q, want 4", c.Scaling.CPU)
	}
	if c.K8s.MsgNode.Mem != "12435Mi" {
		t.Errorf("MsgNode.Mem = %q, want 12435Mi", c.K8s.MsgNode.Mem)
	}
	// The removed key stays empty so validateK8s can treat any value as user-set.
	if c.K8s.MsgNode.CPU != "" {
		t.Errorf("MsgNode.CPU = %q, want empty", c.K8s.MsgNode.CPU)
	}
}

// TestApplyScalingTierDefaultsMemOverride pins the asymmetry the whole change
// rests on: memory is a default, CPU is not.
func TestApplyScalingTierDefaultsMemOverride(t *testing.T) {
	c := &Config{}
	c.Scaling.MaxConnections = 200000
	c.K8s.MsgNode.Mem = "64Gi"
	c.ApplyDefaults(K8s)

	if c.K8s.MsgNode.Mem != "64Gi" {
		t.Errorf("MsgNode.Mem = %q, want the explicit 64Gi to survive", c.K8s.MsgNode.Mem)
	}
	if c.Scaling.CPU != "12" {
		t.Errorf("Scaling.CPU = %q, want the tier's 12 regardless of the mem override", c.Scaling.CPU)
	}

	// Same for a container block.
	d := &Config{}
	d.Scaling.MaxConnections = 100000
	d.Docker.Container.Mem = "24g"
	d.ApplyDefaults(Docker)
	if d.Docker.Container.Mem != "24g" {
		t.Errorf("Docker.Container.Mem = %q, want the explicit 24g to survive", d.Docker.Container.Mem)
	}
	if d.Scaling.CPU != "8" {
		t.Errorf("Scaling.CPU = %q, want 8", d.Scaling.CPU)
	}
}

// TestApplyScalingTierDefaultsContainerBlocks mirrors applyContainerDefaults'
// existing parity: both blocks are filled whichever container platform is active.
func TestApplyScalingTierDefaultsContainerBlocks(t *testing.T) {
	for _, p := range []Platform{Docker, Podman} {
		c := &Config{}
		c.Scaling.MaxConnections = 10000
		c.ApplyDefaults(p)
		if c.Docker.Container.Mem != "12435m" || c.Podman.Container.Mem != "12435m" {
			t.Errorf("%s: container mem = docker %q / podman %q, want 12435m for both",
				p, c.Docker.Container.Mem, c.Podman.Container.Mem)
		}
		if c.Scaling.CPU != "4" {
			t.Errorf("%s: Scaling.CPU = %q, want 4", p, c.Scaling.CPU)
		}
	}
}

// TestApplyScalingTierDefaultsOffTier pins the fail-safe: an unresolvable tier
// derives nothing rather than inventing a footprint, so validateScalingTier's
// error is what the operator sees instead of a plausible-looking artifact.
func TestApplyScalingTierDefaultsOffTier(t *testing.T) {
	c := &Config{}
	c.Scaling.MaxConnections = 12345
	c.ApplyDefaults(K8s)
	if c.Scaling.CPU != "" || c.K8s.MsgNode.Mem != "" {
		t.Errorf("off-tier derived cpu=%q mem=%q, want both empty", c.Scaling.CPU, c.K8s.MsgNode.Mem)
	}
	if err := c.Validate(K8s); err == nil || !strings.Contains(err.Error(), "scaling.maxConnections must be one of") {
		t.Errorf("expected the tier error, got: %v", err)
	}
}

func TestValidateScalingTierRejectsOffTier(t *testing.T) {
	for _, v := range []int{0, 500, 5000, 200001} {
		c := validK8sConfig()
		c.Scaling.MaxConnections = v
		err := c.Validate(K8s)
		if err == nil || !strings.Contains(err.Error(), "scaling.maxConnections must be one of") {
			t.Errorf("maxConnections %d: expected the tier error, got: %v", v, err)
		}
		if err != nil && !strings.Contains(err.Error(), scalingTierList) {
			t.Errorf("maxConnections %d: error should list the tiers, got: %v", v, err)
		}
		// Every platform reads this value, so every platform rejects it.
		for _, p := range []Platform{Docker, Podman} {
			cc := validContainerConfig(p, "yes")
			cc.Scaling.MaxConnections = v
			if err := cc.Validate(p); err == nil || !strings.Contains(err.Error(), "scaling.maxConnections must be one of") {
				t.Errorf("%s maxConnections %d: expected the tier error, got: %v", p, v, err)
			}
		}
	}
}

func TestValidateScalingTierAcceptsEveryTier(t *testing.T) {
	for _, v := range tierOrder {
		c := validK8sConfig()
		c.Scaling.MaxConnections = v
		c.ApplyDefaults(K8s) // re-derive: the fixture defaulted at 100
		if err := c.Validate(K8s); err != nil {
			t.Errorf("k8s tier %d rejected: %v", v, err)
		}
		for _, p := range []Platform{Docker, Podman} {
			cc := validContainerConfig(p, "yes")
			cc.Scaling.MaxConnections = v
			cc.ApplyDefaults(p)
			if err := cc.Validate(p); err != nil {
				t.Errorf("%s tier %d rejected: %v", p, v, err)
			}
		}
	}
}

// TestValidateK8sMsgNodeCPURemoved mirrors TestValidateDockerRunModeRemoved: the
// key still decodes, so the operator gets a reason rather than "field not found".
func TestValidateK8sMsgNodeCPURemoved(t *testing.T) {
	c := validK8sConfig()
	c.K8s.MsgNode.CPU = "4"
	err := c.Validate(K8s)
	if err == nil || !strings.Contains(err.Error(), "was removed") {
		t.Fatalf("expected the msgNode.cpu removal error, got: %v", err)
	}
	if !strings.Contains(err.Error(), "scaling.maxConnections") {
		t.Errorf("removal error should name scaling.maxConnections, got: %v", err)
	}
	if !strings.Contains(err.Error(), "kubernetes.msgNode.mem") {
		t.Errorf("removal error should say mem is unaffected, got: %v", err)
	}
}

// TestValidateMaxPoolRemoved covers the second key this change folded away:
// maxPool and maxSpoolUsageMB named one broker setting under two platform-
// specific names, which is exactly what the scaling block no longer has.
func TestValidateMaxPoolRemoved(t *testing.T) {
	for _, tc := range []struct {
		p   Platform
		cfg func() *Config
	}{
		{K8s, validK8sConfig},
		{Docker, func() *Config { return validContainerConfig(Docker, "yes") }},
		{Podman, func() *Config { return validContainerConfig(Podman, "yes") }},
	} {
		c := tc.cfg()
		c.Scaling.MaxPool = 10000
		err := c.Validate(tc.p)
		if err == nil || !strings.Contains(err.Error(), "was removed") {
			t.Fatalf("%s: expected the maxPool removal error, got: %v", tc.p, err)
		}
		if !strings.Contains(err.Error(), "scaling.maxSpoolUsageMB") {
			t.Errorf("%s: removal error should name the replacement, got: %v", tc.p, err)
		}
	}
	// Zero is unset, not a value: the sentinel must not fire on every config.
	if err := validK8sConfig().Validate(K8s); err != nil {
		t.Errorf("an unset maxPool must not trip the removal error: %v", err)
	}
}

func TestValidateContainerMem(t *testing.T) {
	for _, p := range []Platform{Docker, Podman} {
		// The likely mistake: the Kubernetes spelling copied across.
		c := validContainerConfig(p, "yes")
		setContainerMem(c, p, "3410Mi")
		err := c.Validate(p)
		if err == nil || !strings.Contains(err.Error(), ".container.mem") {
			t.Fatalf("%s: expected a container.mem format error, got: %v", p, err)
		}
		if !strings.Contains(err.Error(), "Mi/Gi") {
			t.Errorf("%s: the error should name the Mi/Gi trap, got: %v", p, err)
		}

		for _, bad := range []string{"6898", "6898mb", "6g b", "-1g", "1.5g", "lots"} {
			c := validContainerConfig(p, "yes")
			setContainerMem(c, p, bad)
			if err := c.Validate(p); err == nil {
				t.Errorf("%s: container.mem %q was accepted", p, bad)
			}
		}
		for _, good := range []string{"6898m", "512M", "2g", "4G", "1024k", "536870912b", ""} {
			c := validContainerConfig(p, "yes")
			setContainerMem(c, p, good)
			if err := c.Validate(p); err != nil {
				t.Errorf("%s: container.mem %q was rejected: %v", p, good, err)
			}
		}
	}
}

func setContainerMem(c *Config, p Platform, mem string) {
	if p == Podman {
		c.Podman.Container.Mem = mem
		return
	}
	c.Docker.Container.Mem = mem
}

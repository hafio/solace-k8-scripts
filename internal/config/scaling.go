package config

import (
	"fmt"
	"regexp"
	"strings"
)

// scalingTier is the compute footprint Solace publishes for one connection tier.
//
// CPU is a property of the tier, not a knob: sizing a broker by connection count
// and then sizing its CPU independently is how a 200k-connection broker ends up
// on 2 cores, so the env file states the tier and this table states the cores.
// Mem is only the tier's *default* -- kubernetes.msgNode.mem and the container blocks'
// mem both still override it, because memory headroom depends on the message
// mix in a way core count does not.
type scalingTier struct {
	cpu string // messagingNodeCpu (k8s), cpus:/--cpus (container); never settable
	mem string // Kubernetes quantity; containerMem rewrites it for the engines
}

// scalingTiers is the authoritative maxConnections -> footprint table, and
// doubles as the enum validateScalingTier checks against: these five are the
// only connection counts Solace publishes sizing for. A value between tiers is
// deliberately refused rather than rounded up -- rounding would silently
// provision cores the operator never asked for, and rounding down would
// under-size a broker that had already declared its load.
var scalingTiers = map[int]scalingTier{
	100:    {cpu: "2", mem: "3410Mi"},
	1000:   {cpu: "2", mem: "6898Mi"},
	10000:  {cpu: "4", mem: "12435Mi"},
	100000: {cpu: "8", mem: "30925Mi"},
	200000: {cpu: "12", mem: "52581Mi"},
}

// scalingTierList names scalingTiers' keys in ascending order for the error
// message. It is a literal for the same reason accessLevelList is: the message
// has to be stable, and this package avoids pulling in sort for a handful of
// items (see sortStrings). TestScalingTierListMatchesTable pins the two
// together so the list cannot drift from the table.
const scalingTierList = "100, 1000, 10000, 100000, 200000"

// tierFor looks up the footprint for maxConnections. ok is false for anything
// that is not a tier -- including the zero value, which is what a Config built
// in code rather than through Load carries. validateScalingTier is what turns
// that into a loud failure; this stays total so applyScalingTierDefaults (which
// necessarily runs before Validate) never has to guess.
func tierFor(maxConnections int) (scalingTier, bool) {
	t, ok := scalingTiers[maxConnections]
	return t, ok
}

// containerMemRE is docker's and podman's own memory syntax: an integer then one
// of b/k/m/g. It exists because this schema sits a Kubernetes quantity
// (kubernetes.msgNode.mem, "3410Mi") next to a container one that rejects that exact
// spelling, so the likeliest mistake is copying the k8s form across.
var containerMemRE = regexp.MustCompile(`(?i)^[0-9]+[bkmg]$`)

// containerMem rewrites a tier's Kubernetes quantity into the suffix docker's
// mem_limit and podman's Memory= accept. Both conventions count in binary units
// -- Kubernetes spells the mebibyte "Mi", the engines spell it "m" -- so this
// only ever drops a trailing "i", never rescales. It is not a general converter:
// it sees only the five fixed strings above, and anything without the trailing
// "i" passes through untouched.
func containerMem(k8sMem string) string {
	if strings.HasSuffix(k8sMem, "i") {
		return strings.ToLower(strings.TrimSuffix(k8sMem, "i"))
	}
	return k8sMem
}

// applyScalingTierDefaults derives the tier-fixed CPU and the tier-defaulted
// memory. ApplyDefaults calls it *after* the platform branches, which is the
// whole point: maxConnections only reaches its final value in those branches
// (they default it to 100 on k8s and 1000 on containers), so a tier lookup any
// earlier would read a zero.
//
// CPU is assigned unconditionally -- it is derived, so there is no user value to
// preserve. Memory keeps setDefault semantics because both mem keys override it.
//
// An out-of-tier maxConnections leaves both alone: validateScalingTier rejects
// it moments later, and inventing a footprint for a tier that does not exist
// would bury that error under a plausible-looking artifact.
func (c *Config) applyScalingTierDefaults(p Platform) {
	t, ok := tierFor(c.Scaling.MaxConnections)
	if !ok {
		return
	}
	c.Scaling.CPU = t.cpu
	if p == K8s {
		setDefault(&c.K8s.MsgNode.Mem, t.mem)
	}
	if p.IsContainer() {
		// Both blocks are filled whichever container platform is active, matching
		// applyContainerDefaults' existing parity for name/shmSize/ulimits.
		mem := containerMem(t.mem)
		setDefault(&c.Docker.Container.Mem, mem)
		setDefault(&c.Podman.Container.Mem, mem)
	}
}

// validateScaling checks the scaling block, which is platform-independent: every
// knob reaches every platform, so this runs ahead of the platform switch in
// Validate rather than inside either platform validator.
func (c *Config) validateScaling() error {
	if _, ok := tierFor(c.Scaling.MaxConnections); !ok {
		return fmt.Errorf("scaling.maxConnections must be one of the supported scaling tiers %s (got: %d); "+
			"broker CPU is fixed by the tier, so a value between tiers has no published sizing to apply",
			scalingTierList, c.Scaling.MaxConnections)
	}
	if c.Scaling.MaxPool != 0 {
		// Removed rather than aliased: silently forwarding it would leave two keys
		// for one setting, and an env file setting both would have no defined
		// winner.
		return fmt.Errorf("scaling.maxPool was removed; it named the same broker setting as "+
			"scaling.maxSpoolUsageMB, which now feeds both the k8s CR (spec.systemScaling.maxSpoolUsage) "+
			"and the container environment (messagespool_maxspoolusage) -- "+
			"rename it to scaling.maxSpoolUsageMB (got: %d)", c.Scaling.MaxPool)
	}
	return nil
}

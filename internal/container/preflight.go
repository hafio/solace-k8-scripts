package container

import (
	"context"
	"fmt"

	"solace/internal/config"
)

// Preflight is the read-only probe every mutating operation runs first. `<runtime>
// info` is the daemon-style equivalent of the cluster's `auth can-i`: it answers
// with one round-trip whether this process can actually talk to the engine, which
// on docker means the daemon is up and this user is in the right group, and on
// rootless podman means the user session and its subuid mapping exist.
//
// It runs before the first artifact is written, not after. A deploy that renders a
// compose file or installs a quadlet unit and only then discovers a stopped daemon
// leaves host state the operator now has to reason about; failing first keeps
// "nothing happened" true.
//
// There is deliberately no skip flag -- the only legitimate reason to skip it is
// previewing without an engine, which is --dry-run and takes the branch below.
// It never starts the daemon or logs anyone in: both are the operator's decisions,
// made with privileges this tool should not be exercising on their behalf.
func (m *Manager) Preflight(ctx context.Context) error {
	rt, err := m.runtime()
	if err != nil {
		return err
	}
	if m.isDryRun() {
		// Echo the probe so the preview shows it, then skip the assertion: the
		// Echo runner answers nothing, and there is no engine to answer.
		if _, err := m.R.Output(ctx, rt.Name(), rt.Args("info")...); err != nil {
			return err
		}
		fmt.Fprintln(m.out(), "  engine         : skipped (dry-run)")
		return nil
	}
	if _, err := m.R.Output(ctx, rt.Name(), rt.Args("info")...); err != nil {
		// The engine's own message already went to stderr; add the one line that
		// says what to do about it, which differs by platform and rootless mode.
		return fmt.Errorf("cannot talk to the %s engine (%q info failed): %w\n  %s",
			platformTitle(m.P), rt, err, m.engineHint())
	}
	return nil
}

// engineHint is the actionable half of a failed Preflight: the one thing to try
// next. Rootless podman is called out separately because its usual failure is a
// missing user session rather than a stopped service, and `sudo systemctl start`
// is the wrong answer there -- it would start the rootful engine the deploy is not
// using.
func (m *Manager) engineHint() string {
	if m.P == config.Podman {
		if m.Cfg.Podman.Rootless {
			return "start the rootless user service: `systemctl --user start podman.socket` " +
				"(and `loginctl enable-linger $USER` so it survives logout) -- do NOT use sudo, " +
				"podman.rootless=true deploys as this user"
		}
		return "start the engine: `sudo systemctl start podman.socket`, and re-run with the privileges " +
			"rootful podman needs"
	}
	return "start the daemon: `sudo systemctl start docker`, and check this user is in the `docker` group " +
		"(`newgrp docker` applies a fresh membership without logging out)"
}

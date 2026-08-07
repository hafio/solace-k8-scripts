module solace

go 1.26

// Pinned so builds use a patched stdlib. `go 1.26` alone is only a floor, so
// GOTOOLCHAIN=auto selects go1.26.0 on a machine with an older Go -- which is
// what CI has, and what shipped a net panic bug into release binaries. Bump
// this whenever `scan` reports a standard-library vulnerability.
toolchain go1.26.5

require (
	github.com/spf13/cobra v1.8.1
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	golang.org/x/mod v0.38.0 // indirect
	golang.org/x/sync v0.22.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/telemetry v0.0.0-20260708182218-49f421fb7959 // indirect
	golang.org/x/tools v0.48.0 // indirect
	golang.org/x/vuln v1.6.0 // indirect
)

tool golang.org/x/vuln/cmd/govulncheck

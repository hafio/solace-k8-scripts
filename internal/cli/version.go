package cli

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

// version is set at link time by the dev scripts:
//
//	-ldflags "-X solace/internal/cli.version=$(git describe --tags --dirty --always)"
//
// On a tag push (tag.yml) HEAD is exactly the pushed tag, so a release binary
// prints e.g. "v1.2.3" -- the same string as the GitHub release. A dev-script
// build between tags gets a pseudo-version like "v1.2.3-5-gabc1234-dirty"; a
// plain `go build .` or `go test` leaves it at "dev". Link-time-set and never
// mutated at runtime (tests aside): -X can only overwrite a package-level
// string var, so a var is the only shape the linker allows here -- the one
// deliberate exception to threading state through App.
var version = "dev"

// newVersionCmd builds `solace-util version`. A bare literal like
// newCompletionCmd, not leaf: it needs no *App, loads no env file, and takes
// no flags. Deliberately a subcommand rather than cobra's Version field --
// the implicit --version flag only appears inside Execute, so it would never
// reach the generated reference, and -v already means --verbose.
func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the solace-util version",
		Long: "Print the version this binary was built at, plus the Go toolchain and\n" +
			"platform that built it -- useful to paste alongside a support request.\n\n" +
			"A release binary (built by scripts/dev.sh or dev.ps1) reports the git tag\n" +
			"it shipped as, e.g. v1.2.3 -- matching the GitHub release exactly. A plain\n" +
			"`go build .` with no version stamped reports \"dev\".",
		Args:              cobra.NoArgs,
		ValidArgsFunction: cobra.NoFileCompletions,
		RunE: func(*cobra.Command, []string) error {
			return emit([]byte(fmt.Sprintf("solace-util %s %s %s/%s\n",
				version, runtime.Version(), runtime.GOOS, runtime.GOARCH)))
		},
	}
}

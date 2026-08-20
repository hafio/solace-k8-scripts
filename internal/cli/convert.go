package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"solace/internal/config"
	"solace/internal/convert"
)

// newConvertCmd builds `solace-util convert`, the migration aid from the pre-Go bash
// env format to the YAML env file every other command reads. It sits at the root
// rather than under a platform because it loads no config of its own: the file
// it reads is the argument, not -e/--env, so the app context stays unused.
func newConvertCmd(app *App) *cobra.Command {
	var (
		out   string
		force bool
	)
	cmd := &cobra.Command{
		Use:   "convert <bash-env-file>",
		Short: "Convert a legacy bash env file into a YAML env file",
		Long: "Convert a legacy bash env file -- the pre-Go format sourced by bash/000-env.sh --\n" +
			"into the YAML env file this CLI reads.\n\n" +
			"The target platform section is detected from the variables present; pass\n" +
			"--platform to choose it yourself. Variables with no YAML equivalent are\n" +
			"reported on stderr rather than dropped silently.\n\n" +
			"The output carries every secret from the source file verbatim, so treat it\n" +
			"like the source: write it with -o rather than through a shared terminal, and\n" +
			"never commit it.\n\n" +
			"  solace-util convert bash/env/prod -o prod.yaml\n" +
			"  solace-util convert bash/env/prod --platform podman -o prod.yaml\n" +
			"  solace-util check deploy -e prod.yaml",
		Args:          cobra.ExactArgs(1),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(_ *cobra.Command, args []string) error {
			return runConvert(args[0], out, app.PlatformFlag, force)
		},
	}
	cmd.Flags().StringVarP(&out, "out", "o", "", "write the YAML here instead of stdout")
	cmd.Flags().BoolVar(&force, "force", false, "overwrite the --out file if it already exists")
	return cmd
}

// runConvert reads --platform through the same parser the rest of the tree uses,
// so one word means one thing everywhere: it names a platform, in canonical or
// abbreviated form. What it SELECTS still differs by necessity -- here it is the
// section to write into a new file, elsewhere the section to read from an
// existing one -- but there is no second spelling to learn. Empty still means
// detect, which for a bash source is a question about its variable names
// (internal/convert), not about YAML sections.
func runConvert(src, out, platform string, force bool) error {
	p, err := config.ParsePlatform(platform)
	if err != nil {
		return err
	}
	raw, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("read bash env file %q: %w", src, err)
	}
	res, err := convert.Convert(raw, src, p)
	if err != nil {
		return err
	}
	for _, w := range res.Warnings {
		warn("%s", w)
	}
	if out == "" {
		step("converted %s for platform %s", src, res.Platform)
		return emit(res.YAML)
	}
	if _, err := os.Stat(out); err == nil && !force {
		return fmt.Errorf("refusing to overwrite %q: pass --force to replace it, or choose another --out path", out)
	}
	// 0o600: the converted file carries the same secrets as the source.
	if err := os.WriteFile(out, res.YAML, 0o600); err != nil {
		return fmt.Errorf("write %q: %w", out, err)
	}
	step("converted %s -> %s (platform %s)", src, out, res.Platform)
	step("review it before use; it carries the secrets from %s verbatim", src)
	return nil
}

package cli

import (
	"github.com/spf13/cobra"
)

func NewRootCommand(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ksrc",
		Short: "Gradle dependency source search",
		Long: "Gradle dependency source search CLI.\nStart by running `ksrc search` to locate snippets, rg-style, then run `ksrc cat` to read files using the returned file id.\n\n" +
			"Guidance & caveats:\n" +
			"  - Prefer --module/--artifact to reduce noise; use --subproject in monorepos.\n" +
			"  - If no matches in composite builds, try running from the included build root or set --project.\n" +
			"  - Use --scope for build-time deps; use --targets to limit KMP resolution.\n" +
			"  - --offline uses cached sources only; --refresh forces downloads.\n\n" +
			"Common issues:\n" +
			"  - E_NO_SOURCES: try ksrc deps, ksrc fetch <coord>, or set --project/--scope.\n" +
			"  - Gradle not found: run in a Gradle project or set --project to the root.\n" +
			"  - Unsupported class version: fix Gradle <-> JDK mismatch (JAVA_HOME).\n\n" +
			"File-id format:\n" +
			"  group:artifact:version!/path/inside/jar.kt",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	cmd.PersistentFlags().BoolVarP(&app.Verbose, "verbose", "v", false, "show verbose output (including Gradle failures)")
	cmd.Version = versionString()
	cmd.SetVersionTemplate("{{.Version}}\n")

	cmd.AddCommand(newSearchCmd(app))
	cmd.AddCommand(newCatCmd(app))
	cmd.AddCommand(newOpenCmd(app))
	cmd.AddCommand(newDepsCmd(app))
	cmd.AddCommand(newResolveCmd(app))
	cmd.AddCommand(newFetchCmd(app))
	cmd.AddCommand(newWhereCmd(app))
	cmd.AddCommand(newDoctorCmd(app))

	return cmd
}

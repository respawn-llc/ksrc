package cli

import (
	"github.com/spf13/cobra"
)

func NewRootCommand(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ksrc",
		Short: "Kotlin dependency source search",
		Long: "Kotlin dependency source search.\n\n" +
			"Common recipes:\n" +
			"  Android: ksrc search \"symbol\" --config \"*debugCompileClasspath\" --module \"group:artifact\"\n" +
			"  KMP:     ksrc search \"symbol\" --targets jvm --module \"group:artifact\"\n" +
			"  JVM:     ksrc search \"symbol\" --scope compile --module \"group:artifact\"\n\n" +
			"If E_NO_SOURCES: try --project <root>, --config \"*debugCompileClasspath\", or --subproject :module.",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	cmd.PersistentFlags().BoolVarP(&app.Verbose, "verbose", "v", false, "show verbose output (including Gradle failures)")

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

package cli

import (
	"context"
	"fmt"

	"github.com/respawn-app/ksrc/internal/adapter"
	"github.com/respawn-app/ksrc/internal/resolve"
	"github.com/spf13/cobra"
)

func newFetchCmd(app *App) *cobra.Command {
	var flags ResolveFlags

	cmd := &cobra.Command{
		Use:   "fetch <coord>",
		Short: "Ensure sources for a coordinate exist in Gradle caches",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			coord, err := resolve.ParseCoord(args[0])
			if err != nil {
				return err
			}
			if coord.Version == "" {
				return fmt.Errorf("version required for fetch. Use group:artifact:version.")
			}
			flags.Module = coord.String()
			flags.Version = coord.Version

			sources, _, meta, err := resolveSources(context.Background(), app, flags, coord.String(), false, false)
			if err != nil {
				return err
			}
			emitDiagnostics(cmd, meta, app.Verbose)
			if len(sources) == 0 {
				return noSourcesErr(flags, joinHints("Try: verify the coordinate exists in the project or run ksrc deps to see resolved coords.", projectHint(flags, meta)))
			}
			return adapter.WriteCoordMatches(cmd.OutOrStdout(), sources, coord)
		},
	}

	cmd.Flags().StringVar(&flags.Project, "project", ".", "project root")
	cmd.Flags().BoolVar(&flags.Offline, "offline", false, "offline mode")
	cmd.Flags().BoolVar(&flags.Refresh, "refresh", false, "refresh dependencies")
	cmd.Flags().BoolVar(&flags.IncludeBuildSrc, "buildsrc", true, "include buildSrc dependencies (set --buildsrc=false to disable)")
	cmd.Flags().BoolVar(&flags.IncludeBuildscript, "buildscript", true, "include buildscript classpath dependencies (set --buildscript=false to disable)")
	cmd.Flags().BoolVar(&flags.IncludeIncludedBuilds, "include-builds", true, "include composite builds (includeBuild) (set --include-builds=false to disable)")

	return cmd
}

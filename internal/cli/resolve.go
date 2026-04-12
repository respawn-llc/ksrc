package cli

import (
	"context"

	"github.com/respawn-app/ksrc/internal/adapter"
	"github.com/spf13/cobra"
)

func newResolveCmd(app *App) *cobra.Command {
	var flags ResolveFlags

	cmd := &cobra.Command{
		Use:   "resolve",
		Short: "Resolve dependency sources",
		RunE: func(cmd *cobra.Command, args []string) error {
			sources, _, meta, err := resolveSources(context.Background(), app, flags, "", true, true)
			if err != nil {
				return err
			}
			emitDiagnostics(cmd, meta, app.Verbose)
			if len(sources) == 0 {
				return noSourcesErr(flags, noSourcesHintForFlags(flags, meta))
			}
			return adapter.WriteCoordPaths(cmd.OutOrStdout(), sources)
		},
	}

	cmd.Flags().StringVar(&flags.Project, "project", ".", "project root")
	cmd.Flags().StringVar(&flags.Module, "module", "", "module selector (group:artifact[:version])")
	cmd.Flags().StringVar(&flags.Group, "group", "", "group filter")
	cmd.Flags().StringVar(&flags.Artifact, "artifact", "", "artifact filter")
	cmd.Flags().StringVar(&flags.Version, "version", "", "version filter")
	cmd.Flags().StringVar(&flags.Scope, "scope", "compile", "dependency scope (compile|runtime|test|all)")
	cmd.Flags().StringVar(&flags.Config, "config", "", "configuration name(s) or glob patterns (comma-separated)")
	cmd.Flags().StringVar(&flags.Targets, "targets", "", "KMP targets (comma-separated)")
	cmd.Flags().StringSliceVar(&flags.Subprojects, "subproject", nil, "limit to subproject (repeatable)")
	cmd.Flags().BoolVar(&flags.Offline, "offline", false, "offline mode")
	cmd.Flags().BoolVar(&flags.Refresh, "refresh", false, "refresh dependencies")
	cmd.Flags().BoolVar(&flags.IncludeBuildSrc, "buildsrc", true, "include buildSrc dependencies (set --buildsrc=false to disable)")
	cmd.Flags().BoolVar(&flags.IncludeBuildscript, "buildscript", true, "include buildscript classpath dependencies (set --buildscript=false to disable)")
	cmd.Flags().BoolVar(&flags.IncludeIncludedBuilds, "include-builds", true, "include composite builds (includeBuild) (set --include-builds=false to disable)")

	return cmd
}

package cli

import (
	"context"

	"github.com/respawn-app/ksrc/internal/adapter"
	"github.com/spf13/cobra"
)

func newDepsCmd(app *App) *cobra.Command {
	var flags ResolveFlags

	cmd := &cobra.Command{
		Use:   "deps",
		Short: "List resolved dependencies and source availability",
		RunE: func(cmd *cobra.Command, args []string) error {
			applyFilters := hasSelector(flags)
			sources, deps, meta, err := resolveSources(context.Background(), app, flags, "", applyFilters, false)
			if err != nil {
				return err
			}
			emitDiagnostics(cmd, meta, app.Verbose)
			return adapter.WriteDepsOutput(cmd.OutOrStdout(), sources, deps)
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
	cmd.Flags().StringVar(&flags.GradleUserHome, "gradle-user-home", "", "Gradle user home (default: GRADLE_USER_HOME or ~/.gradle)")
	BindAndroidHome(cmd, &flags.AndroidHome)
	cmd.Flags().BoolVar(&flags.IncludeBuildSrc, "buildsrc", true, "include buildSrc dependencies (set --buildsrc=false to disable)")
	cmd.Flags().BoolVar(&flags.IncludeBuildscript, "buildscript", true, "include buildscript classpath dependencies (set --buildscript=false to disable)")
	cmd.Flags().BoolVar(&flags.IncludeIncludedBuilds, "include-builds", true, "include composite builds (includeBuild) (set --include-builds=false to disable)")

	return cmd
}

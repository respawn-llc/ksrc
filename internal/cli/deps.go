package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

func newDepsCmd(app *App) *cobra.Command {
	var flags ResolveFlags

	cmd := &cobra.Command{
		Use:   "deps",
		Short: "List resolved dependencies and source availability",
		RunE: func(cmd *cobra.Command, args []string) error {
			sources, deps, meta, err := resolveSources(context.Background(), app, flags, "", false, false)
			if err != nil {
				return err
			}
			emitDiagnostics(cmd, meta, app.Verbose)
			sourceByCoord := make(map[string]string)
			for _, s := range sources {
				sourceByCoord[s.Coord.String()] = s.Path
			}

			seen := make(map[string]struct{})
			for _, d := range deps {
				key := d.String()
				if _, ok := seen[key]; ok {
					continue
				}
				seen[key] = struct{}{}
				path := sourceByCoord[key]
				sourcesYes := "no"
				if path != "" {
					sourcesYes = "yes"
				}
				fmt.Fprintf(cmd.OutOrStdout(), "%s  [sources: %s]  [path: %s]\n", key, sourcesYes, path)
			}

			if len(deps) == 0 {
				for _, s := range sources {
					key := s.Coord.String()
					if _, ok := seen[key]; ok {
						continue
					}
					seen[key] = struct{}{}
					fmt.Fprintf(cmd.OutOrStdout(), "%s  [sources: yes]  [path: %s]\n", key, s.Path)
				}
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&flags.Project, "project", ".", "project root")
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

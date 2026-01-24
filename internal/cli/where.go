package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/respawn-app/ksrc/internal/resolve"
	"github.com/spf13/cobra"
)

func newWhereCmd(app *App) *cobra.Command {
	var flags ResolveFlags

	cmd := &cobra.Command{
		Use:   "where <path|coord>",
		Short: "Locate cached source artifact or file",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			arg := strings.TrimSpace(args[0])
			if strings.Contains(arg, "!/") {
				coord, inner, err := resolve.ParseFileID(arg)
				if err != nil {
					return err
				}
				flags.Module = coord.String()
				flags.Version = coord.Version
				sources, _, _, err := resolveSources(context.Background(), app, flags, coord.String(), true, true)
				if err != nil {
					return err
				}
				jarPath, err := findJarByCoord(sources, coord)
				if err != nil {
					return err
				}
				fmt.Fprintf(cmd.OutOrStdout(), "%s|%s\n", coord.String()+"!/"+inner, jarPath)
				return nil
			}

			if coord, err := resolve.ParseCoord(arg); err == nil {
				flags.Module = coord.String()
				flags.Version = coord.Version
				dep := ""
				if coord.Version != "" {
					dep = coord.String()
				}
				sources, _, meta, err := resolveSources(context.Background(), app, flags, dep, true, true)
				if err != nil {
					return err
				}
				emitDiagnostics(cmd, meta, app.Verbose)
				if len(sources) == 0 {
					return noSourcesErr(flags, noSourcesHintForFlags(flags, meta))
				}
				jarPath, err := findJarByCoord(sources, coord)
				if err != nil {
					return err
				}
				fmt.Fprintf(cmd.OutOrStdout(), "%s|%s\n", coord.String(), jarPath)
				return nil
			}

			if flags.Module == "" && flags.Group == "" && flags.Artifact == "" {
				return fmt.Errorf("path requires --module or a file-id")
			}

			sources, _, meta, err := resolveSources(context.Background(), app, flags, "", true, true)
			if err != nil {
				return err
			}
			emitDiagnostics(cmd, meta, app.Verbose)
			if len(sources) == 0 {
				return noSourcesErr(flags, noSourcesHintForFlags(flags, meta))
			}
			jarPath, inner, err := findFileInJars(sources, arg)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%s|%s\n", flags.Module+"!/"+inner, jarPath)
			return nil
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

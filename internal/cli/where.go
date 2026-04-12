package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/respawn-app/ksrc/internal/adapter"
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
				location := adapter.FileLocation{}
				found := false
				if !hasExplicitFollowupResolutionContext(cmd) {
					var err error
					location, found, err = adapter.FindFollowupFileIDLocation(arg)
					if err != nil {
						return err
					}
				}
				if found {
					return adapter.WriteFileLocation(cmd.OutOrStdout(), location)
				}
				coord, _, err := resolve.ParseFileID(arg)
				if err != nil {
					return err
				}
				flags.Module = coord.String()
				flags.Version = coord.Version
				sources, _, _, err := resolveSources(context.Background(), app, flags, coord.String(), true, true)
				if err != nil {
					return err
				}
				location, err = adapter.ResolveFileIDLocation(sources, arg, adapter.NoSourcesHintForCoord(coord))
				if err != nil {
					return err
				}
				adapter.TryTrackFileLocation(location)
				return adapter.WriteFileLocation(cmd.OutOrStdout(), location)
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
				source, err := adapter.ResolveCoordSource(sources, coord, adapter.NoSourcesHintForCoord(coord))
				if err != nil {
					return err
				}
				return adapter.WriteCoordPath(cmd.OutOrStdout(), coord, source.Path)
			}

			if flags.Module == "" && flags.Group == "" && flags.Artifact == "" {
				return fmt.Errorf("path requires --module, or --group plus --artifact, or a file-id")
			}

			sources, _, meta, err := resolveSources(context.Background(), app, flags, "", true, true)
			if err != nil {
				return err
			}
			emitDiagnostics(cmd, meta, app.Verbose)
			if len(sources) == 0 {
				return noSourcesErr(flags, noSourcesHintForFlags(flags, meta))
			}
			location, err := adapter.ResolvePathLocation(sources, arg, "Try: ksrc search \"<pattern>\" --module group:artifact to get a file-id")
			if err != nil {
				return err
			}
			adapter.TryTrackFileLocation(location)
			return adapter.WriteFileLocation(cmd.OutOrStdout(), location)
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

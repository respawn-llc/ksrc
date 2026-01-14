package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/respawn-app/ksrc/internal/cat"
	"github.com/respawn-app/ksrc/internal/resolve"
	"github.com/spf13/cobra"
)

func newCatCmd(app *App) *cobra.Command {
	var flags ResolveFlags
	var lines string

	cmd := &cobra.Command{
		Use:   "cat <file-id|path>",
		Short: "Print file contents from dependency sources",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			arg := strings.TrimSpace(args[0])
			lr, err := cat.ParseLineRange(lines)
			if err != nil {
				return err
			}

			if strings.Contains(arg, "!/") {
				coord, inner, err := resolve.ParseFileID(arg)
				if err != nil {
					return err
				}
				flags.Module = coord.String()
				flags.Version = coord.Version

				sources, _, _, err := resolveSources(context.Background(), app, flags, "", true, false)
				if err != nil {
					return err
				}
				if len(sources) == 0 {
					return noSourcesErr(flags, noSourcesHintForCoord(coord))
				}
				jarPath, err := findJarByCoord(sources, coord)
				if err != nil {
					return err
				}
				data, err := cat.ReadFileFromZip(jarPath, inner, lr)
				if err != nil {
					return err
				}
				_, err = cmd.OutOrStdout().Write(data)
				return err
			}

			if flags.Module == "" && flags.Group == "" && flags.Artifact == "" {
				return fmt.Errorf("path requires --module or a file-id. Try: ksrc cat <file-id> or ksrc cat --module group:artifact[:version] <path>")
			}

			sources, _, meta, err := resolveSources(context.Background(), app, flags, "", true, true)
			if err != nil {
				return err
			}
			emitWarnings(cmd, meta)
			if len(sources) == 0 {
				return noSourcesErr(flags, noSourcesHintForFlags(flags, meta))
			}
			jarPath, inner, err := findFileInJars(sources, arg)
			if err != nil {
				return err
			}
			data, err := cat.ReadFileFromZip(jarPath, inner, lr)
			if err != nil {
				return err
			}
			_, err = cmd.OutOrStdout().Write(data)
			return err
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
	cmd.Flags().StringVar(&lines, "lines", "", "line range (start,end)")

	return cmd
}

func findJarByCoord(sources []resolve.SourceJar, coord resolve.Coord) (string, error) {
	for _, s := range sources {
		if s.Coord.Group == coord.Group && s.Coord.Artifact == coord.Artifact && s.Coord.Version == coord.Version {
			return s.Path, nil
		}
	}
	return "", fmt.Errorf("source jar not found for %s. Try: ksrc fetch %s", coord.String(), coord.String())
}

func findFileInJars(sources []resolve.SourceJar, inner string) (string, string, error) {
	inner = strings.TrimPrefix(inner, "/")
	for _, s := range sources {
		data, err := cat.ReadFileFromZip(s.Path, inner, nil)
		if err == nil && len(data) > 0 {
			return s.Path, inner, nil
		}
	}
	return "", "", fmt.Errorf("file not found in resolved sources: %s. Try: ksrc search \"<pattern>\" --module group:artifact to get a file-id", inner)
}

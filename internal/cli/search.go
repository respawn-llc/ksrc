package cli

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/respawn-app/ksrc/internal/adapter"
	"github.com/respawn-app/ksrc/internal/search"
	"github.com/spf13/cobra"
)

func newSearchCmd(app *App) *cobra.Command {
	var flags ResolveFlags
	var rgArgs string
	var showExtractedPath bool
	var contextLines int

	cmd := &cobra.Command{
		Use:     "search <pattern> [-- <rg-args>]",
		Aliases: []string{"rg"},
		Short:   "Search dependency sources",
		Args: func(cmd *cobra.Command, args []string) error {
			dash := cmd.Flags().ArgsLenAtDash()
			if dash == -1 {
				return cobra.ExactArgs(1)(cmd, args)
			}
			if dash == 0 {
				if len(args) < 1 {
					return fmt.Errorf("pattern is required")
				}
				return nil
			}
			if dash > 1 {
				return fmt.Errorf("expected a single <pattern> before --")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			dash := cmd.Flags().ArgsLenAtDash()
			var pattern string
			var passArgs []string
			if dash >= 0 {
				if dash > len(args) {
					dash = len(args)
				}
				if dash == 0 {
					pattern = args[0]
					if len(args) > 1 {
						passArgs = append(passArgs, args[1:]...)
					}
				} else {
					pattern = args[0]
					passArgs = append(passArgs, args[dash:]...)
				}
			} else if len(args) > 0 {
				pattern = args[0]
			}
			if strings.TrimSpace(pattern) == "" {
				return fmt.Errorf("pattern is required. Try: ksrc search \"<pattern>\"")
			}
			if !flags.All && !hasSelector(flags) {
				flags.All = true
			}
			ctx := context.Background()
			sources, _, meta, err := resolveSources(ctx, app, flags, "", true, true)
			if err != nil {
				return err
			}
			emitDiagnostics(cmd, meta, app.Verbose)
			if len(sources) == 0 {
				return noSourcesErr(flags, noSourcesHintForFlags(flags, meta))
			}
			rgExtra := adapter.SplitCSV(rgArgs)
			if contextLines > 0 {
				rgExtra = append(rgExtra, "-C", strconv.Itoa(contextLines))
			}
			rgExtra = append(rgExtra, passArgs...)
			var report func(search.ExecPlan)
			if app.Verbose {
				report = func(plan search.ExecPlan) {
					_ = adapter.WriteRGCommandReport(cmd.ErrOrStderr(), plan)
				}
			}
			matches, err := search.Run(ctx, app.Runner, search.Options{
				Pattern: pattern,
				Jars:    sources,
				RGArgs:  rgExtra,
				WorkDir: flags.Project,
				Report:  report,
			})
			if err != nil {
				return err
			}
			adapter.TryTrackSearchMatches(matches)
			return adapter.WriteSearchMatches(cmd.OutOrStdout(), matches, showExtractedPath)
		},
	}

	cmd.Flags().StringVar(&flags.Project, "project", ".", "project root")
	cmd.Flags().BoolVar(&flags.All, "all", false, "search all resolved dependencies (default when no module/group/artifact/version is set)")
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
	cmd.Flags().StringVar(&rgArgs, "rg-args", "", "extra args for rg (comma-separated)")
	cmd.Flags().BoolVar(&showExtractedPath, "show-extracted-path", false, "include temp extracted path in output")
	cmd.Flags().IntVar(&contextLines, "context", 0, "show N lines before/after matches (rg -C)")

	return cmd
}

func hasSelector(flags ResolveFlags) bool {
	return strings.TrimSpace(flags.Module) != "" ||
		strings.TrimSpace(flags.Group) != "" ||
		strings.TrimSpace(flags.Artifact) != "" ||
		strings.TrimSpace(flags.Version) != ""
}

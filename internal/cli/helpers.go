package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/respawn-app/ksrc/internal/resolution"
	"github.com/respawn-app/ksrc/internal/resolve"
)

func resolveSources(ctx context.Context, app *App, flags ResolveFlags, dep string, applyFilters bool, allowCacheFallback bool) ([]resolve.SourceJar, []resolve.Coord, resolution.ResolveMeta, error) {
	service := resolution.Service{Runner: app.Runner, Verbose: app.Verbose}
	req := flags.ToRequest(dep, applyFilters, allowCacheFallback)
	result, err := service.ResolveSources(ctx, req)
	return result.Sources, result.Deps, result.Meta, err
}

func noSourcesErr(flags ResolveFlags, hint string) error {
	msg := "E_NO_SOURCES: no sources resolved."
	var parts []string
	if flags.Offline {
		parts = append(parts, "You ran with --offline; rerun without it to allow downloads.")
	}
	if strings.TrimSpace(hint) != "" {
		parts = append(parts, hint)
	}
	if len(parts) == 0 {
		return errors.New(strings.TrimSuffix(msg, "."))
	}
	return errors.New(msg + " " + strings.Join(parts, " "))
}

func noSourcesHintForFlags(flags ResolveFlags, meta resolution.ResolveMeta) string {
	if flags.All {
		return joinHints(
			"Try: ksrc deps (list resolved coords), then ksrc fetch <coord> to download sources.",
			projectHint(flags, meta),
		)
	}
	if coord, ok := resolve.SelectorToCoord(flags.Module, flags.Group, flags.Artifact, flags.Version); ok {
		if coord.Version != "" {
			return joinHints(
				fmt.Sprintf("Try: ksrc fetch %s to download sources.", coord.String()),
				projectHint(flags, meta),
			)
		}
		if coord.Group != "" || coord.Artifact != "" {
			return joinHints(
				"Try: add a version (group:artifact:version) or run ksrc deps to see resolved coords.",
				projectHint(flags, meta),
			)
		}
	}
	return joinHints(
		"Try: ksrc deps (list resolved coords), then ksrc fetch <coord> to download sources.",
		projectHint(flags, meta),
	)
}

func noSourcesHintForCoord(coord resolve.Coord) string {
	if coord.Version != "" {
		return fmt.Sprintf("Try: ksrc fetch %s to download sources.", coord.String())
	}
	return "Try: ksrc deps (list resolved coords), then ksrc fetch <coord> to download sources."
}

func joinHints(parts ...string) string {
	var out []string
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	if len(out) == 0 {
		return ""
	}
	return strings.Join(out, " ")
}

func projectHint(flags ResolveFlags, meta resolution.ResolveMeta) string {
	hints := DetectProjectHints(flags.Project)
	var parts []string

	if hints.HasIncludeBuilds {
		parts = append(parts, fmt.Sprintf("Composite build detected; try: --project %s", hints.IncludeBuildHint))
	}
	if hints.Android && !metaHasConfig(meta, "*debugCompileClasspath") && !metaHasConfig(meta, "*DebugCompileClasspath") && strings.TrimSpace(flags.Config) == "" {
		parts = append(parts, "Android detected; try: --config \"*debugCompileClasspath\" (or --config debugCompileClasspath)")
	}
	if hints.KMP && strings.TrimSpace(flags.Targets) == "" {
		parts = append(parts, "KMP detected; try: --targets jvm (or another target)")
	}
	if len(parts) == 0 {
		return ""
	}
	parts = append(parts, "If resolution is slow: narrow with --subproject, --config, --targets, or --scope.")
	return strings.Join(parts, " ")
}

func metaHasConfig(meta resolution.ResolveMeta, pattern string) bool {
	for _, tried := range meta.TriedConfigPatterns {
		if tried == pattern {
			return true
		}
	}
	return false
}

func emitDiagnostics(cmd stderrWriter, meta resolution.ResolveMeta, verbose bool) {
	for _, warning := range meta.Warnings {
		fmt.Fprintf(cmd.ErrOrStderr(), "WARN: %s\n", warning)
	}
	if !verbose {
		return
	}
	for _, line := range meta.Verbose {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fmt.Fprintf(cmd.ErrOrStderr(), "VERBOSE: %s\n", line)
	}
}

type stderrWriter interface {
	ErrOrStderr() io.Writer
}

func emitVerbose(cmd stderrWriter, verbose bool, lines ...string) {
	if !verbose {
		return
	}
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fmt.Fprintf(cmd.ErrOrStderr(), "VERBOSE: %s\n", line)
	}
}

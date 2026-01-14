package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/respawn-app/ksrc/internal/gradle"
	"github.com/respawn-app/ksrc/internal/resolve"
)

type ResolveMeta struct {
	Attempts            []string
	TriedConfigPatterns []string
	Warnings            []string
}

func resolveSources(ctx context.Context, app *App, flags ResolveFlags, dep string, applyFilters bool, allowCacheFallback bool) ([]resolve.SourceJar, []resolve.Coord, ResolveMeta, error) {
	if strings.TrimSpace(flags.Project) == "" {
		flags.Project = "."
	}
	opts := flags.ToOptions()
	opts.Dep = dep

	meta := ResolveMeta{}
	attempts := buildResolveAttempts(opts, flags)
	var lastDeps []resolve.Coord
	var mergedSources []resolve.SourceJar
	var mergedDeps []resolve.Coord
	seenSources := make(map[string]struct{})
	seenDeps := make(map[string]struct{})
	for _, attempt := range attempts {
		res, err := gradle.Resolve(ctx, app.Runner, attempt.Options)
		if err != nil {
			return nil, nil, meta, err
		}
		meta.Attempts = append(meta.Attempts, attempt.Label)
		meta.TriedConfigPatterns = append(meta.TriedConfigPatterns, attempt.ConfigPatterns...)
		meta.Warnings = append(meta.Warnings, res.Warnings...)
		lastDeps = res.Deps
		sources := res.Sources
		if applyFilters {
			sources = resolve.FilterSources(sources, flags.Module, flags.Group, flags.Artifact, flags.Version)
		}
		if flags.All {
			mergeSources(&mergedSources, seenSources, sources)
			mergeDeps(&mergedDeps, seenDeps, res.Deps)
			continue
		}
		if len(sources) > 0 || (!applyFilters && len(res.Deps) > 0) {
			return sources, res.Deps, meta, nil
		}
	}

	var sources []resolve.SourceJar
	if flags.All && (len(mergedSources) > 0 || (!applyFilters && len(mergedDeps) > 0)) {
		return mergedSources, mergedDeps, meta, nil
	}
	if allowCacheFallback {
		if coord, ok := resolve.SelectorToCoord(flags.Module, flags.Group, flags.Artifact, flags.Version); ok {
			if coord.Version == "" {
				cached, err := resolve.FindCachedSources(coord.Group, coord.Artifact, "")
				if err == nil {
					sources = cached
				}
			}
		}
	}
	return sources, lastDeps, meta, nil
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

func noSourcesHintForFlags(flags ResolveFlags, meta ResolveMeta) string {
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

type resolveAttempt struct {
	Options        gradle.ResolveOptions
	Label          string
	ConfigPatterns []string
}

func buildResolveAttempts(opts gradle.ResolveOptions, flags ResolveFlags) []resolveAttempt {
	attempts := []resolveAttempt{{Options: opts, Label: "default"}}
	if strings.TrimSpace(flags.Config) != "" {
		return attempts
	}
	switch opts.Scope {
	case "compile":
		debugPatterns := []string{"*debugCompileClasspath", "*DebugCompileClasspath"}
		attempt := opts
		attempt.Configs = debugPatterns
		attempts = append(attempts, resolveAttempt{
			Options:        attempt,
			Label:          "config:" + strings.Join(debugPatterns, ","),
			ConfigPatterns: debugPatterns,
		})
	case "runtime":
		debugPatterns := []string{"*debugRuntimeClasspath", "*DebugRuntimeClasspath"}
		attempt := opts
		attempt.Configs = debugPatterns
		attempts = append(attempts, resolveAttempt{
			Options:        attempt,
			Label:          "config:" + strings.Join(debugPatterns, ","),
			ConfigPatterns: debugPatterns,
		})
	}
	return attempts
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

func projectHint(flags ResolveFlags, meta ResolveMeta) string {
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

func metaHasConfig(meta ResolveMeta, pattern string) bool {
	for _, tried := range meta.TriedConfigPatterns {
		if tried == pattern {
			return true
		}
	}
	return false
}

func emitWarnings(cmd stderrWriter, meta ResolveMeta) {
	for _, warning := range meta.Warnings {
		fmt.Fprintf(cmd.ErrOrStderr(), "WARN: %s\n", warning)
	}
}

type stderrWriter interface {
	ErrOrStderr() io.Writer
}

func mergeSources(dest *[]resolve.SourceJar, seen map[string]struct{}, sources []resolve.SourceJar) {
	for _, s := range sources {
		key := s.Coord.String() + "|" + s.Path
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		*dest = append(*dest, s)
	}
}

func mergeDeps(dest *[]resolve.Coord, seen map[string]struct{}, deps []resolve.Coord) {
	for _, d := range deps {
		key := d.String()
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		*dest = append(*dest, d)
	}
}

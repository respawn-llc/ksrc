package resolution

import (
	"context"
	"fmt"
	"strings"

	"github.com/respawn-app/ksrc/internal/executil"
	"github.com/respawn-app/ksrc/internal/gradle"
	"github.com/respawn-app/ksrc/internal/resolve"
)

type Service struct {
	Runner  executil.Runner
	Verbose bool
}

type Request struct {
	Project               string
	Module                string
	Group                 string
	Artifact              string
	Version               string
	Scope                 string
	Config                string
	Targets               string
	Subprojects           []string
	Offline               bool
	Refresh               bool
	All                   bool
	IncludeBuildSrc       bool
	IncludeBuildscript    bool
	IncludeIncludedBuilds bool
	Dep                   string
	ApplyFilters          bool
	AllowCacheFallback    bool
}

type ResolveMeta struct {
	Attempts            []string
	TriedConfigPatterns []string
	Warnings            []string
	Verbose             []string
}

type Result struct {
	Sources []resolve.SourceJar
	Deps    []resolve.Coord
	Meta    ResolveMeta
}

func (s Service) ResolveSources(ctx context.Context, req Request) (Result, error) {
	if s.Runner == nil {
		return Result{}, fmt.Errorf("runner is required")
	}
	if strings.TrimSpace(req.Project) == "" {
		req.Project = "."
	}
	opts := req.toOptions()
	opts.Dep = req.Dep
	opts.Verbose = s.Verbose

	meta := ResolveMeta{}
	attempts := buildResolveAttempts(opts, req)
	if s.Verbose {
		labels := make([]string, 0, len(attempts))
		for _, attempt := range attempts {
			labels = append(labels, attempt.Label)
		}
		if len(labels) > 0 {
			meta.Verbose = append(meta.Verbose, fmt.Sprintf("Resolve attempts: %s", strings.Join(labels, ", ")))
		}
		if strings.TrimSpace(req.Config) != "" && len(attempts) == 1 {
			meta.Verbose = append(meta.Verbose, "Skipped debug config attempts because --config was provided.")
		} else if opts.Scope != "compile" && opts.Scope != "runtime" && len(attempts) == 1 {
			meta.Verbose = append(meta.Verbose, fmt.Sprintf("Skipped debug config attempts for scope=%s.", opts.Scope))
		}
	}
	var lastDeps []resolve.Coord
	var mergedSources []resolve.SourceJar
	var mergedDeps []resolve.Coord
	seenSources := make(map[string]struct{})
	seenDeps := make(map[string]struct{})
	var gradleErr *gradle.ExecError
	for _, attempt := range attempts {
		if s.Verbose {
			meta.Verbose = append(meta.Verbose, fmt.Sprintf("Attempt %s: %s", attempt.Label, formatAttemptConfigs(attempt.Options.Configs)))
		}
		res, err := gradle.Resolve(ctx, s.Runner, attempt.Options)
		if err != nil {
			if execErr, ok := gradle.AsExecError(err); ok {
				gradleErr = execErr
				if s.Verbose {
					if len(execErr.Invocation) > 0 {
						meta.Verbose = append(meta.Verbose, execErr.Invocation...)
					}
					meta.Verbose = append(meta.Verbose, fmt.Sprintf("Attempt %s failed; skipping remaining attempts.", attempt.Label))
				}
				meta.Warnings = append(meta.Warnings, gradleFailureWarnings(s.Verbose, execErr)...)
				break
			}
			return Result{}, err
		}
		meta.Attempts = append(meta.Attempts, attempt.Label)
		meta.TriedConfigPatterns = append(meta.TriedConfigPatterns, attempt.ConfigPatterns...)
		meta.Warnings = append(meta.Warnings, res.Warnings...)
		if len(res.Verbose) > 0 {
			meta.Verbose = append(meta.Verbose, res.Verbose...)
		}
		lastDeps = res.Deps
		sources := res.Sources
		if req.ApplyFilters {
			sources = resolve.FilterSources(sources, req.Module, req.Group, req.Artifact, req.Version)
		}
		if s.Verbose {
			meta.Verbose = append(meta.Verbose, fmt.Sprintf("Attempt %s result: sources=%d filtered=%d deps=%d", attempt.Label, len(res.Sources), len(sources), len(res.Deps)))
		}
		if req.All {
			mergeSources(&mergedSources, seenSources, sources)
			mergeDeps(&mergedDeps, seenDeps, res.Deps)
			if s.Verbose {
				meta.Verbose = append(meta.Verbose, fmt.Sprintf("Attempt %s merged into --all results.", attempt.Label))
			}
			continue
		}
		if len(sources) > 0 || (!req.ApplyFilters && len(res.Deps) > 0) {
			if s.Verbose {
				meta.Verbose = append(meta.Verbose, fmt.Sprintf("Attempt %s returned results; skipping remaining attempts.", attempt.Label))
			}
			return Result{Sources: sources, Deps: res.Deps, Meta: meta}, nil
		}
		if s.Verbose {
			meta.Verbose = append(meta.Verbose, fmt.Sprintf("Attempt %s had no results; continuing.", attempt.Label))
		}
	}

	if gradleErr != nil {
		if s.Verbose {
			meta.Verbose = append(meta.Verbose, "Gradle failed; using cache-only fallback.")
		}
		sources, deps, err := cacheFallbackSources(req, req.Dep, req.ApplyFilters)
		if err != nil {
			meta.Warnings = append(meta.Warnings, fmt.Sprintf("Cache-only fallback failed: %v", err))
		}
		if req.All && (len(mergedSources) > 0 || (!req.ApplyFilters && len(mergedDeps) > 0)) {
			mergeSources(&mergedSources, seenSources, sources)
			mergeDeps(&mergedDeps, seenDeps, deps)
			if s.Verbose {
				meta.Verbose = append(meta.Verbose, "Merged cache fallback into --all results.")
			}
			return Result{Sources: mergedSources, Deps: mergedDeps, Meta: meta}, nil
		}
		return Result{Sources: sources, Deps: deps, Meta: meta}, nil
	}

	if req.All && (len(mergedSources) > 0 || (!req.ApplyFilters && len(mergedDeps) > 0)) {
		return Result{Sources: mergedSources, Deps: mergedDeps, Meta: meta}, nil
	}
	if req.AllowCacheFallback {
		if coord, ok := resolve.SelectorToCoord(req.Module, req.Group, req.Artifact, req.Version); ok {
			cached, err := resolve.FindCachedSources(coord.Group, coord.Artifact, coord.Version)
			if err == nil {
				return Result{Sources: cached, Deps: lastDeps, Meta: meta}, nil
			}
		}
	}
	return Result{Sources: nil, Deps: lastDeps, Meta: meta}, nil
}

type resolveAttempt struct {
	Options        gradle.ResolveOptions
	Label          string
	ConfigPatterns []string
}

func buildResolveAttempts(opts gradle.ResolveOptions, req Request) []resolveAttempt {
	attempts := []resolveAttempt{{Options: opts, Label: "default"}}
	if strings.TrimSpace(req.Config) != "" {
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

func formatAttemptConfigs(configs []string) string {
	if len(configs) == 0 {
		return "configs=(default)"
	}
	return fmt.Sprintf("configs=%s", strings.Join(configs, ","))
}

func (r Request) toOptions() gradle.ResolveOptions {
	return gradle.ResolveOptions{
		ProjectDir:            r.Project,
		RootDir:               r.Project,
		Module:                r.Module,
		Group:                 r.Group,
		Artifact:              r.Artifact,
		Version:               r.Version,
		Scope:                 r.Scope,
		Configs:               splitCSV(r.Config),
		Targets:               splitCSV(r.Targets),
		Subprojects:           r.Subprojects,
		Offline:               r.Offline,
		Refresh:               r.Refresh,
		IncludeBuildSrc:       r.IncludeBuildSrc,
		IncludeBuildscript:    r.IncludeBuildscript,
		IncludeIncludedBuilds: r.IncludeIncludedBuilds,
	}
}

func splitCSV(value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func cacheFallbackSources(req Request, dep string, applyFilters bool) ([]resolve.SourceJar, []resolve.Coord, error) {
	var sources []resolve.SourceJar
	var err error

	if dep != "" {
		coord, err := resolve.ParseCoord(dep)
		if err != nil {
			return nil, nil, err
		}
		sources, err = resolve.FindCachedSources(coord.Group, coord.Artifact, coord.Version)
		if err != nil {
			return nil, nil, err
		}
	} else if req.All || !hasExactSelector(req) {
		sources, err = resolve.FindAllCachedSources()
		if err != nil {
			return nil, nil, err
		}
	} else {
		coord, _ := resolve.SelectorToCoord(req.Module, req.Group, req.Artifact, req.Version)
		sources, err = resolve.FindCachedSources(coord.Group, coord.Artifact, coord.Version)
		if err != nil {
			return nil, nil, err
		}
	}

	if applyFilters {
		sources = resolve.FilterSources(sources, req.Module, req.Group, req.Artifact, req.Version)
	}
	return sources, collectCoords(sources), nil
}

func hasExactSelector(req Request) bool {
	coord, ok := resolve.SelectorToCoord(req.Module, req.Group, req.Artifact, req.Version)
	if !ok {
		return false
	}
	return isExactToken(coord.Group) && isExactToken(coord.Artifact) && isExactToken(coord.Version)
}

func isExactToken(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return true
	}
	if strings.ContainsAny(value, "*?[]") {
		return false
	}
	return !strings.Contains(value, ",")
}

func collectCoords(sources []resolve.SourceJar) []resolve.Coord {
	seen := make(map[string]struct{})
	out := make([]resolve.Coord, 0, len(sources))
	for _, s := range sources {
		key := s.Coord.String()
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, s.Coord)
	}
	return out
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

func gradleFailureWarnings(verbose bool, execErr *gradle.ExecError) []string {
	warnings := []string{"Gradle failed; rerun with -v to see Gradle output. Falling back to cache-only resolution."}
	if verbose {
		warnings[0] = "Gradle failed. Falling back to cache-only resolution."
	}
	if !verbose || execErr == nil {
		return warnings
	}
	output := strings.TrimSpace(execErr.Stderr)
	if output == "" {
		output = strings.TrimSpace(execErr.Stdout)
	}
	if output == "" {
		return warnings
	}
	warnings = append(warnings, "Gradle output:")
	warnings = append(warnings, splitLines(output)...)
	return warnings
}

func splitLines(input string) []string {
	lines := strings.Split(input, "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimRight(line, "\r")
		if strings.TrimSpace(line) == "" {
			continue
		}
		out = append(out, line)
	}
	return out
}

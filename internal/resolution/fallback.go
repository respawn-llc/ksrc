package resolution

import (
	"fmt"
	"strings"

	"github.com/respawn-app/ksrc/internal/gradle"
	"github.com/respawn-app/ksrc/internal/resolve"
)

type cacheFallbackMode string

const (
	cacheFallbackNone         cacheFallbackMode = "none"
	cacheFallbackCacheOnly    cacheFallbackMode = "cache-only"
	cacheFallbackSelectorOnly cacheFallbackMode = "selector-only"
)

type cacheFallbackResult struct {
	Mode    cacheFallbackMode
	Sources []resolve.SourceJar
	Deps    []resolve.Coord
	Meta    ResolveMeta
	Found   bool
}

func applyCacheFallbackPolicy(plan resolutionPlan, execution gradleExecution) cacheFallbackResult {
	switch {
	case execution.StopReason == executionGradleFailure:
		return resolveCacheOnlyFallback(plan)
	case !plan.Request.AllowCacheFallback:
		return cacheFallbackResult{Mode: cacheFallbackNone}
	}

	coord, ok := cacheFallbackCoord(plan, execution)
	if !ok {
		return cacheFallbackResult{Mode: cacheFallbackNone}
	}

	sources, err := resolve.FindCachedSourcesWithOptions(coord.Group, coord.Artifact, coord.Version, cacheOptions(plan))
	if err != nil {
		return cacheFallbackResult{Mode: cacheFallbackSelectorOnly}
	}
	return cacheFallbackResult{
		Mode:    cacheFallbackSelectorOnly,
		Sources: sources,
		Found:   true,
	}
}

func cacheFallbackCoord(plan resolutionPlan, execution gradleExecution) (resolve.Coord, bool) {
	resolved := resolve.FilterCoords(execution.LastDeps, plan.Request.Module, plan.Request.Group, plan.Request.Artifact, plan.Request.Version)
	if len(resolved) > 0 {
		return resolved[0], true
	}
	return resolve.SelectorToCoord(plan.Request.Module, plan.Request.Group, plan.Request.Artifact, plan.Request.Version)
}

func resolveCacheOnlyFallback(plan resolutionPlan) cacheFallbackResult {
	result := cacheFallbackResult{Mode: cacheFallbackCacheOnly}
	if plan.Options.Verbose {
		result.Meta.Verbose = append(result.Meta.Verbose, "Gradle failed; using cache-only fallback.")
	}

	sources, deps, err := cacheFallbackSources(plan.Request, plan.Request.Dep, plan.Request.ApplyFilters)
	if err != nil {
		result.Meta.Warnings = append(result.Meta.Warnings, fmt.Sprintf("Cache-only fallback failed: %v", err))
		return result
	}

	result.Sources = sources
	result.Deps = deps
	result.Found = true
	return result
}

func cacheFallbackSources(req Request, dep string, applyFilters bool) ([]resolve.SourceJar, []resolve.Coord, error) {
	var (
		sources []resolve.SourceJar
		err     error
	)

	switch {
	case dep != "":
		coord, parseErr := resolve.ParseCoord(dep)
		if parseErr != nil {
			return nil, nil, parseErr
		}
		sources, err = resolve.FindCachedSourcesWithOptions(coord.Group, coord.Artifact, coord.Version, cacheOptionsForRequest(req))
	case req.All || !hasExactSelector(req):
		sources, err = resolve.FindAllCachedSourcesWithOptions(cacheOptionsForRequest(req))
	default:
		coord, ok := resolve.SelectorToCoord(req.Module, req.Group, req.Artifact, req.Version)
		if !ok {
			return nil, nil, fmt.Errorf("cache fallback requires a concrete selector")
		}
		sources, err = resolve.FindCachedSourcesWithOptions(coord.Group, coord.Artifact, coord.Version, cacheOptionsForRequest(req))
	}
	if err != nil {
		return nil, nil, err
	}

	if applyFilters {
		sources = resolve.FilterSources(sources, req.Module, req.Group, req.Artifact, req.Version)
	}
	return sources, collectCoords(sources), nil
}

func cacheOptions(plan resolutionPlan) resolve.CacheOptions {
	return resolve.CacheOptions{
		GradleUserHome: plan.Request.GradleUserHome,
		WorkDir:        plan.Options.ProjectDir,
	}
}

func cacheOptionsForRequest(req Request) resolve.CacheOptions {
	return resolve.CacheOptions{
		GradleUserHome: req.GradleUserHome,
		WorkDir:        req.Project,
	}
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
	seen := make(map[string]struct{}, len(sources))
	out := make([]resolve.Coord, 0, len(sources))
	for _, source := range sources {
		key := source.Coord.String()
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, source.Coord)
	}
	return out
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

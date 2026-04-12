package resolution

import "github.com/respawn-app/ksrc/internal/resolve"

type mergedAttemptResults struct {
	Sources []resolve.SourceJar
	Deps    []resolve.Coord
}

func assembleResolveResult(plan resolutionPlan, execution gradleExecution, fallback cacheFallbackResult) Result {
	meta := mergeResolveMeta(plan.Meta, execution.Meta, fallback.Meta)
	if plan.Request.All {
		return assembleAllResult(plan, execution, fallback, meta)
	}
	return assembleSingleResult(plan, execution, fallback, meta)
}

func assembleAllResult(plan resolutionPlan, execution gradleExecution, fallback cacheFallbackResult, meta ResolveMeta) Result {
	merged := mergeAttemptOutcomes(execution.Outcomes)
	if execution.StopReason == executionGradleFailure {
		if hasVisibleResult(plan.Request.ApplyFilters, merged.Sources, merged.Deps) {
			mergeSources(&merged.Sources, sourceSet(merged.Sources), fallback.Sources)
			mergeDeps(&merged.Deps, depSet(merged.Deps), fallback.Deps)
			if plan.Options.Verbose {
				meta.Verbose = append(meta.Verbose, "Merged cache fallback into --all results.")
			}
			return Result{Sources: merged.Sources, Deps: merged.Deps, Meta: meta}
		}
		return Result{Sources: fallback.Sources, Deps: fallback.Deps, Meta: meta}
	}

	if hasVisibleResult(plan.Request.ApplyFilters, merged.Sources, merged.Deps) {
		return Result{Sources: merged.Sources, Deps: merged.Deps, Meta: meta}
	}
	if fallback.Mode == cacheFallbackSelectorOnly && fallback.Found {
		return Result{Sources: fallback.Sources, Deps: execution.LastDeps, Meta: meta}
	}
	return Result{Sources: nil, Deps: execution.LastDeps, Meta: meta}
}

func assembleSingleResult(plan resolutionPlan, execution gradleExecution, fallback cacheFallbackResult, meta ResolveMeta) Result {
	if execution.StopReason == executionFoundResults && len(execution.Outcomes) > 0 {
		last := execution.Outcomes[len(execution.Outcomes)-1]
		return Result{Sources: last.FilteredSources, Deps: last.GradleResult.Deps, Meta: meta}
	}
	if execution.StopReason == executionGradleFailure {
		return Result{Sources: fallback.Sources, Deps: fallback.Deps, Meta: meta}
	}
	if fallback.Mode == cacheFallbackSelectorOnly && fallback.Found {
		return Result{Sources: fallback.Sources, Deps: execution.LastDeps, Meta: meta}
	}
	return Result{Sources: nil, Deps: execution.LastDeps, Meta: meta}
}

func mergeResolveMeta(parts ...ResolveMeta) ResolveMeta {
	meta := ResolveMeta{}
	for _, part := range parts {
		meta.Attempts = append(meta.Attempts, part.Attempts...)
		meta.TriedConfigPatterns = append(meta.TriedConfigPatterns, part.TriedConfigPatterns...)
		meta.Warnings = append(meta.Warnings, part.Warnings...)
		meta.Verbose = append(meta.Verbose, part.Verbose...)
	}
	return meta
}

func mergeAttemptOutcomes(outcomes []attemptOutcome) mergedAttemptResults {
	merged := mergedAttemptResults{}
	seenSources := make(map[string]struct{})
	seenDeps := make(map[string]struct{})
	for _, outcome := range outcomes {
		mergeSources(&merged.Sources, seenSources, outcome.FilteredSources)
		mergeDeps(&merged.Deps, seenDeps, outcome.GradleResult.Deps)
	}
	return merged
}

func sourceSet(sources []resolve.SourceJar) map[string]struct{} {
	seen := make(map[string]struct{}, len(sources))
	for _, source := range sources {
		seen[source.Coord.String()+"|"+source.Path] = struct{}{}
	}
	return seen
}

func depSet(deps []resolve.Coord) map[string]struct{} {
	seen := make(map[string]struct{}, len(deps))
	for _, dep := range deps {
		seen[dep.String()] = struct{}{}
	}
	return seen
}

func mergeSources(dest *[]resolve.SourceJar, seen map[string]struct{}, sources []resolve.SourceJar) {
	for _, source := range sources {
		key := source.Coord.String() + "|" + source.Path
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		*dest = append(*dest, source)
	}
}

func mergeDeps(dest *[]resolve.Coord, seen map[string]struct{}, deps []resolve.Coord) {
	for _, dep := range deps {
		key := dep.String()
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		*dest = append(*dest, dep)
	}
}

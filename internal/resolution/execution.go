package resolution

import (
	"context"
	"fmt"

	"github.com/respawn-app/ksrc/internal/gradle"
	"github.com/respawn-app/ksrc/internal/resolve"
)

type executionStopReason string

const (
	executionExhausted     executionStopReason = "exhausted"
	executionFoundResults  executionStopReason = "found-results"
	executionGradleFailure executionStopReason = "gradle-failure"
)

type attemptOutcome struct {
	Attempt         resolveAttempt
	GradleResult    gradle.ResolveResult
	FilteredSources []resolve.SourceJar
}

type gradleExecution struct {
	Outcomes   []attemptOutcome
	LastDeps   []resolve.Coord
	StopReason executionStopReason
	Failure    *gradle.ExecError
	Meta       ResolveMeta
}

func (s Service) executePlan(ctx context.Context, plan resolutionPlan) (gradleExecution, error) {
	execution := gradleExecution{}

	for _, stage := range plan.Stages {
		if s.Verbose {
			execution.Meta.Verbose = append(execution.Meta.Verbose, fmt.Sprintf("Attempt %s: %s", stage.Label, formatAttemptConfigs(stage.Options.Configs)))
		}

		result, err := gradle.Resolve(ctx, s.Runner, stage.Options)
		if err != nil {
			execErr, ok := gradle.AsExecError(err)
			if !ok {
				return gradleExecution{}, err
			}

			execution.Failure = execErr
			execution.StopReason = executionGradleFailure
			if s.Verbose {
				if len(execErr.Invocation) > 0 {
					execution.Meta.Verbose = append(execution.Meta.Verbose, execErr.Invocation...)
				}
				execution.Meta.Verbose = append(execution.Meta.Verbose, fmt.Sprintf("Attempt %s failed; skipping remaining attempts.", stage.Label))
			}
			execution.Meta.Warnings = append(execution.Meta.Warnings, gradleFailureWarnings(s.Verbose, execErr)...)
			return execution, nil
		}

		filteredSources := result.Sources
		if plan.Request.ApplyFilters {
			filteredSources = resolve.FilterSources(filteredSources, plan.Request.Module, plan.Request.Group, plan.Request.Artifact, plan.Request.Version)
		}

		execution.Outcomes = append(execution.Outcomes, attemptOutcome{
			Attempt:         stage,
			GradleResult:    result,
			FilteredSources: filteredSources,
		})
		execution.LastDeps = result.Deps
		execution.Meta.Attempts = append(execution.Meta.Attempts, stage.Label)
		execution.Meta.TriedConfigPatterns = append(execution.Meta.TriedConfigPatterns, stage.ConfigPatterns...)
		execution.Meta.Warnings = append(execution.Meta.Warnings, result.Warnings...)
		execution.Meta.Verbose = append(execution.Meta.Verbose, result.Verbose...)

		if s.Verbose {
			execution.Meta.Verbose = append(execution.Meta.Verbose, fmt.Sprintf("Attempt %s result: sources=%d filtered=%d deps=%d", stage.Label, len(result.Sources), len(filteredSources), len(result.Deps)))
		}

		if plan.Request.All {
			if s.Verbose {
				execution.Meta.Verbose = append(execution.Meta.Verbose, fmt.Sprintf("Attempt %s merged into --all results.", stage.Label))
			}
			continue
		}

		if hasVisibleResult(plan.Request.ApplyFilters, filteredSources, result.Deps) {
			execution.StopReason = executionFoundResults
			if s.Verbose {
				execution.Meta.Verbose = append(execution.Meta.Verbose, fmt.Sprintf("Attempt %s returned results; skipping remaining attempts.", stage.Label))
			}
			return execution, nil
		}

		if s.Verbose {
			execution.Meta.Verbose = append(execution.Meta.Verbose, fmt.Sprintf("Attempt %s had no results; continuing.", stage.Label))
		}
	}

	execution.StopReason = executionExhausted
	return execution, nil
}

func hasVisibleResult(applyFilters bool, sources []resolve.SourceJar, deps []resolve.Coord) bool {
	return len(sources) > 0 || (!applyFilters && len(deps) > 0)
}

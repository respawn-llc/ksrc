package resolution

import (
	"fmt"
	"strings"

	"github.com/respawn-app/ksrc/internal/gradle"
)

type resolutionPlan struct {
	Request Request
	Options gradle.ResolveOptions
	Meta    ResolveMeta
	Stages  []resolveAttempt
}

type resolveAttempt struct {
	Options        gradle.ResolveOptions
	Label          string
	ConfigPatterns []string
}

func buildResolutionPlan(req Request, verbose bool) resolutionPlan {
	if strings.TrimSpace(req.Project) == "" {
		req.Project = "."
	}

	opts := req.toOptions()
	opts.Dep = req.Dep
	opts.Verbose = verbose
	stages := buildResolveAttempts(opts, req)

	return resolutionPlan{
		Request: req,
		Options: opts,
		Meta:    planMeta(req, stages, verbose),
		Stages:  stages,
	}
}

func planMeta(req Request, stages []resolveAttempt, verbose bool) ResolveMeta {
	if !verbose {
		return ResolveMeta{}
	}

	meta := ResolveMeta{}
	labels := make([]string, 0, len(stages))
	for _, stage := range stages {
		labels = append(labels, stage.Label)
	}
	if len(labels) > 0 {
		meta.Verbose = append(meta.Verbose, fmt.Sprintf("Resolve attempts: %s", strings.Join(labels, ", ")))
	}
	if strings.TrimSpace(req.Config) != "" && len(stages) == 1 {
		meta.Verbose = append(meta.Verbose, "Skipped debug config attempts because --config was provided.")
	} else if req.Scope != "compile" && req.Scope != "runtime" && len(stages) == 1 {
		meta.Verbose = append(meta.Verbose, fmt.Sprintf("Skipped debug config attempts for scope=%s.", req.Scope))
	}
	return meta
}

func buildResolveAttempts(opts gradle.ResolveOptions, req Request) []resolveAttempt {
	attempts := []resolveAttempt{{Options: opts, Label: "default"}}
	if strings.TrimSpace(req.Config) != "" {
		return attempts
	}

	var debugPatterns []string
	switch opts.Scope {
	case "compile":
		debugPatterns = []string{"*debugCompileClasspath", "*DebugCompileClasspath"}
	case "runtime":
		debugPatterns = []string{"*debugRuntimeClasspath", "*DebugRuntimeClasspath"}
	default:
		return attempts
	}

	attempt := opts
	attempt.Configs = debugPatterns
	return append(attempts, resolveAttempt{
		Options:        attempt,
		Label:          "config:" + strings.Join(debugPatterns, ","),
		ConfigPatterns: debugPatterns,
	})
}

func formatAttemptConfigs(configs []string) string {
	if len(configs) == 0 {
		return "configs=(default)"
	}
	return fmt.Sprintf("configs=%s", strings.Join(configs, ","))
}

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
	IncludedBuilds      []string
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

	plan := buildResolutionPlan(req, s.Verbose)
	execution, err := s.executePlan(ctx, plan)
	if err != nil {
		return Result{}, err
	}

	fallback := applyCacheFallbackPolicy(plan, execution)
	return assembleResolveResult(plan, execution, fallback), nil
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

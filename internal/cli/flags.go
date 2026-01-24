package cli

import (
	"strings"

	"github.com/respawn-app/ksrc/internal/gradle"
	"github.com/respawn-app/ksrc/internal/resolution"
)

type ResolveFlags struct {
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
}

func (f ResolveFlags) ToOptions() gradle.ResolveOptions {
	return gradle.ResolveOptions{
		ProjectDir:            f.Project,
		RootDir:               f.Project,
		Module:                f.Module,
		Group:                 f.Group,
		Artifact:              f.Artifact,
		Version:               f.Version,
		Scope:                 f.Scope,
		Configs:               splitCSV(f.Config),
		Targets:               splitCSV(f.Targets),
		Subprojects:           f.Subprojects,
		Offline:               f.Offline,
		Refresh:               f.Refresh,
		IncludeBuildSrc:       f.IncludeBuildSrc,
		IncludeBuildscript:    f.IncludeBuildscript,
		IncludeIncludedBuilds: f.IncludeIncludedBuilds,
	}
}

func (f ResolveFlags) ToRequest(dep string, applyFilters bool, allowCacheFallback bool) resolution.Request {
	return resolution.Request{
		Project:               f.Project,
		Module:                f.Module,
		Group:                 f.Group,
		Artifact:              f.Artifact,
		Version:               f.Version,
		Scope:                 f.Scope,
		Config:                f.Config,
		Targets:               f.Targets,
		Subprojects:           f.Subprojects,
		Offline:               f.Offline,
		Refresh:               f.Refresh,
		All:                   f.All,
		IncludeBuildSrc:       f.IncludeBuildSrc,
		IncludeBuildscript:    f.IncludeBuildscript,
		IncludeIncludedBuilds: f.IncludeIncludedBuilds,
		Dep:                   dep,
		ApplyFilters:          applyFilters,
		AllowCacheFallback:    allowCacheFallback,
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

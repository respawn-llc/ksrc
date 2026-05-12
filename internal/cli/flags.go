package cli

import (
	"github.com/respawn-app/ksrc/internal/adapter"
	"github.com/respawn-app/ksrc/internal/gradle"
	"github.com/respawn-app/ksrc/internal/resolution"
	"github.com/respawn-app/ksrc/internal/resolve"
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
	GradleUserHome        string
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
		Configs:               adapter.SplitCSV(f.Config),
		Targets:               adapter.SplitCSV(f.Targets),
		Subprojects:           f.Subprojects,
		Offline:               f.Offline,
		Refresh:               f.Refresh,
		IncludeBuildSrc:       f.IncludeBuildSrc,
		IncludeBuildscript:    f.IncludeBuildscript,
		IncludeIncludedBuilds: f.IncludeIncludedBuilds,
		GradleUserHome:        f.GradleUserHome,
	}
}

func (f ResolveFlags) ToRequest(dep string, applyFilters bool, allowCacheFallback bool) resolution.Request {
	return adapter.BuildRequest(f.ToSpec(dep, applyFilters, allowCacheFallback))
}

func (f ResolveFlags) ToSpec(dep string, applyFilters bool, allowCacheFallback bool) adapter.ResolveSpec {
	return adapter.ResolveSpec{
		Project:               f.Project,
		Module:                f.Module,
		Group:                 f.Group,
		Artifact:              f.Artifact,
		Version:               f.Version,
		Scope:                 f.Scope,
		Config:                adapter.SplitCSV(f.Config),
		Targets:               adapter.SplitCSV(f.Targets),
		Subprojects:           f.Subprojects,
		Offline:               f.Offline,
		Refresh:               f.Refresh,
		All:                   f.All,
		IncludeBuildSrc:       f.IncludeBuildSrc,
		IncludeBuildscript:    f.IncludeBuildscript,
		IncludeIncludedBuilds: f.IncludeIncludedBuilds,
		Dep:                   dep,
		GradleUserHome:        f.GradleUserHome,
		ApplyFilters:          applyFilters,
		AllowCacheFallback:    allowCacheFallback,
	}
}

func (f ResolveFlags) ToCacheOptions() resolve.CacheOptions {
	return resolve.CacheOptions{
		GradleUserHome: f.GradleUserHome,
		WorkDir:        f.Project,
	}
}

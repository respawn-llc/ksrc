package adapter

import "github.com/respawn-app/ksrc/internal/resolution"

type ResolveSpec struct {
	Project               string
	Module                string
	Group                 string
	Artifact              string
	Version               string
	Scope                 string
	Config                []string
	Targets               []string
	Subprojects           []string
	Offline               bool
	Refresh               bool
	All                   bool
	IncludeBuildSrc       bool
	IncludeBuildscript    bool
	IncludeIncludedBuilds bool
	Dep                   string
	GradleUserHome        string
	ApplyFilters          bool
	AllowCacheFallback    bool
}

func BuildRequest(spec ResolveSpec) resolution.Request {
	return resolution.Request{
		Project:               DefaultString(spec.Project, "."),
		Module:                DefaultString(spec.Module, ""),
		Group:                 DefaultString(spec.Group, ""),
		Artifact:              DefaultString(spec.Artifact, ""),
		Version:               DefaultString(spec.Version, ""),
		Scope:                 DefaultString(spec.Scope, "compile"),
		Config:                JoinCSV(spec.Config),
		Targets:               JoinCSV(spec.Targets),
		Subprojects:           CleanList(spec.Subprojects),
		Offline:               spec.Offline,
		Refresh:               spec.Refresh,
		All:                   spec.All,
		IncludeBuildSrc:       spec.IncludeBuildSrc,
		IncludeBuildscript:    spec.IncludeBuildscript,
		IncludeIncludedBuilds: spec.IncludeIncludedBuilds,
		Dep:                   DefaultString(spec.Dep, ""),
		GradleUserHome:        DefaultString(spec.GradleUserHome, ""),
		ApplyFilters:          spec.ApplyFilters,
		AllowCacheFallback:    spec.AllowCacheFallback,
	}
}

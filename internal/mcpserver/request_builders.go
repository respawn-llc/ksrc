package mcpserver

import (
	"strings"

	"github.com/respawn-app/ksrc/internal/adapter"
	"github.com/respawn-app/ksrc/internal/resolve"
)

type resolveSpecInput struct {
	Project       string
	Module        string
	Group         string
	Artifact      string
	Version       string
	Scope         string
	Config        []string
	Targets       []string
	Subprojects   []string
	Buildsrc      *bool
	Buildscript   *bool
	IncludeBuilds *bool
	GradleHome    string
}

func buildSearchSpec(input SearchInput) adapter.ResolveSpec {
	spec := buildResolveSpec(resolveSpecInput{
		Project:     input.Project,
		Group:       input.Group,
		Artifact:    input.Artifact,
		Version:     input.Version,
		Scope:       input.Scope,
		Config:      input.Config,
		Targets:     input.Targets,
		Subprojects: input.Subprojects,
		GradleHome:  input.GradleHome,
	})
	spec.ApplyFilters = true
	spec.AllowCacheFallback = true
	if strings.TrimSpace(input.Group) == "" && strings.TrimSpace(input.Artifact) == "" && strings.TrimSpace(input.Version) == "" {
		spec.All = true
	}
	return spec
}

func buildFileIDSpec(input CatInput, coord resolve.Coord) adapter.ResolveSpec {
	spec := buildResolveSpec(resolveSpecInput{
		Project:       input.Project,
		Module:        coord.String(),
		Version:       coord.Version,
		Scope:         input.Scope,
		Config:        input.Config,
		Targets:       input.Targets,
		Subprojects:   input.Subprojects,
		Buildsrc:      input.Buildsrc,
		Buildscript:   input.Buildscript,
		IncludeBuilds: input.IncludeBuilds,
		GradleHome:    input.GradleHome,
	})
	spec.ApplyFilters = true
	spec.AllowCacheFallback = true
	return spec
}

func buildDepsSpec(input DepsInput) adapter.ResolveSpec {
	spec := buildResolveSpec(resolveSpecInput{
		Project:       input.Project,
		Scope:         input.Scope,
		Config:        input.Config,
		Targets:       input.Targets,
		Subprojects:   input.Subprojects,
		Buildsrc:      input.Buildsrc,
		Buildscript:   input.Buildscript,
		IncludeBuilds: input.IncludeBuilds,
		GradleHome:    input.GradleHome,
	})
	spec.ApplyFilters = false
	spec.AllowCacheFallback = false
	return spec
}

func buildFetchSpec(input FetchInput, coord resolve.Coord) adapter.ResolveSpec {
	spec := buildResolveSpec(resolveSpecInput{
		Project:       input.Project,
		Module:        coord.String(),
		Version:       coord.Version,
		Buildsrc:      input.Buildsrc,
		Buildscript:   input.Buildscript,
		IncludeBuilds: input.IncludeBuilds,
		GradleHome:    input.GradleHome,
	})
	spec.Dep = coord.String()
	spec.ApplyFilters = false
	spec.AllowCacheFallback = false
	return spec
}

func buildResolveToolSpec(input ResolveInput) adapter.ResolveSpec {
	spec := buildResolveSpec(resolveSpecInput{
		Project:       input.Project,
		Group:         input.Group,
		Artifact:      input.Artifact,
		Version:       input.Version,
		Scope:         input.Scope,
		Config:        input.Config,
		Targets:       input.Targets,
		Subprojects:   input.Subprojects,
		Buildsrc:      input.Buildsrc,
		Buildscript:   input.Buildscript,
		IncludeBuilds: input.IncludeBuilds,
		GradleHome:    input.GradleHome,
	})
	spec.ApplyFilters = true
	spec.AllowCacheFallback = true
	if strings.TrimSpace(input.Group) == "" && strings.TrimSpace(input.Artifact) == "" && strings.TrimSpace(input.Version) == "" {
		spec.All = true
	}
	return spec
}

func buildWhereCoordSpec(input WhereInput, coord resolve.Coord, dep string) adapter.ResolveSpec {
	spec := buildResolveSpec(resolveSpecInput{
		Project:       input.Project,
		Module:        coord.String(),
		Version:       coord.Version,
		Scope:         input.Scope,
		Config:        input.Config,
		Targets:       input.Targets,
		Subprojects:   input.Subprojects,
		Buildsrc:      input.Buildsrc,
		Buildscript:   input.Buildscript,
		IncludeBuilds: input.IncludeBuilds,
		GradleHome:    input.GradleHome,
	})
	spec.Dep = dep
	spec.ApplyFilters = true
	spec.AllowCacheFallback = true
	return spec
}

func buildWhereSpec(input WhereInput, group, artifact, version, dep string) adapter.ResolveSpec {
	spec := buildResolveSpec(resolveSpecInput{
		Project:       input.Project,
		Group:         group,
		Artifact:      artifact,
		Version:       version,
		Scope:         input.Scope,
		Config:        input.Config,
		Targets:       input.Targets,
		Subprojects:   input.Subprojects,
		Buildsrc:      input.Buildsrc,
		Buildscript:   input.Buildscript,
		IncludeBuilds: input.IncludeBuilds,
		GradleHome:    input.GradleHome,
	})
	spec.Dep = dep
	spec.ApplyFilters = true
	spec.AllowCacheFallback = true
	return spec
}

func buildResolveSpec(input resolveSpecInput) adapter.ResolveSpec {
	return adapter.ResolveSpec{
		Project:               input.Project,
		Module:                input.Module,
		Group:                 input.Group,
		Artifact:              input.Artifact,
		Version:               input.Version,
		Scope:                 input.Scope,
		Config:                input.Config,
		Targets:               input.Targets,
		Subprojects:           input.Subprojects,
		IncludeBuildSrc:       boolOrDefault(input.Buildsrc, true),
		IncludeBuildscript:    boolOrDefault(input.Buildscript, true),
		IncludeIncludedBuilds: boolOrDefault(input.IncludeBuilds, true),
		GradleUserHome:        input.GradleHome,
	}
}

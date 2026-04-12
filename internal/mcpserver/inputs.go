package mcpserver

type SearchInput struct {
	Query       string   `json:"query" jsonschema:"search pattern as a rg-style glob. required."`
	Context     int      `json:"context,omitempty" jsonschema:"context lines (optional, default: 0)"`
	Group       string   `json:"group,omitempty" jsonschema:"group filter (optional, default: all dependencies)"`
	Artifact    string   `json:"artifact,omitempty" jsonschema:"artifact filter (optional, default: all artifacts)"`
	Version     string   `json:"version,omitempty" jsonschema:"version filter (optional, default: all versions)"`
	Config      []string `json:"config,omitempty" jsonschema:"Gradle config filters (optional, default: scope defaults)"`
	Project     string   `json:"project,omitempty" jsonschema:"project path (optional, default: . (cwd))"`
	Subprojects []string `json:"subprojects,omitempty" jsonschema:"subproject filters (optional, default: all subprojects)"`
	RgArgs      []string `json:"rgArgs,omitempty" jsonschema:"extra rg args (optional, default: none)"`
	Scope       string   `json:"scope,omitempty" jsonschema:"dependency scope (optional, default: compile)"`
	Targets     []string `json:"targets,omitempty" jsonschema:"KMP target filters (optional, default: all targets)"`
}

type CatInput struct {
	FileID string `json:"fileId" jsonschema:"file-id from search tool output. required."`
	Lines  string `json:"lines,omitempty" jsonschema:"line range A,B (optional, default: entire file)"`
}

type DepsInput struct {
	Project       string   `json:"project,omitempty" jsonschema:"project path (optional, default: . (cwd))"`
	Scope         string   `json:"scope,omitempty" jsonschema:"dependency scope (optional, default: compile)"`
	Config        []string `json:"config,omitempty" jsonschema:"Gradle config filters (optional, default: scope defaults)"`
	Targets       []string `json:"targets,omitempty" jsonschema:"target filters (optional, default: all targets)"`
	Subprojects   []string `json:"subprojects,omitempty" jsonschema:"subproject filters (optional, default: all subprojects)"`
	Buildsrc      *bool    `json:"buildsrc,omitempty" jsonschema:"include buildSrc (optional, default: true)"`
	Buildscript   *bool    `json:"buildscript,omitempty" jsonschema:"include buildscript (optional, default: true)"`
	IncludeBuilds *bool    `json:"includeBuilds,omitempty" jsonschema:"include builds (optional, default: true)"`
	Group         string   `json:"group,omitempty" jsonschema:"group filter (optional, default: all dependencies)"`
	Artifact      string   `json:"artifact,omitempty" jsonschema:"artifact filter (optional, default: all artifacts)"`
	Version       string   `json:"version,omitempty" jsonschema:"version filter (optional, default: all versions)"`
}

type FetchInput struct {
	Group         string `json:"group" jsonschema:"group. required."`
	Artifact      string `json:"artifact" jsonschema:"artifact. required."`
	Version       string `json:"version" jsonschema:"version. required."`
	Project       string `json:"project,omitempty" jsonschema:"project path (optional, default: .)"`
	Buildsrc      *bool  `json:"buildsrc,omitempty" jsonschema:"include buildSrc (optional, default: true)"`
	Buildscript   *bool  `json:"buildscript,omitempty" jsonschema:"include buildscript (optional, default: true)"`
	IncludeBuilds *bool  `json:"includeBuilds,omitempty" jsonschema:"include builds (optional, default: true)"`
}

type ResolveInput struct {
	Project       string   `json:"project,omitempty" jsonschema:"project path (optional, default: .)"`
	Group         string   `json:"group,omitempty" jsonschema:"group filter (optional, default: all dependencies)"`
	Artifact      string   `json:"artifact,omitempty" jsonschema:"artifact filter (optional, default: all artifacts)"`
	Version       string   `json:"version,omitempty" jsonschema:"version filter (optional, default: all versions)"`
	Scope         string   `json:"scope,omitempty" jsonschema:"dependency scope (optional, default: compile)"`
	Config        []string `json:"config,omitempty" jsonschema:"Gradle config filters (optional, default: scope defaults)"`
	Targets       []string `json:"targets,omitempty" jsonschema:"target filters (optional, default: all targets)"`
	Subprojects   []string `json:"subprojects,omitempty" jsonschema:"subproject filters (optional, default: all subprojects)"`
	Buildsrc      *bool    `json:"buildsrc,omitempty" jsonschema:"include buildSrc (optional, default: true)"`
	Buildscript   *bool    `json:"buildscript,omitempty" jsonschema:"include buildscript (optional, default: true)"`
	IncludeBuilds *bool    `json:"includeBuilds,omitempty" jsonschema:"include builds (optional, default: true)"`
}

type WhereInput struct {
	PathOrCoord   string   `json:"pathOrCoord" jsonschema:"file-id or path/coord. required."`
	Project       string   `json:"project,omitempty" jsonschema:"project path (optional, default: . (cwd))"`
	Group         string   `json:"group,omitempty" jsonschema:"group filter (optional, default: all dependencies; required for path lookup)"`
	Artifact      string   `json:"artifact,omitempty" jsonschema:"artifact filter (optional, default: all artifacts; required for path lookup)"`
	Version       string   `json:"version,omitempty" jsonschema:"version filter (optional, default: all versions)"`
	Scope         string   `json:"scope,omitempty" jsonschema:"dependency scope (optional, default: compile)"`
	Config        []string `json:"config,omitempty" jsonschema:"Gradle config filters (optional, default: scope defaults)"`
	Targets       []string `json:"targets,omitempty" jsonschema:"target filters (optional, default: all targets)"`
	Subprojects   []string `json:"subprojects,omitempty" jsonschema:"subproject filters (optional, default: all subprojects)"`
	Buildsrc      *bool    `json:"buildsrc,omitempty" jsonschema:"include buildSrc (optional, default: true)"`
	Buildscript   *bool    `json:"buildscript,omitempty" jsonschema:"include buildscript (optional, default: true)"`
	IncludeBuilds *bool    `json:"includeBuilds,omitempty" jsonschema:"include builds (optional, default: true)"`
}

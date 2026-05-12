package mcpserver

import (
	"fmt"
	"strings"
)

type rgArgRule struct {
	Canonical     string
	RequiresValue bool
}

var allowedRgArgs = map[string]rgArgRule{
	"-F":               {Canonical: "-F"},
	"--fixed-strings":  {Canonical: "--fixed-strings"},
	"-P":               {Canonical: "-P"},
	"--pcre2":          {Canonical: "--pcre2"},
	"-a":               {Canonical: "-a"},
	"--text":           {Canonical: "--text"},
	"-i":               {Canonical: "-i"},
	"--ignore-case":    {Canonical: "--ignore-case"},
	"-s":               {Canonical: "-s"},
	"--case-sensitive": {Canonical: "--case-sensitive"},
	"-w":               {Canonical: "-w"},
	"--word-regexp":    {Canonical: "--word-regexp"},
	"-x":               {Canonical: "-x"},
	"--line-regexp":    {Canonical: "--line-regexp"},
	"-g":               {Canonical: "-g", RequiresValue: true},
	"--glob":           {Canonical: "--glob", RequiresValue: true},
	"-m":               {Canonical: "-m", RequiresValue: true},
	"--max-count":      {Canonical: "--max-count", RequiresValue: true},
}

func sanitizeRgArgs(values []string) ([]string, error) {
	cleaned := make([]string, 0, len(values))
	for i := 0; i < len(values); i++ {
		arg := strings.TrimSpace(values[i])
		if arg == "" {
			continue
		}
		if !strings.HasPrefix(arg, "-") {
			return nil, fmt.Errorf("unsupported rg arg %q: only a safe subset of flags is allowed", arg)
		}

		name := arg
		inlineValue := ""
		if strings.HasPrefix(arg, "--") {
			if base, value, ok := strings.Cut(arg, "="); ok {
				name = base
				inlineValue = value
			}
		}

		rule, ok := allowedRgArgs[name]
		if !ok {
			return nil, fmt.Errorf("unsupported rg arg %q: only search-safe flags are allowed", arg)
		}
		if !rule.RequiresValue {
			if inlineValue != "" {
				return nil, fmt.Errorf("rg arg %q does not accept an inline value", arg)
			}
			cleaned = append(cleaned, arg)
			continue
		}
		if inlineValue != "" {
			cleaned = append(cleaned, arg)
			continue
		}
		if i+1 >= len(values) {
			return nil, fmt.Errorf("rg arg %q requires a value", arg)
		}
		value := strings.TrimSpace(values[i+1])
		if value == "" || strings.HasPrefix(value, "-") {
			return nil, fmt.Errorf("rg arg %q requires a non-flag value", arg)
		}
		cleaned = append(cleaned, arg, value)
		i++
	}
	return cleaned, nil
}

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
	GradleHome  string   `json:"gradleUserHome,omitempty" jsonschema:"Gradle user home (optional, default: GRADLE_USER_HOME or ~/.gradle)"`
}

type CatInput struct {
	FileID        string   `json:"fileId" jsonschema:"file-id from search tool output. required."`
	Lines         string   `json:"lines,omitempty" jsonschema:"line range A,B (optional, default: entire file)"`
	Project       string   `json:"project,omitempty" jsonschema:"project path for fallback resolution (optional, default: . (cwd))"`
	Config        []string `json:"config,omitempty" jsonschema:"Gradle config filters for fallback resolution (optional, default: scope defaults)"`
	Subprojects   []string `json:"subprojects,omitempty" jsonschema:"subproject filters for fallback resolution (optional, default: all subprojects)"`
	Scope         string   `json:"scope,omitempty" jsonschema:"dependency scope for fallback resolution (optional, default: compile)"`
	Targets       []string `json:"targets,omitempty" jsonschema:"KMP target filters for fallback resolution (optional, default: all targets)"`
	Buildsrc      *bool    `json:"buildsrc,omitempty" jsonschema:"include buildSrc in fallback resolution (optional, default: true)"`
	Buildscript   *bool    `json:"buildscript,omitempty" jsonschema:"include buildscript in fallback resolution (optional, default: true)"`
	IncludeBuilds *bool    `json:"includeBuilds,omitempty" jsonschema:"include builds in fallback resolution (optional, default: true)"`
	GradleHome    string   `json:"gradleUserHome,omitempty" jsonschema:"Gradle user home for fallback resolution (optional, default: GRADLE_USER_HOME or ~/.gradle)"`
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
	GradleHome    string   `json:"gradleUserHome,omitempty" jsonschema:"Gradle user home (optional, default: GRADLE_USER_HOME or ~/.gradle)"`
}

type FetchInput struct {
	Group         string `json:"group" jsonschema:"group. required."`
	Artifact      string `json:"artifact" jsonschema:"artifact. required."`
	Version       string `json:"version" jsonschema:"version. required."`
	Project       string `json:"project,omitempty" jsonschema:"project path (optional, default: .)"`
	Buildsrc      *bool  `json:"buildsrc,omitempty" jsonschema:"include buildSrc (optional, default: true)"`
	Buildscript   *bool  `json:"buildscript,omitempty" jsonschema:"include buildscript (optional, default: true)"`
	IncludeBuilds *bool  `json:"includeBuilds,omitempty" jsonschema:"include builds (optional, default: true)"`
	GradleHome    string `json:"gradleUserHome,omitempty" jsonschema:"Gradle user home (optional, default: GRADLE_USER_HOME or ~/.gradle)"`
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
	GradleHome    string   `json:"gradleUserHome,omitempty" jsonschema:"Gradle user home (optional, default: GRADLE_USER_HOME or ~/.gradle)"`
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
	GradleHome    string   `json:"gradleUserHome,omitempty" jsonschema:"Gradle user home (optional, default: GRADLE_USER_HOME or ~/.gradle)"`
}

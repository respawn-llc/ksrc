package mcpserver

import "strings"

func catInputHasExplicitResolutionContext(input CatInput) bool {
	return strings.TrimSpace(input.Project) != "" ||
		strings.TrimSpace(input.GradleHome) != "" ||
		strings.TrimSpace(input.Scope) != "" ||
		len(cleanList(input.Config)) > 0 ||
		len(cleanList(input.Targets)) > 0 ||
		len(cleanList(input.Subprojects)) > 0 ||
		input.Buildsrc != nil ||
		input.Buildscript != nil ||
		input.IncludeBuilds != nil
}

func whereInputHasExplicitFileIDContext(input WhereInput) bool {
	return strings.TrimSpace(input.Project) != "" ||
		strings.TrimSpace(input.GradleHome) != "" ||
		strings.TrimSpace(input.Scope) != "" ||
		len(cleanList(input.Config)) > 0 ||
		len(cleanList(input.Targets)) > 0 ||
		len(cleanList(input.Subprojects)) > 0 ||
		input.Buildsrc != nil ||
		input.Buildscript != nil ||
		input.IncludeBuilds != nil
}

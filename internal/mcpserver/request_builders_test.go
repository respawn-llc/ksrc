package mcpserver_test

import (
	"reflect"
	"testing"

	"github.com/respawn-app/ksrc/internal/cli"
	"github.com/respawn-app/ksrc/internal/mcpserver"
	"github.com/respawn-app/ksrc/internal/resolve"
)

func TestBuildSearchSpecMatchesCLIRequest(t *testing.T) {
	input := mcpserver.SearchInput{
		Project:     " ./sample ",
		Config:      []string{" compileClasspath ", "", " *debugCompileClasspath "},
		Targets:     []string{" jvm ", "", " iosArm64 "},
		Subprojects: []string{" app ", "", " lib "},
	}

	got := mcpserver.BuildSearchRequestForTest(input)
	want := cli.ResolveFlags{
		Project:               input.Project,
		Config:                " compileClasspath , , *debugCompileClasspath ",
		Targets:               " jvm , , iosArm64 ",
		Subprojects:           input.Subprojects,
		All:                   true,
		IncludeBuildSrc:       true,
		IncludeBuildscript:    true,
		IncludeIncludedBuilds: true,
	}.ToRequest("", true, true)

	assertRequestsEqual(t, want, got)
}

func TestBuildResolveSpecMatchesCLIRequest(t *testing.T) {
	buildsrc := false
	includeBuilds := false
	input := mcpserver.ResolveInput{
		Project:       " ./sample ",
		Group:         " org.jetbrains.kotlinx ",
		Artifact:      " kotlinx-datetime ",
		Version:       " 0.7.1 ",
		Scope:         " runtime ",
		Config:        []string{" runtimeClasspath ", ""},
		Targets:       []string{" jvm ", ""},
		Subprojects:   []string{" shared ", ""},
		Buildsrc:      &buildsrc,
		IncludeBuilds: &includeBuilds,
	}

	got := mcpserver.BuildResolveRequestForTest(input)
	want := cli.ResolveFlags{
		Project:               input.Project,
		Group:                 input.Group,
		Artifact:              input.Artifact,
		Version:               input.Version,
		Scope:                 input.Scope,
		Config:                " runtimeClasspath ",
		Targets:               " jvm ",
		Subprojects:           input.Subprojects,
		IncludeBuildSrc:       false,
		IncludeBuildscript:    true,
		IncludeIncludedBuilds: false,
	}.ToRequest("", true, true)

	assertRequestsEqual(t, want, got)
}

func TestBuildFetchSpecMatchesCLIRequest(t *testing.T) {
	buildscript := false
	coord := resolve.Coord{Group: "org.jetbrains.kotlinx", Artifact: "kotlinx-datetime", Version: "0.7.1"}
	input := mcpserver.FetchInput{
		Project:     " ./sample ",
		Buildscript: &buildscript,
	}

	got := mcpserver.BuildFetchRequestForTest(input, coord)
	want := cli.ResolveFlags{
		Project:               input.Project,
		Module:                coord.String(),
		Version:               coord.Version,
		IncludeBuildSrc:       true,
		IncludeBuildscript:    false,
		IncludeIncludedBuilds: true,
	}.ToRequest(coord.String(), false, false)

	assertRequestsEqual(t, want, got)
}

func TestBuildWhereCoordSpecMatchesCLIRequest(t *testing.T) {
	buildsrc := false
	buildscript := false
	coord := resolve.Coord{Group: "org.jetbrains.kotlinx", Artifact: "kotlinx-datetime", Version: "0.7.1"}
	input := mcpserver.WhereInput{
		Project:       " ./sample ",
		Scope:         " runtime ",
		Config:        []string{" runtimeClasspath ", ""},
		Targets:       []string{" jvm ", ""},
		Subprojects:   []string{" shared ", ""},
		Buildsrc:      &buildsrc,
		Buildscript:   &buildscript,
		IncludeBuilds: nil,
	}

	got := mcpserver.BuildWhereCoordRequestForTest(input, coord, coord.String())
	want := cli.ResolveFlags{
		Project:               input.Project,
		Module:                coord.String(),
		Version:               coord.Version,
		Scope:                 input.Scope,
		Config:                " runtimeClasspath ",
		Targets:               " jvm ",
		Subprojects:           input.Subprojects,
		IncludeBuildSrc:       false,
		IncludeBuildscript:    false,
		IncludeIncludedBuilds: true,
	}.ToRequest(coord.String(), true, true)

	assertRequestsEqual(t, want, got)
}

func assertRequestsEqual(t *testing.T, want, got any) {
	t.Helper()
	if !reflect.DeepEqual(want, got) {
		t.Fatalf("request mismatch\nwant: %#v\n got: %#v", want, got)
	}
}

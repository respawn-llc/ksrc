package adapter

import "testing"

func TestBuildRequestNormalizesValues(t *testing.T) {
	request := BuildRequest(ResolveSpec{
		Project:               "  ",
		Module:                " org.example:demo ",
		Group:                 " org.example ",
		Artifact:              " demo ",
		Version:               " 1.0.0 ",
		Scope:                 " ",
		Config:                []string{" compileClasspath ", "", " runtimeClasspath "},
		Targets:               []string{" jvm ", ""},
		Subprojects:           []string{" app ", "", "lib "},
		Dep:                   " org.example:demo:1.0.0 ",
		GradleUserHome:        " .gradle-custom ",
		ApplyFilters:          true,
		AllowCacheFallback:    true,
		IncludeBuildSrc:       true,
		IncludeBuildscript:    true,
		IncludeIncludedBuilds: true,
	})

	if request.Project != "." {
		t.Fatalf("expected default project, got %q", request.Project)
	}
	if request.Scope != "compile" {
		t.Fatalf("expected default scope, got %q", request.Scope)
	}
	if request.Module != "org.example:demo" {
		t.Fatalf("unexpected module: %q", request.Module)
	}
	if request.Group != "org.example" || request.Artifact != "demo" || request.Version != "1.0.0" {
		t.Fatalf("unexpected selector: %+v", request)
	}
	if request.Config != "compileClasspath,runtimeClasspath" {
		t.Fatalf("unexpected config: %q", request.Config)
	}
	if request.Targets != "jvm" {
		t.Fatalf("unexpected targets: %q", request.Targets)
	}
	if len(request.Subprojects) != 2 || request.Subprojects[0] != "app" || request.Subprojects[1] != "lib" {
		t.Fatalf("unexpected subprojects: %#v", request.Subprojects)
	}
	if request.Dep != "org.example:demo:1.0.0" {
		t.Fatalf("unexpected dep: %q", request.Dep)
	}
	if request.GradleUserHome != ".gradle-custom" {
		t.Fatalf("unexpected gradle user home: %q", request.GradleUserHome)
	}
}

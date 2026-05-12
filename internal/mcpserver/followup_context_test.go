package mcpserver

import "testing"

func TestCatInputHasExplicitResolutionContext(t *testing.T) {
	if catInputHasExplicitResolutionContext(CatInput{}) {
		t.Fatal("expected empty cat input to use tracked follow-up lookup")
	}
	buildscript := false
	if !catInputHasExplicitResolutionContext(CatInput{
		Project:     " ./sample ",
		Config:      []string{" compileClasspath "},
		Targets:     []string{" jvm "},
		Subprojects: []string{" app "},
		Buildscript: &buildscript,
		GradleHome:  " .gradle-custom ",
	}) {
		t.Fatal("expected explicit cat context to bypass tracked follow-up lookup")
	}
	if !catInputHasExplicitResolutionContext(CatInput{GradleHome: " .gradle-custom "}) {
		t.Fatal("expected explicit cat Gradle user home to bypass tracked follow-up lookup")
	}
}

func TestWhereInputHasExplicitFileIDContext(t *testing.T) {
	if whereInputHasExplicitFileIDContext(WhereInput{}) {
		t.Fatal("expected empty where input to use tracked follow-up lookup")
	}
	buildsrc := false
	if !whereInputHasExplicitFileIDContext(WhereInput{
		Project:     " ./sample ",
		Scope:       " runtime ",
		Config:      []string{" runtimeClasspath "},
		Targets:     []string{" iosArm64 "},
		Subprojects: []string{" shared "},
		Buildsrc:    &buildsrc,
		GradleHome:  " .gradle-custom ",
	}) {
		t.Fatal("expected explicit where context to bypass tracked follow-up lookup")
	}
	if !whereInputHasExplicitFileIDContext(WhereInput{GradleHome: " .gradle-custom "}) {
		t.Fatal("expected explicit where Gradle user home to bypass tracked follow-up lookup")
	}
}

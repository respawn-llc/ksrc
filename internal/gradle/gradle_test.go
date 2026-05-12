package gradle

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/respawn-app/ksrc/internal/executil"
	"github.com/respawn-app/ksrc/internal/resolve"
)

type fakeRunner struct{}

func (fakeRunner) Run(_ context.Context, _ string, _ string, _ ...string) (string, string, error) {
	return "", "", nil
}

func (fakeRunner) LookPath(_ string) (string, error) {
	return "", errors.New("not found")
}

func TestFindGradlePrefersLocalWrapper(t *testing.T) {
	dir := t.TempDir()
	wrapper := filepath.Join(dir, "gradlew")
	if err := os.WriteFile(wrapper, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("write wrapper: %v", err)
	}
	cmd, err := findGradle(fakeRunner{}, dir, "")
	if err != nil {
		t.Fatalf("findGradle: %v", err)
	}
	if cmd != "./gradlew" {
		t.Fatalf("expected local wrapper, got %q", cmd)
	}
}

func TestFindGradleFallsBackToRootWrapper(t *testing.T) {
	root := t.TempDir()
	included := t.TempDir()
	wrapper := filepath.Join(root, "gradlew")
	if err := os.WriteFile(wrapper, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("write wrapper: %v", err)
	}
	cmd, err := findGradle(fakeRunner{}, included, root)
	if err != nil {
		t.Fatalf("findGradle: %v", err)
	}
	if cmd != wrapper {
		t.Fatalf("expected root wrapper %q, got %q", wrapper, cmd)
	}
}

func TestMergeResultsIncludesWarnings(t *testing.T) {
	base := ResolveResult{
		Sources: []resolve.SourceJar{},
		Deps:    []resolve.Coord{},
		Warnings: []string{
			"base warning",
		},
	}
	extra := ResolveResult{
		Warnings: []string{
			"extra warning",
		},
	}
	merged := mergeResults(base, extra)
	if len(merged.Warnings) != 2 {
		t.Fatalf("expected 2 warnings, got %d", len(merged.Warnings))
	}
}

func TestResolveStopsAfterRootSources(t *testing.T) {
	root := t.TempDir()
	runner := &scriptedRunner{
		responses: map[string]runResult{
			root: {
				stdout: outputRecordLine(t, outputRecord{Type: "source", Group: "com.example", Artifact: "demo", Version: "1.0.0", Path: "/tmp/demo-sources.jar"}),
			},
		},
	}
	opts := ResolveOptions{
		ProjectDir:      root,
		IncludeBuildSrc: true,
	}
	res, err := Resolve(context.Background(), runner, opts)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if len(res.Sources) != 1 {
		t.Fatalf("expected 1 source, got %d", len(res.Sources))
	}
	if len(runner.calls) != 1 {
		t.Fatalf("expected 1 Gradle call, got %d", len(runner.calls))
	}
}

func TestResolveOncePassesExplicitGradleUserHomeToWrapper(t *testing.T) {
	root := t.TempDir()
	userHome := "relative-gradle-home"
	runner := &scriptedRunner{
		responses: map[string]runResult{
			root: {
				stdout: outputRecordLine(t, outputRecord{Type: "source", Group: "com.example", Artifact: "demo", Version: "1.0.0", Path: "/tmp/demo-sources.jar"}),
			},
		},
	}
	if err := os.WriteFile(filepath.Join(root, "gradlew"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("write wrapper: %v", err)
	}

	_, err := Resolve(context.Background(), runner, ResolveOptions{
		ProjectDir:      root,
		GradleUserHome:  userHome,
		IncludeBuildSrc: true,
	})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if len(runner.calls) != 1 {
		t.Fatalf("expected 1 Gradle call, got %d", len(runner.calls))
	}
	if !strings.Contains(runner.calls[0], "--gradle-user-home "+userHome) {
		t.Fatalf("expected --gradle-user-home in args, got %q", runner.calls[0])
	}
}

func TestResolveFallsBackToBuildSrc(t *testing.T) {
	dir := t.TempDir()
	buildSrcDir := filepath.Join(dir, "buildSrc")
	if err := os.MkdirAll(buildSrcDir, 0o755); err != nil {
		t.Fatalf("mkdir buildSrc: %v", err)
	}
	if err := os.WriteFile(filepath.Join(buildSrcDir, "build.gradle.kts"), []byte(""), 0o644); err != nil {
		t.Fatalf("write buildSrc build file: %v", err)
	}

	runner := &scriptedRunner{
		responses: map[string]runResult{
			dir: {
				stdout: "",
			},
			buildSrcDir: {
				stdout: outputRecordLine(t, outputRecord{Type: "source", Group: "com.example", Artifact: "demo", Version: "1.0.0", Path: "/tmp/demo-sources.jar"}),
			},
		},
	}
	opts := ResolveOptions{
		ProjectDir:      dir,
		IncludeBuildSrc: true,
	}
	res, err := Resolve(context.Background(), runner, opts)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if len(res.Sources) != 1 {
		t.Fatalf("expected 1 source, got %d", len(res.Sources))
	}
	if len(runner.calls) != 2 {
		t.Fatalf("expected 2 Gradle calls, got %d", len(runner.calls))
	}
}

func TestResolveFallsBackToIncludedBuilds(t *testing.T) {
	root := t.TempDir()
	included := t.TempDir()
	runner := &scriptedRunner{
		responses: map[string]runResult{
			root: {
				stdout: outputRecordLine(t, outputRecord{Type: "include", Path: included}),
			},
			included: {
				stdout: outputRecordLine(t, outputRecord{Type: "source", Group: "com.example", Artifact: "demo", Version: "1.0.0", Path: "/tmp/demo-sources.jar"}),
			},
		},
	}
	opts := ResolveOptions{
		ProjectDir:            root,
		IncludeIncludedBuilds: true,
	}
	res, err := Resolve(context.Background(), runner, opts)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if len(res.Sources) != 1 {
		t.Fatalf("expected 1 source, got %d", len(res.Sources))
	}
	if len(runner.calls) != 2 {
		t.Fatalf("expected 2 Gradle calls, got %d", len(runner.calls))
	}
}

func TestResolveIncludedBuildsDedupesAndBfs(t *testing.T) {
	root := t.TempDir()
	buildA := filepath.Join(root, "buildA")
	buildB := filepath.Join(root, "buildB")
	buildC := filepath.Join(root, "buildC")

	resolver := &stubResolver{
		results: map[string]ResolveResult{
			root:   {IncludedBuilds: []string{buildA, buildB, buildA}},
			buildA: {IncludedBuilds: []string{buildB, buildC}},
			buildB: {},
			buildC: {},
		},
	}
	opts := ResolveOptions{
		ProjectDir:            root,
		IncludeIncludedBuilds: true,
	}
	if _, err := resolveWith(context.Background(), fakeRunner{}, opts, resolver); err != nil {
		t.Fatalf("resolveWith: %v", err)
	}
	expected := []string{root, buildA, buildB, buildC}
	if !reflect.DeepEqual(resolver.calls, expected) {
		t.Fatalf("expected call order %v, got %v", expected, resolver.calls)
	}
}

func TestResolveParsesMachineReadableRecordsWithPipesInPath(t *testing.T) {
	root := t.TempDir()
	pipePath := filepath.Join(root, "demo|sources.jar")
	included := filepath.Join(root, "build|one")
	runner := &scriptedRunner{
		responses: map[string]runResult{
			root: {
				stdout: outputRecordLine(t, outputRecord{Type: "include", Path: included}) +
					outputRecordLine(t, outputRecord{Type: "source", Group: "com.example", Artifact: "demo", Version: "1.0.0", Path: pipePath}),
			},
		},
	}
	res, err := Resolve(context.Background(), runner, ResolveOptions{ProjectDir: root, IncludeIncludedBuilds: true})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if len(res.Sources) != 1 || res.Sources[0].Path != pipePath {
		t.Fatalf("unexpected sources: %+v", res.Sources)
	}
	if len(res.IncludedBuilds) != 1 || res.IncludedBuilds[0] != included {
		t.Fatalf("unexpected included builds: %+v", res.IncludedBuilds)
	}
}

func TestResolveParsesSourceSelectedByRecords(t *testing.T) {
	root := t.TempDir()
	runner := &scriptedRunner{
		responses: map[string]runResult{
			root: {
				stdout: outputRecordLine(t, outputRecord{
					Type:     "source",
					Group:    "org.jetbrains.kotlinx",
					Artifact: "kotlinx-datetime-jvm",
					Version:  "0.7.1",
					Path:     "/tmp/kotlinx-datetime-jvm-sources.jar",
					SelectedBy: []outputCoord{
						{Group: "org.jetbrains.kotlinx", Artifact: "kotlinx-datetime", Version: "0.7.1"},
					},
				}),
			},
		},
	}
	res, err := Resolve(context.Background(), runner, ResolveOptions{ProjectDir: root})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if len(res.Sources) != 1 {
		t.Fatalf("expected one source, got %+v", res.Sources)
	}
	want := []resolve.Coord{{Group: "org.jetbrains.kotlinx", Artifact: "kotlinx-datetime", Version: "0.7.1"}}
	if !reflect.DeepEqual(res.Sources[0].SelectedBy, want) {
		t.Fatalf("unexpected selectedBy: %+v", res.Sources[0].SelectedBy)
	}
}

func TestResolveDoesNotDisableConfigurationCache(t *testing.T) {
	root := t.TempDir()
	runner := &scriptedRunner{
		responses: map[string]runResult{
			root: {},
		},
	}
	_, err := Resolve(context.Background(), runner, ResolveOptions{ProjectDir: root})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if len(runner.calls) != 1 {
		t.Fatalf("expected 1 Gradle call, got %d", len(runner.calls))
	}
	if strings.Contains(runner.calls[0], "--no-configuration-cache") {
		t.Fatalf("expected Gradle args to allow configuration cache, got %q", runner.calls[0])
	}
}

func TestWriteInitScriptUsesStableCachePath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CACHE_HOME", filepath.Join(home, ".cache"))

	first, firstCleanup, err := writeInitScript()
	if err != nil {
		t.Fatalf("write first init script: %v", err)
	}
	firstCleanup()
	second, secondCleanup, err := writeInitScript()
	if err != nil {
		t.Fatalf("write second init script: %v", err)
	}
	secondCleanup()

	if first != second {
		t.Fatalf("expected stable init script path, got %q and %q", first, second)
	}
	if !strings.Contains(filepath.Base(first), initScriptTemplateVersion) {
		t.Fatalf("expected init script path to include template version, got %q", first)
	}
	got, err := os.ReadFile(first)
	if err != nil {
		t.Fatalf("read stable init script: %v", err)
	}
	if string(got) != InitScript() {
		t.Fatal("stable init script content differs from rendered init script")
	}
}

func TestWriteInitScriptCachePathInvalidatesOnVersionOrContentChange(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CACHE_HOME", filepath.Join(home, ".cache"))

	base, baseCleanup, err := writeInitScriptContent("v-test", "println 'one'\n")
	if err != nil {
		t.Fatalf("write base init script: %v", err)
	}
	baseCleanup()
	changedContent, changedContentCleanup, err := writeInitScriptContent("v-test", "println 'two'\n")
	if err != nil {
		t.Fatalf("write changed-content init script: %v", err)
	}
	changedContentCleanup()
	changedVersion, changedVersionCleanup, err := writeInitScriptContent("v-test-2", "println 'one'\n")
	if err != nil {
		t.Fatalf("write changed-version init script: %v", err)
	}
	changedVersionCleanup()

	if base == changedContent {
		t.Fatalf("expected content change to produce new init script path, got %q", base)
	}
	if base == changedVersion {
		t.Fatalf("expected version change to produce new init script path, got %q", base)
	}
}

type scriptedRunner struct {
	responses map[string]runResult
	calls     []string
}

type runResult struct {
	stdout string
	stderr string
	err    error
}

func (r *scriptedRunner) Run(_ context.Context, dir string, _ string, args ...string) (string, string, error) {
	key := dir
	for i := 0; i < len(args)-1; i++ {
		if args[i] == "-p" {
			key = args[i+1]
			break
		}
	}
	r.calls = append(r.calls, strings.Join(args, " "))
	if res, ok := r.responses[key]; ok {
		return res.stdout, res.stderr, res.err
	}
	return "", "", nil
}

func (r *scriptedRunner) LookPath(_ string) (string, error) {
	return "gradle", nil
}

type stubResolver struct {
	results map[string]ResolveResult
	calls   []string
}

func (s *stubResolver) ResolveOnce(_ context.Context, _ executil.Runner, opts ResolveOptions) (ResolveResult, error) {
	s.calls = append(s.calls, opts.ProjectDir)
	if res, ok := s.results[opts.ProjectDir]; ok {
		return res, nil
	}
	return ResolveResult{}, nil
}

func outputRecordLine(t *testing.T, record outputRecord) string {
	t.Helper()
	encoded, err := json.Marshal(record)
	if err != nil {
		t.Fatalf("marshal output record: %v", err)
	}
	return recordPrefix + string(encoded) + "\n"
}

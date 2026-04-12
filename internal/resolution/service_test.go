package resolution

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type failingRunner struct {
	stderr string
}

func (f failingRunner) Run(_ context.Context, _ string, _ string, _ ...string) (string, string, error) {
	return "", f.stderr, errors.New("exit status 1")
}

func (f failingRunner) LookPath(_ string) (string, error) {
	return "gradle", nil
}

type runResult struct {
	stdout string
	stderr string
	err    error
}

type scriptedRunner struct {
	calls   int
	results []runResult
}

func (s *scriptedRunner) Run(_ context.Context, _ string, _ string, _ ...string) (string, string, error) {
	if s.calls >= len(s.results) {
		return "", "", errors.New("unexpected runner call")
	}
	result := s.results[s.calls]
	s.calls++
	return result.stdout, result.stderr, result.err
}

func (s *scriptedRunner) LookPath(_ string) (string, error) {
	return "gradle", nil
}

func TestExecutePlanStopsAfterFirstResult(t *testing.T) {
	runner := &scriptedRunner{
		results: []runResult{
			{stdout: ""},
			{stdout: gradleRecordLine(t, gradleRecord{Type: "source", Group: "com.example", Artifact: "demo", Version: "1.0.0", Path: "/tmp/demo-1.0.0-sources.jar"})},
		},
	}

	service := Service{Runner: runner, Verbose: true}
	plan := buildResolutionPlan(Request{Project: ".", Module: "com.example:demo:1.0.0", Scope: "compile", ApplyFilters: true}, true)
	execution, err := service.executePlan(context.Background(), plan)
	if err != nil {
		t.Fatalf("executePlan error: %v", err)
	}
	if execution.StopReason != executionFoundResults {
		t.Fatalf("expected stop reason %q, got %q", executionFoundResults, execution.StopReason)
	}
	if runner.calls != 2 {
		t.Fatalf("expected 2 runner calls, got %d", runner.calls)
	}
	if len(execution.Outcomes) != 2 {
		t.Fatalf("expected 2 outcomes, got %d", len(execution.Outcomes))
	}
	if len(execution.Outcomes[1].FilteredSources) != 1 {
		t.Fatalf("expected filtered source on second outcome, got %d", len(execution.Outcomes[1].FilteredSources))
	}
	if len(execution.Meta.Attempts) != 2 || execution.Meta.Attempts[1] != "config:*debugCompileClasspath,*DebugCompileClasspath" {
		t.Fatalf("unexpected attempts: %v", execution.Meta.Attempts)
	}
	if !strings.Contains(strings.Join(execution.Meta.Verbose, "\n"), "returned results; skipping remaining attempts") {
		t.Fatalf("expected stop verbose line, got: %v", execution.Meta.Verbose)
	}
}

func TestResolveSourcesUsesSelectorCacheFallbackWithLastDeps(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	cacheDir := filepath.Join(home, ".gradle", "caches", "modules-2", "files-2.1", "com.example", "demo", "1.0.0", "hash")
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		t.Fatalf("mkdir cache dir: %v", err)
	}
	jar := filepath.Join(cacheDir, "demo-1.0.0-sources.jar")
	if err := os.WriteFile(jar, []byte{}, 0o644); err != nil {
		t.Fatalf("write jar: %v", err)
	}

	runner := &scriptedRunner{
		results: []runResult{{
			stdout: gradleRecordLine(t, gradleRecord{Type: "dep", Group: "com.example", Artifact: "demo", Version: "1.0.0"}),
		}},
	}

	service := Service{Runner: runner}
	result, err := service.ResolveSources(context.Background(), Request{
		Project:            ".",
		Module:             "com.example:demo:1.0.0",
		Config:             "compileClasspath",
		ApplyFilters:       true,
		AllowCacheFallback: true,
	})
	if err != nil {
		t.Fatalf("ResolveSources error: %v", err)
	}
	if len(result.Sources) != 1 || result.Sources[0].Path != jar {
		t.Fatalf("expected cached source %q, got %+v", jar, result.Sources)
	}
	if len(result.Deps) != 1 || result.Deps[0].String() != "com.example:demo:1.0.0" {
		t.Fatalf("expected last deps to be preserved, got %+v", result.Deps)
	}
	if len(result.Meta.Warnings) != 0 {
		t.Fatalf("expected no warnings, got %v", result.Meta.Warnings)
	}
}

func TestResolveSourcesUsesConcreteLastDepForSelectorCacheFallback(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	cacheDir := filepath.Join(home, ".gradle", "caches", "modules-2", "files-2.1", "com.example", "demo", "1.0.0", "hash")
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		t.Fatalf("mkdir cache dir: %v", err)
	}
	jar := filepath.Join(cacheDir, "demo-1.0.0-sources.jar")
	if err := os.WriteFile(jar, []byte{}, 0o644); err != nil {
		t.Fatalf("write jar: %v", err)
	}

	runner := &scriptedRunner{
		results: []runResult{{
			stdout: gradleRecordLine(t, gradleRecord{Type: "dep", Group: "com.example", Artifact: "demo", Version: "1.0.0"}),
		}},
	}

	service := Service{Runner: runner}
	result, err := service.ResolveSources(context.Background(), Request{
		Project:            ".",
		Module:             "com.example:demo",
		Config:             "compileClasspath",
		ApplyFilters:       true,
		AllowCacheFallback: true,
	})
	if err != nil {
		t.Fatalf("ResolveSources error: %v", err)
	}
	if len(result.Sources) != 1 || result.Sources[0].Path != jar {
		t.Fatalf("expected cached source %q, got %+v", jar, result.Sources)
	}
	if len(result.Deps) != 1 || result.Deps[0].String() != "com.example:demo:1.0.0" {
		t.Fatalf("expected concrete last dep to be preserved, got %+v", result.Deps)
	}
}

func TestResolveSourcesPrefersFirstMatchingLastDepForSelectorCacheFallback(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	cacheDirA := filepath.Join(home, ".gradle", "caches", "modules-2", "files-2.1", "com.example", "demo", "1.0.0", "hash")
	if err := os.MkdirAll(cacheDirA, 0o755); err != nil {
		t.Fatalf("mkdir cache dir A: %v", err)
	}
	jarA := filepath.Join(cacheDirA, "demo-1.0.0-sources.jar")
	if err := os.WriteFile(jarA, []byte{}, 0o644); err != nil {
		t.Fatalf("write jar A: %v", err)
	}

	cacheDirB := filepath.Join(home, ".gradle", "caches", "modules-2", "files-2.1", "com.example", "demo-jvm", "1.0.0", "hash")
	if err := os.MkdirAll(cacheDirB, 0o755); err != nil {
		t.Fatalf("mkdir cache dir B: %v", err)
	}
	jarB := filepath.Join(cacheDirB, "demo-jvm-1.0.0-sources.jar")
	if err := os.WriteFile(jarB, []byte{}, 0o644); err != nil {
		t.Fatalf("write jar B: %v", err)
	}

	runner := &scriptedRunner{
		results: []runResult{{
			stdout: gradleRecordLine(t, gradleRecord{Type: "dep", Group: "com.example", Artifact: "demo", Version: "1.0.0"}) +
				gradleRecordLine(t, gradleRecord{Type: "dep", Group: "com.example", Artifact: "demo-jvm", Version: "1.0.0"}),
		}},
	}

	service := Service{Runner: runner}
	result, err := service.ResolveSources(context.Background(), Request{
		Project:            ".",
		Module:             "com.example:demo",
		Config:             "compileClasspath",
		ApplyFilters:       true,
		AllowCacheFallback: true,
	})
	if err != nil {
		t.Fatalf("ResolveSources error: %v", err)
	}
	if len(result.Sources) != 1 || result.Sources[0].Path != jarA {
		t.Fatalf("expected first matching last-dep source %q, got %+v", jarA, result.Sources)
	}
	if len(result.Deps) != 2 {
		t.Fatalf("expected both deps preserved, got %+v", result.Deps)
	}
	if result.Deps[0].String() != "com.example:demo:1.0.0" || result.Deps[1].String() != "com.example:demo-jvm:1.0.0" {
		t.Fatalf("unexpected dep order: %+v", result.Deps)
	}
	_ = jarB
}

func TestResolveSourcesExhaustedAttemptsUseExactSelectorCacheFallback(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	cacheDir := filepath.Join(home, ".gradle", "caches", "modules-2", "files-2.1", "com.example", "demo", "1.0.0", "hash")
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		t.Fatalf("mkdir cache dir: %v", err)
	}
	jar := filepath.Join(cacheDir, "demo-1.0.0-sources.jar")
	if err := os.WriteFile(jar, []byte{}, 0o644); err != nil {
		t.Fatalf("write jar: %v", err)
	}

	runner := &scriptedRunner{results: []runResult{{stdout: ""}, {stdout: ""}}}
	service := Service{Runner: runner}
	result, err := service.ResolveSources(context.Background(), Request{
		Project:            ".",
		Module:             "com.example:demo:1.0.0",
		Scope:              "compile",
		ApplyFilters:       true,
		AllowCacheFallback: true,
	})
	if err != nil {
		t.Fatalf("ResolveSources error: %v", err)
	}
	if runner.calls != 2 {
		t.Fatalf("expected 2 runner calls, got %d", runner.calls)
	}
	if len(result.Sources) != 1 || result.Sources[0].Path != jar {
		t.Fatalf("expected cached source %q, got %+v", jar, result.Sources)
	}
	if len(result.Deps) != 0 {
		t.Fatalf("expected no deps, got %+v", result.Deps)
	}
	if len(result.Meta.Attempts) != 2 {
		t.Fatalf("expected two attempts recorded, got %v", result.Meta.Attempts)
	}
	if len(result.Meta.Warnings) != 0 {
		t.Fatalf("expected no warnings, got %v", result.Meta.Warnings)
	}
}

func TestResolveSourcesAllMergesCacheFallbackOnGradleFailure(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	cacheDir := filepath.Join(home, ".gradle", "caches", "modules-2", "files-2.1", "com.example", "cached", "2.0.0", "hash")
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		t.Fatalf("mkdir cache dir: %v", err)
	}
	jar := filepath.Join(cacheDir, "cached-2.0.0-sources.jar")
	if err := os.WriteFile(jar, []byte{}, 0o644); err != nil {
		t.Fatalf("write jar: %v", err)
	}

	runner := &scriptedRunner{
		results: []runResult{
			{
				stdout: gradleRecordLine(t, gradleRecord{Type: "source", Group: "com.example", Artifact: "demo", Version: "1.0.0", Path: "/tmp/demo-1.0.0-sources.jar"}) +
					gradleRecordLine(t, gradleRecord{Type: "dep", Group: "com.example", Artifact: "demo", Version: "1.0.0"}),
			},
			{
				stderr: "BUILD FAILED",
				err:    errors.New("exit status 1"),
			},
		},
	}

	service := Service{Runner: runner, Verbose: true}
	result, err := service.ResolveSources(context.Background(), Request{
		Project:      ".",
		All:          true,
		Scope:        "compile",
		ApplyFilters: true,
	})
	if err != nil {
		t.Fatalf("ResolveSources error: %v", err)
	}
	if len(result.Sources) != 2 {
		t.Fatalf("expected 2 sources, got %d", len(result.Sources))
	}
	seen := make(map[string]struct{}, len(result.Sources))
	for _, source := range result.Sources {
		seen[source.Path] = struct{}{}
	}
	if _, ok := seen["/tmp/demo-1.0.0-sources.jar"]; !ok {
		t.Fatalf("expected gradle source in merged result")
	}
	if _, ok := seen[jar]; !ok {
		t.Fatalf("expected cache source in merged result")
	}
	if len(result.Deps) != 2 {
		t.Fatalf("expected 2 deps, got %d", len(result.Deps))
	}
	if len(result.Meta.Warnings) == 0 || !strings.Contains(result.Meta.Warnings[0], "Gradle failed") {
		t.Fatalf("expected gradle warning, got %v", result.Meta.Warnings)
	}
	if !strings.Contains(strings.Join(result.Meta.Verbose, "\n"), "Merged cache fallback into --all results.") {
		t.Fatalf("expected merge verbose line, got %v", result.Meta.Verbose)
	}
}

func TestResolveSourcesAllKeepsMergedResultsWhenCacheFallbackMisses(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	runner := &scriptedRunner{
		results: []runResult{
			{
				stdout: gradleRecordLine(t, gradleRecord{Type: "source", Group: "com.example", Artifact: "demo", Version: "1.0.0", Path: "/tmp/demo-1.0.0-sources.jar"}) +
					gradleRecordLine(t, gradleRecord{Type: "dep", Group: "com.example", Artifact: "demo", Version: "1.0.0"}),
			},
			{
				stderr: "BUILD FAILED",
				err:    errors.New("exit status 1"),
			},
		},
	}

	service := Service{Runner: runner, Verbose: true}
	result, err := service.ResolveSources(context.Background(), Request{
		Project:      ".",
		All:          true,
		Scope:        "compile",
		ApplyFilters: true,
	})
	if err != nil {
		t.Fatalf("ResolveSources error: %v", err)
	}
	if len(result.Sources) != 1 || result.Sources[0].Path != "/tmp/demo-1.0.0-sources.jar" {
		t.Fatalf("expected merged gradle source to survive fallback miss, got %+v", result.Sources)
	}
	if len(result.Deps) != 1 || result.Deps[0].String() != "com.example:demo:1.0.0" {
		t.Fatalf("expected merged gradle deps to survive fallback miss, got %+v", result.Deps)
	}
	if !containsLine(result.Meta.Warnings, "Cache-only fallback failed:") {
		t.Fatalf("expected cache fallback warning, got %v", result.Meta.Warnings)
	}
	if !containsLine(result.Meta.Verbose, "Merged cache fallback into --all results.") {
		t.Fatalf("expected merge verbose line, got %v", result.Meta.Verbose)
	}
}

func TestResolveSourcesMetaOrderForCompileFallbackAttempt(t *testing.T) {
	runner := &scriptedRunner{
		results: []runResult{
			{stdout: ""},
			{stdout: gradleRecordLine(t, gradleRecord{Type: "source", Group: "com.example", Artifact: "demo", Version: "1.0.0", Path: "/tmp/demo-1.0.0-sources.jar"})},
		},
	}

	service := Service{Runner: runner, Verbose: true}
	result, err := service.ResolveSources(context.Background(), Request{
		Project:      ".",
		Module:       "com.example:demo:1.0.0",
		Scope:        "compile",
		ApplyFilters: true,
	})
	if err != nil {
		t.Fatalf("ResolveSources error: %v", err)
	}
	if len(result.Meta.Attempts) != 2 {
		t.Fatalf("expected 2 attempts, got %v", result.Meta.Attempts)
	}
	if result.Meta.Attempts[0] != "default" || result.Meta.Attempts[1] != "config:*debugCompileClasspath,*DebugCompileClasspath" {
		t.Fatalf("unexpected attempt order: %v", result.Meta.Attempts)
	}
	expectedPatterns := []string{"*debugCompileClasspath", "*DebugCompileClasspath"}
	if strings.Join(result.Meta.TriedConfigPatterns, ",") != strings.Join(expectedPatterns, ",") {
		t.Fatalf("unexpected config patterns: %v", result.Meta.TriedConfigPatterns)
	}
	assertOrderedSubsequence(t, result.Meta.Verbose, []string{
		"Resolve attempts: default, config:*debugCompileClasspath,*DebugCompileClasspath",
		"Attempt default: configs=(default)",
		"Attempt default result: sources=0 filtered=0 deps=0",
		"Attempt default had no results; continuing.",
		"Attempt config:*debugCompileClasspath,*DebugCompileClasspath: configs=*debugCompileClasspath,*DebugCompileClasspath",
		"Attempt config:*debugCompileClasspath,*DebugCompileClasspath result: sources=1 filtered=1 deps=0",
		"Attempt config:*debugCompileClasspath,*DebugCompileClasspath returned results; skipping remaining attempts.",
	})
}

func TestResolveSourcesMetaIncludesDetectedIncludedBuilds(t *testing.T) {
	included := filepath.Join(t.TempDir(), "build-logic")
	runner := &scriptedRunner{
		results: []runResult{
			{stdout: gradleRecordLine(t, gradleRecord{Type: "include", Path: included})},
			{stdout: gradleRecordLine(t, gradleRecord{Type: "include", Path: included})},
		},
	}

	service := Service{Runner: runner}
	result, err := service.ResolveSources(context.Background(), Request{
		Project:            ".",
		Module:             "com.example:demo:1.0.0",
		Scope:              "compile",
		ApplyFilters:       true,
		AllowCacheFallback: false,
	})
	if err != nil {
		t.Fatalf("ResolveSources error: %v", err)
	}
	if len(result.Meta.IncludedBuilds) != 1 || result.Meta.IncludedBuilds[0] != included {
		t.Fatalf("expected included build metadata %q, got %v", included, result.Meta.IncludedBuilds)
	}
}

type gradleRecord struct {
	Type     string `json:"type"`
	Group    string `json:"group,omitempty"`
	Artifact string `json:"artifact,omitempty"`
	Version  string `json:"version,omitempty"`
	Path     string `json:"path,omitempty"`
}

func gradleRecordLine(t *testing.T, payload gradleRecord) string {
	t.Helper()
	encoded, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal gradle record: %v", err)
	}
	return "KSRCJSON\t" + string(encoded) + "\n"
}

func containsLine(lines []string, needle string) bool {
	for _, line := range lines {
		if strings.Contains(line, needle) {
			return true
		}
	}
	return false
}

func assertOrderedSubsequence(t *testing.T, lines []string, want []string) {
	t.Helper()
	index := 0
	for _, line := range lines {
		if index < len(want) && strings.Contains(line, want[index]) {
			index++
		}
	}
	if index == len(want) {
		return
	}
	t.Fatalf("expected ordered subsequence %v in %v", want, lines)
}

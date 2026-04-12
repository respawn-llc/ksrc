package cli

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/respawn-app/ksrc/internal/resolution"
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
	res := s.results[s.calls]
	s.calls++
	return res.stdout, res.stderr, res.err
}

func (s *scriptedRunner) LookPath(_ string) (string, error) {
	return "gradle", nil
}

func TestResolveSourcesFallsBackToCacheOnGradleFailure(t *testing.T) {
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

	app := &App{Runner: failingRunner{stderr: "BUILD FAILED"}}
	flags := ResolveFlags{
		Project: ".",
		Module:  "com.example:demo:1.0.0",
	}
	sources, _, meta, err := resolveSources(context.Background(), app, flags, "", true, true)
	if err != nil {
		t.Fatalf("resolveSources error: %v", err)
	}
	if len(sources) != 1 {
		t.Fatalf("expected 1 source, got %d", len(sources))
	}
	if len(meta.Warnings) == 0 || !strings.Contains(meta.Warnings[0], "Gradle failed") {
		t.Fatalf("expected Gradle failed warning, got: %v", meta.Warnings)
	}
	if strings.Contains(strings.Join(meta.Warnings, " "), "BUILD FAILED") {
		t.Fatalf("did not expect Gradle output in warnings when not verbose")
	}
}

func TestResolveSourcesCacheFallbackUsesVersionWhenProvided(t *testing.T) {
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
		results: []runResult{
			{stdout: ""},
		},
	}
	app := &App{Runner: runner}
	flags := ResolveFlags{
		Project: ".",
		Module:  "com.example:demo:1.0.0",
		Config:  "compileClasspath",
	}
	sources, _, meta, err := resolveSources(context.Background(), app, flags, "", true, true)
	if err != nil {
		t.Fatalf("resolveSources error: %v", err)
	}
	if len(sources) != 1 {
		t.Fatalf("expected 1 source, got %d", len(sources))
	}
	if len(meta.Warnings) != 0 {
		t.Fatalf("expected no warnings, got: %v", meta.Warnings)
	}
}

func TestResolveSourcesAllKeepsMergedResultsOnGradleFailure(t *testing.T) {
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
	app := &App{Runner: runner}
	flags := ResolveFlags{
		Project: ".",
		All:     true,
		Scope:   "compile",
	}
	sources, deps, meta, err := resolveSources(context.Background(), app, flags, "", true, true)
	if err != nil {
		t.Fatalf("resolveSources error: %v", err)
	}
	if len(sources) != 2 {
		t.Fatalf("expected 2 sources, got %d", len(sources))
	}
	seen := make(map[string]struct{})
	for _, s := range sources {
		seen[s.Path] = struct{}{}
	}
	if _, ok := seen["/tmp/demo-1.0.0-sources.jar"]; !ok {
		t.Fatalf("expected gradle source to be preserved")
	}
	if _, ok := seen[jar]; !ok {
		t.Fatalf("expected cache source to be included")
	}
	if len(deps) != 2 {
		t.Fatalf("expected 2 deps, got %d", len(deps))
	}
	if len(meta.Warnings) == 0 || !strings.Contains(meta.Warnings[0], "Gradle failed") {
		t.Fatalf("expected Gradle failed warning, got: %v", meta.Warnings)
	}
}

func TestNoSourcesHintForFlagsUsesIncludedBuildMetaOnly(t *testing.T) {
	included := filepath.Join(".", "build-logic")
	hint := noSourcesHintForFlags(ResolveFlags{All: true}, resolution.ResolveMeta{IncludedBuilds: []string{included}})

	if !strings.Contains(hint, "Composite build detected; try: --project build-logic") {
		t.Fatalf("expected composite build hint, got %q", hint)
	}
	if strings.Contains(hint, "Android detected") || strings.Contains(hint, "KMP detected") {
		t.Fatalf("expected no build-script heuristics in hint, got %q", hint)
	}
	if strings.Contains(hint, "If resolution is slow") {
		t.Fatalf("expected no generic narrowing hint, got %q", hint)
	}
}

func TestResolveNoSourcesAfterGradleFailureHasNoCompositeHintWithoutTraversalMeta(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	app := &App{Runner: failingRunner{stderr: "BUILD FAILED"}}
	output, err := runCommand(app, []string{"resolve", "--project", ".", "--module", "com.example:missing:1.0.0"})
	if err == nil {
		t.Fatalf("expected resolve error, got output %q", output)
	}
	if !strings.Contains(err.Error(), "E_NO_SOURCES") {
		t.Fatalf("expected E_NO_SOURCES, got %v", err)
	}
	if strings.Contains(err.Error(), "Composite build detected") {
		t.Fatalf("did not expect composite build hint without traversal metadata: %v", err)
	}
	if !strings.Contains(output, "WARN: Gradle failed") {
		t.Fatalf("expected Gradle warning in output, got %q", output)
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

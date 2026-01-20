package cli

import (
	"context"
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
				stdout: "KSRC|com.example:demo:1.0.0|/tmp/demo-1.0.0-sources.jar\nKSRCDEP|com.example:demo:1.0.0\n",
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

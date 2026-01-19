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

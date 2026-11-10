package mcpserver

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/respawn-app/ksrc/internal/adapter"
)

func TestBuildDepsSpecCarriesPlatformSelectorBeforeResolve(t *testing.T) {
	sdk := t.TempDir()
	source := filepath.Join(sdk, "sources", "android-36", "android", "app", "Activity.java")
	if err := os.MkdirAll(filepath.Dir(source), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(source, []byte("public class Activity extends ContextThemeWrapper {\n}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	project := t.TempDir()
	spec := buildDepsSpec(DepsInput{
		Project:     project,
		Group:       "sdk.android",
		Artifact:    "platform",
		Version:     "android-36",
		AndroidHome: sdk,
	})
	if spec.Group != "sdk.android" || spec.Artifact != "platform" || spec.Version != "android-36" {
		t.Fatalf("selector not on spec before resolve: %#v", spec)
	}

	result, err := (adapter.Resolver{}).ResolveSources(context.Background(), spec)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Sources) != 1 || result.Sources[0].Path != filepath.Join(sdk, "sources", "android-36") {
		t.Fatalf("resolved roots = %#v warnings=%v", result.Sources, result.Meta.Warnings)
	}
	if result.Deps[0].String() != "sdk.android:platform:android-36" {
		t.Fatalf("deps = %#v", result.Deps)
	}
}

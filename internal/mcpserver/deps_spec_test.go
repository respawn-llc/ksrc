package mcpserver

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
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

func TestHandleDepsNumericPlatformVersion(t *testing.T) {
	sdk := t.TempDir()
	source := filepath.Join(sdk, "sources", "android-36", "android", "app", "Activity.java")
	if err := os.MkdirAll(filepath.Dir(source), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(source, []byte("public class Activity extends ContextThemeWrapper {\n}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(map[string]any{
		"group":       "sdk.android",
		"artifact":    "platform",
		"version":     "36",
		"project":     t.TempDir(),
		"androidHome": sdk,
	})
	if err != nil {
		t.Fatal(err)
	}
	result, err := (&toolState{}).handleDeps(context.Background(), &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{Arguments: raw},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("deps error: %#v", result)
	}
	text := result.Content[0].(*mcp.TextContent).Text
	if !strings.Contains(text, "sdk.android:platform:android-36") || !strings.Contains(text, filepath.Join(sdk, "sources", "android-36")) {
		t.Fatalf("deps = %s", text)
	}
}

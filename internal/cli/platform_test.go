package cli

import (
	"archive/zip"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/respawn-app/ksrc/internal/executil"
)

type rgOnlyRunner struct {
	executil.OSRunner
}

func (r rgOnlyRunner) Run(ctx context.Context, dir string, name string, args ...string) (string, string, error) {
	if name != "rg" {
		return "", "", fmt.Errorf("unexpected command %s", name)
	}
	return r.OSRunner.Run(ctx, dir, name, args...)
}

func TestSearchAndroidPlatformWithoutGradle(t *testing.T) {
	sdk := writeAndroidSDK(t, true)
	app := NewApp()
	app.Runner = rgOnlyRunner{}
	out, err := runCommand(app, []string{
		"search", "class Activity",
		"--module", "sdk.android:platform:android-36",
		"--android-home", sdk,
		"--project", t.TempDir(),
	})
	if err != nil {
		t.Fatalf("search error: %v\n%s", err, out)
	}
	if !strings.Contains(out, "sdk.android:platform:android-36!/android/app/Activity.java") {
		t.Fatalf("missing file-id:\n%s", out)
	}
	if strings.Contains(out, "Stub!") || strings.Contains(out, "WARN:") {
		t.Fatalf("stub or warning leaked:\n%s", out)
	}

	catOut, err := runCommand(app, []string{
		"cat", "sdk.android:platform:android-36!/android/app/Activity.java",
		"--android-home", sdk,
		"--project", t.TempDir(),
	})
	if err != nil {
		t.Fatalf("cat error: %v\n%s", err, catOut)
	}
	if !strings.Contains(catOut, "ContextThemeWrapper") || strings.Contains(catOut, "Stub!") {
		t.Fatalf("cat = %s", catOut)
	}
}

func TestSearchAndroidPlatformStubSourcesWarn(t *testing.T) {
	sdk := writeAndroidSDK(t, false)
	app := NewApp()
	app.Runner = rgOnlyRunner{}
	out, err := runCommand(app, []string{
		"search", "Stub!",
		"--module", "sdk.android:platform:36",
		"--android-home", sdk,
		"--project", t.TempDir(),
	})
	if err != nil {
		t.Fatalf("search error: %v\n%s", err, out)
	}
	if !strings.Contains(out, "sdk.android:platform:android-36!/android/app/Activity.java") {
		t.Fatalf("missing file-id:\n%s", out)
	}
	if !strings.Contains(out, "WARN:") || !strings.Contains(out, "android sdk install sources/android-36") {
		t.Fatalf("missing install hint:\n%s", out)
	}

	deps, err := runCommand(app, []string{
		"deps",
		"--module", "sdk.android:platform:android-36",
		"--android-home", sdk,
		"--project", t.TempDir(),
	})
	if err != nil {
		t.Fatalf("deps error: %v\n%s", err, deps)
	}
	if !strings.Contains(deps, "[sources: stubs]") {
		t.Fatalf("deps = %s", deps)
	}
}

func TestAndroidPlatformRequiresVersion(t *testing.T) {
	app := NewApp()
	app.Runner = rgOnlyRunner{}
	out, err := runCommand(app, []string{
		"search", "class Activity",
		"--module", "sdk.android:platform",
		"--project", t.TempDir(),
	})
	if err == nil || !strings.Contains(err.Error()+out, "android-36") {
		t.Fatalf("error = %v\n%s", err, out)
	}
}

func writeAndroidSDK(t *testing.T, withSources bool) string {
	t.Helper()
	sdk := t.TempDir()
	if withSources {
		path := filepath.Join(sdk, "sources", "android-36", "android", "app", "Activity.java")
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		body := "public class Activity extends ContextThemeWrapper {\n}\n"
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	jarPath := filepath.Join(sdk, "platforms", "android-36", "android-stubs-src.jar")
	if err := os.MkdirAll(filepath.Dir(jarPath), 0o755); err != nil {
		t.Fatal(err)
	}
	file, err := os.Create(jarPath)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	zw := zip.NewWriter(file)
	entry, err := zw.Create("android/app/Activity.java")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := entry.Write([]byte("public class Activity { throw new RuntimeException(\"Stub!\"); }\n")); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return sdk
}

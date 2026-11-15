package cli

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/respawn-llc/ksrc/internal/executil"
	"github.com/respawn-llc/ksrc/internal/fileidcache"
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
	stubOnly, err := runCommand(app, []string{
		"search", "Stub!",
		"--module", "sdk.android:platform:android-36",
		"--android-home", sdk,
		"--project", t.TempDir(),
	})
	if err != nil {
		t.Fatalf("stub-only search error: %v\n%s", err, stubOnly)
	}
	if strings.Contains(stubOnly, "Stub!") || strings.Contains(stubOnly, "Activity.java") {
		t.Fatalf("stub-only text was a hit:\n%s", stubOnly)
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
	if !strings.Contains(deps, "android-stubs-src.jar") || !strings.Contains(deps, "android sdk install sources/android-36") {
		t.Fatalf("deps = %s", deps)
	}
}

func TestAndroidPlatformNumericVersionMatchesNormalizedRoot(t *testing.T) {
	sdk := writeAndroidSDK(t, true)
	app := NewApp()
	app.Runner = rgOnlyRunner{}
	project := t.TempDir()
	fileID := "sdk.android:platform:36!/android/app/Activity.java"

	catOut, err := runCommand(app, []string{"cat", fileID, "--android-home", sdk, "--project", project})
	if err != nil {
		t.Fatalf("cat error: %v\n%s", err, catOut)
	}
	if !strings.Contains(catOut, "ContextThemeWrapper") || strings.Contains(catOut, "Stub!") {
		t.Fatalf("cat = %s", catOut)
	}

	whereCoord, err := runCommand(app, []string{"where", "sdk.android:platform:36", "--android-home", sdk, "--project", project})
	if err != nil {
		t.Fatalf("where coord error: %v\n%s", err, whereCoord)
	}
	if !strings.Contains(whereCoord, filepath.Join(sdk, "sources", "android-36")) {
		t.Fatalf("where coord = %s", whereCoord)
	}

	whereFile, err := runCommand(app, []string{"where", fileID, "--android-home", sdk, "--project", project})
	if err != nil {
		t.Fatalf("where file-id error: %v\n%s", err, whereFile)
	}
	if !strings.Contains(whereFile, "sdk.android:platform:android-36!/android/app/Activity.java") || !strings.Contains(whereFile, filepath.Join(sdk, "sources", "android-36")) {
		t.Fatalf("where file-id = %s", whereFile)
	}

	fetchOut, err := runCommand(app, []string{"fetch", "sdk.android:platform:36", "--android-home", sdk, "--project", project})
	if err != nil {
		t.Fatalf("fetch error: %v\n%s", err, fetchOut)
	}
	if !strings.Contains(fetchOut, "sdk.android:platform:android-36|") || !strings.Contains(fetchOut, filepath.Join(sdk, "sources", "android-36")) {
		t.Fatalf("fetch = %s", fetchOut)
	}
}

func TestCatUsesFileIDCacheWithoutRediscoveringSDK(t *testing.T) {
	t.Setenv("ANDROID_HOME", "")
	t.Setenv("ANDROID_SDK_ROOT", "")
	sdk := writeAndroidSDK(t, true)
	app := NewApp()
	app.Runner = rgOnlyRunner{}
	searchOut, err := runCommand(app, []string{
		"search", "ContextThemeWrapper",
		"--module", "sdk.android:platform:android-36",
		"--android-home", sdk,
		"--project", t.TempDir(),
	})
	if err != nil {
		t.Fatalf("search error: %v\n%s", err, searchOut)
	}
	fileID := "sdk.android:platform:android-36!/android/app/Activity.java"
	if !strings.Contains(searchOut, fileID) {
		t.Fatalf("search = %s", searchOut)
	}
	catOut, err := runCommand(app, []string{"cat", fileID})
	if err != nil {
		t.Fatalf("cached cat error: %v\n%s", err, catOut)
	}
	if !strings.Contains(catOut, "ContextThemeWrapper") || strings.Contains(catOut, "Stub!") {
		t.Fatalf("cached cat = %s", catOut)
	}
}

func TestAndroidPlatformRelativeSDKWithDifferentProject(t *testing.T) {
	t.Setenv("ANDROID_HOME", "")
	t.Setenv("ANDROID_SDK_ROOT", "")
	sdk := writeAndroidSDK(t, true)
	invocation := t.TempDir()
	project := t.TempDir()
	t.Chdir(invocation)
	relative, err := filepath.Rel(invocation, sdk)
	if err != nil {
		t.Fatal(err)
	}
	app := NewApp()
	app.Runner = rgOnlyRunner{}
	out, err := runCommand(app, []string{
		"search", "ContextThemeWrapper", "--module", "sdk.android:platform:36",
		"--android-home", relative, "--project", project,
	})
	fileID := "sdk.android:platform:android-36!/android/app/Activity.java"
	if err != nil || !strings.Contains(out, fileID) {
		t.Fatalf("relative SDK search failed: %v\n%s", err, out)
	}
	t.Chdir(project)
	out, err = runCommand(app, []string{"cat", fileID})
	if err != nil || !strings.Contains(out, "ContextThemeWrapper") {
		t.Fatalf("file-id followup from another cwd failed: %v\n%s", err, out)
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

func TestCachedAndroidPlatformFollowupWarnsAfterSourcesUninstalled(t *testing.T) {
	for _, command := range []string{"cat", "open", "where"} {
		t.Run(command, func(t *testing.T) {
			t.Setenv("ANDROID_HOME", "")
			t.Setenv("ANDROID_SDK_ROOT", "")
			sdk := writeAndroidSDK(t, true)
			fullSources := filepath.Join(sdk, "sources", "android-36")
			fileID := "sdk.android:platform:android-36!/android/app/Activity.java"
			if err := fileidcache.Register(fileID, fullSources); err != nil {
				t.Fatal(err)
			}
			uninstalled := filepath.Join(t.TempDir(), "android-36")
			if err := os.Rename(fullSources, uninstalled); err != nil {
				t.Fatal(err)
			}
			app := NewApp()
			app.Runner = rgOnlyRunner{}
			run := func() (string, string) {
				t.Helper()
				cmd := NewRootCommand(app)
				cmd.SetArgs([]string{command, fileID})
				var stdout, stderr bytes.Buffer
				cmd.SetOut(&stdout)
				cmd.SetErr(&stderr)
				if err := cmd.Execute(); err != nil {
					t.Fatalf("%s error: %v\n%s", command, err, stderr.String())
				}
				return stdout.String(), stderr.String()
			}
			out, diagnostics := run()
			want := "Stub!"
			if command == "where" {
				want = filepath.Join(sdk, "platforms", "android-36", "android-stubs-src.jar")
			}
			if !strings.Contains(out, want) {
				t.Errorf("missing fallback %q:\n%s", want, out)
			}
			if !strings.Contains(diagnostics, "WARN:") || !strings.Contains(diagnostics, "android sdk install sources/android-36") {
				t.Errorf("missing fallback warning and install hint on stderr:\n%s", diagnostics)
			}
			if strings.Contains(out, "WARN:") {
				t.Errorf("warning leaked into stdout:\n%s", out)
			}

			if err := os.Rename(uninstalled, fullSources); err != nil {
				t.Fatal(err)
			}
			out, diagnostics = run()
			want = "ContextThemeWrapper"
			if command == "where" {
				want = fullSources
			}
			if !strings.Contains(out, want) || strings.Contains(out, "Stub!") {
				t.Errorf("reinstalled sources not selected:\n%s", out)
			}
			if strings.Contains(diagnostics, "WARN:") || strings.Contains(diagnostics, "android sdk install") {
				t.Errorf("stale fallback warning after reinstall:\n%s", diagnostics)
			}
		})
	}
}

func writeAndroidSDK(t *testing.T, withSources bool) string {
	t.Helper()
	t.Setenv("KSRC_FILEID_CACHE_DIR", t.TempDir())
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

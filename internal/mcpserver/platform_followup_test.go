package mcpserver

import (
	"archive/zip"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/respawn-app/ksrc/internal/fileidcache"
)

func TestCachedAndroidPlatformFollowupWarnsAfterSourcesUninstalled(t *testing.T) {
	for _, tool := range []string{"cat", "where"} {
		t.Run(tool, func(t *testing.T) {
			t.Setenv("KSRC_FILEID_CACHE_DIR", t.TempDir())
			t.Setenv("ANDROID_HOME", "")
			t.Setenv("ANDROID_SDK_ROOT", "")
			sdk := t.TempDir()
			fullSources := filepath.Join(sdk, "sources", "android-36")
			source := filepath.Join(fullSources, "android", "app", "Activity.java")
			if err := os.MkdirAll(filepath.Dir(source), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(source, []byte("public class Activity extends ContextThemeWrapper {}\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			stub := filepath.Join(sdk, "platforms", "android-36", "android-stubs-src.jar")
			if err := os.MkdirAll(filepath.Dir(stub), 0o755); err != nil {
				t.Fatal(err)
			}
			writePlatformFollowupStub(t, stub)
			fileID := "sdk.android:platform:android-36!/android/app/Activity.java"
			if err := fileidcache.Register(fileID, fullSources); err != nil {
				t.Fatal(err)
			}
			uninstalled := filepath.Join(t.TempDir(), "android-36")
			if err := os.Rename(fullSources, uninstalled); err != nil {
				t.Fatal(err)
			}
			state := &toolState{}
			handler := state.handleCat
			key := "fileId"
			if tool == "where" {
				handler = state.handleWhere
				key = "pathOrCoord"
			}
			raw, err := json.Marshal(map[string]string{key: fileID})
			if err != nil {
				t.Fatal(err)
			}
			run := func() (string, string) {
				t.Helper()
				// A file avoids pipe backpressure and capture goroutines. This test
				// must remain nonparallel because MCP diagnostics use os.Stderr.
				log, err := os.CreateTemp(t.TempDir(), "stderr")
				if err != nil {
					t.Fatal(err)
				}
				defer log.Close()
				previous := os.Stderr
				os.Stderr = log
				defer func() { os.Stderr = previous }()
				result, err := handler(context.Background(), &mcp.CallToolRequest{
					Params: &mcp.CallToolParamsRaw{Arguments: raw},
				})
				if err != nil {
					t.Fatal(err)
				}
				if result.IsError {
					t.Fatalf("%s error: %#v", tool, result.Content)
				}
				diagnostics, err := os.ReadFile(log.Name())
				if err != nil {
					t.Fatal(err)
				}
				return result.Content[0].(*mcp.TextContent).Text, string(diagnostics)
			}
			out, diagnostics := run()
			want := "Stub!"
			if tool == "where" {
				want = stub
			}
			if !strings.Contains(out, want) {
				t.Errorf("missing fallback %q:\n%s", want, out)
			}
			if !strings.Contains(diagnostics, "WARN:") || !strings.Contains(diagnostics, "android sdk install sources/android-36") {
				t.Errorf("missing fallback warning and install hint on stderr:\n%s", diagnostics)
			}
			if strings.Contains(out, "WARN:") {
				t.Errorf("warning leaked into tool content:\n%s", out)
			}

			if err := os.Rename(uninstalled, fullSources); err != nil {
				t.Fatal(err)
			}
			out, diagnostics = run()
			want = "ContextThemeWrapper"
			if tool == "where" {
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

func writePlatformFollowupStub(t *testing.T, path string) {
	t.Helper()
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	writer := zip.NewWriter(file)
	entry, err := writer.Create("android/app/Activity.java")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := entry.Write([]byte("public class Activity { throw new RuntimeException(\"Stub!\"); }\n")); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
}

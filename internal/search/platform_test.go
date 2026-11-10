package search

import (
	"archive/zip"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/respawn-app/ksrc/internal/executil"
	"github.com/respawn-app/ksrc/internal/resolve"
)

func TestSearchPlatformRootOmitsStubOnlyText(t *testing.T) {
	sdk := t.TempDir()
	sourcePath := filepath.Join(sdk, "sources", "android-36", "android", "app", "Activity.java")
	if err := os.MkdirAll(filepath.Dir(sourcePath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(sourcePath, []byte("public class Activity extends ContextThemeWrapper {\n}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	jarPath := filepath.Join(sdk, "platforms", "android-36", "android-stubs-src.jar")
	if err := os.MkdirAll(filepath.Dir(jarPath), 0o755); err != nil {
		t.Fatal(err)
	}
	file, err := os.Create(jarPath)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(file)
	entry, err := zw.Create("android/app/Activity.java")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := entry.Write([]byte("public class Activity { throw new RuntimeException(\"StubOnlyToken\"); }\n")); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}

	loaded, err := resolve.LoadPlatform(sdk, "android-36")
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded.Sources) != 1 {
		t.Fatalf("roots = %#v", loaded.Sources)
	}
	stubHits, err := Run(context.Background(), executil.OSRunner{}, Options{
		Pattern: "StubOnlyToken",
		Jars:    loaded.Sources,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(stubHits) != 0 {
		t.Fatalf("stub-only text matched: %#v", stubHits)
	}
	fullHits, err := Run(context.Background(), executil.OSRunner{}, Options{
		Pattern: "ContextThemeWrapper",
		Jars:    loaded.Sources,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(fullHits) != 1 || fullHits[0].FileID != "sdk.android:platform:android-36!/android/app/Activity.java" {
		t.Fatalf("full hits = %#v", fullHits)
	}
}

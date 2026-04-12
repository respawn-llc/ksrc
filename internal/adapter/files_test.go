package adapter

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"

	"github.com/respawn-app/ksrc/internal/resolve"
)

func TestResolveFileIDLocation(t *testing.T) {
	jarPath := filepath.Join(t.TempDir(), "demo-sources.jar")
	coord := resolve.Coord{Group: "com.example", Artifact: "demo", Version: "1.0.0"}
	inner := "com/example/Demo.kt"
	if err := writeTestJar(jarPath, inner, "class Demo\n"); err != nil {
		t.Fatalf("write jar: %v", err)
	}

	location, err := ResolveFileIDLocation([]resolve.SourceJar{{Coord: coord, Path: jarPath}}, resolve.FormatFileID(coord, "/"+inner), "")
	if err != nil {
		t.Fatalf("ResolveFileIDLocation error: %v", err)
	}
	if location.Source.Path != jarPath {
		t.Fatalf("unexpected jar path: %s", location.Source.Path)
	}
	if location.InnerPath != inner {
		t.Fatalf("unexpected inner path: %s", location.InnerPath)
	}
	if location.FileID != resolve.FormatFileID(coord, inner) {
		t.Fatalf("unexpected file id: %s", location.FileID)
	}
}

func TestResolvePathLocationNormalizesLeadingSlash(t *testing.T) {
	jarPath := filepath.Join(t.TempDir(), "demo-sources.jar")
	coord := resolve.Coord{Group: "com.example", Artifact: "demo", Version: "1.0.0"}
	inner := "com/example/Demo.kt"
	if err := writeTestJar(jarPath, inner, "class Demo\n"); err != nil {
		t.Fatalf("write jar: %v", err)
	}

	location, err := ResolvePathLocation([]resolve.SourceJar{{Coord: coord, Path: jarPath}}, "/"+inner, "")
	if err != nil {
		t.Fatalf("ResolvePathLocation error: %v", err)
	}
	if location.InnerPath != inner {
		t.Fatalf("unexpected inner path: %s", location.InnerPath)
	}
	if location.FileID != resolve.FormatFileID(coord, inner) {
		t.Fatalf("unexpected file id: %s", location.FileID)
	}
}

func TestResolvePathLocationIncludesHint(t *testing.T) {
	_, err := ResolvePathLocation(nil, "/missing/File.kt", "Try: search first")
	if err == nil {
		t.Fatal("expected error")
	}
	want := "file not found in resolved sources: missing/File.kt. Try: search first"
	if err.Error() != want {
		t.Fatalf("unexpected error: %q", err)
	}
}

func writeTestJar(path, inner, content string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	zw := zip.NewWriter(f)
	w, err := zw.Create(inner)
	if err != nil {
		_ = zw.Close()
		_ = f.Close()
		return err
	}
	if _, err := w.Write([]byte(content)); err != nil {
		_ = zw.Close()
		_ = f.Close()
		return err
	}
	if err := zw.Close(); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}

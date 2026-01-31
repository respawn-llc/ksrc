package cli

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"

	"github.com/respawn-app/ksrc/internal/resolve"
)

func TestFindFileInJarsAllowsEmptyFile(t *testing.T) {
	jarPath := filepath.Join(t.TempDir(), "empty.jar")
	inner := "com/example/Empty.kt"
	if err := writeZipFile(jarPath, inner, ""); err != nil {
		t.Fatalf("write zip: %v", err)
	}

	sources := []resolve.SourceJar{{
		Coord: resolve.Coord{Group: "com.example", Artifact: "demo", Version: "1.0.0"},
		Path:  jarPath,
	}}

	path, foundInner, err := findFileInJars(sources, inner)
	if err != nil {
		t.Fatalf("findFileInJars error: %v", err)
	}
	if path != jarPath {
		t.Fatalf("unexpected jar path: %s", path)
	}
	if foundInner != inner {
		t.Fatalf("unexpected inner path: %s", foundInner)
	}
}

func writeZipFile(path, inner, content string) error {
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

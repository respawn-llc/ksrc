package resolve

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
)

func TestFindFileInSourcesAllowsEmptyFile(t *testing.T) {
	jarPath := filepath.Join(t.TempDir(), "empty.jar")
	inner := "com/example/Empty.kt"
	if err := writeZipFile(t, jarPath, inner, ""); err != nil {
		t.Fatalf("write zip: %v", err)
	}

	sources := []SourceJar{{
		Coord: Coord{Group: "com.example", Artifact: "demo", Version: "1.0.0"},
		Path:  jarPath,
	}}

	source, foundInner, ok := FindFileInSources(sources, inner)
	if !ok {
		t.Fatal("expected file to be found")
	}
	if source.Path != jarPath {
		t.Fatalf("unexpected jar path: %s", source.Path)
	}
	if foundInner != inner {
		t.Fatalf("unexpected inner path: %s", foundInner)
	}
}

func TestFormatFileIDNormalizesInnerPath(t *testing.T) {
	got := FormatFileID(Coord{Group: "com.example", Artifact: "demo", Version: "1.0.0"}, "\\com\\example\\Demo.kt")
	want := "com.example:demo:1.0.0!/com/example/Demo.kt"
	if got != want {
		t.Fatalf("FormatFileID() = %q, want %q", got, want)
	}
}

func TestFileIDRoundTripNormalizesBackslashes(t *testing.T) {
	coord := Coord{Group: "com.example", Artifact: "demo", Version: "1.0.0"}
	fileID := FormatFileID(coord, "\\com\\example\\Demo.kt")

	parsedCoord, inner, err := ParseFileID(fileID)
	if err != nil {
		t.Fatalf("ParseFileID() error = %v", err)
	}
	if parsedCoord != coord {
		t.Fatalf("parsed coord = %#v, want %#v", parsedCoord, coord)
	}
	if inner != "com/example/Demo.kt" {
		t.Fatalf("parsed inner path = %q", inner)
	}
}

func writeZipFile(t *testing.T, path, inner, content string) error {
	t.Helper()
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

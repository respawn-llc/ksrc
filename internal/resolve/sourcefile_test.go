package resolve

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"

	"github.com/respawn-app/ksrc/internal/cat"
	"github.com/respawn-app/ksrc/internal/testutil"
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

func TestFindFileInSourcesUsesZipDirectoryEntries(t *testing.T) {
	jarPath := filepath.Join(t.TempDir(), "broken-body.jar")
	inner := "com/example/Entry.kt"
	if err := writeZipFile(t, jarPath, inner, "class Entry\n"); err != nil {
		t.Fatalf("write zip: %v", err)
	}
	if err := testutil.CorruptZipEntryMethod(jarPath, inner, 99); err != nil {
		t.Fatalf("corrupt zip entry method: %v", err)
	}
	if _, err := cat.ReadFileFromZip(jarPath, inner, nil); err == nil {
		t.Fatal("expected entry body read to fail after method rewrite")
	}

	sources := []SourceJar{{
		Coord: Coord{Group: "com.example", Artifact: "demo", Version: "1.0.0"},
		Path:  jarPath,
	}}

	source, foundInner, ok := FindFileInSources(sources, inner)
	if !ok {
		t.Fatal("expected file to be found from zip metadata")
	}
	if source.Path != jarPath {
		t.Fatalf("unexpected jar path: %s", source.Path)
	}
	if foundInner != inner {
		t.Fatalf("unexpected inner path: %s", foundInner)
	}
	if found, err := cat.HasFileInZip(jarPath, inner); err != nil || !found {
		t.Fatalf("expected HasFileInZip to find entry, found=%v err=%v", found, err)
	}
	if found, err := cat.HasFileInZip(jarPath, "missing/File.kt"); err != nil || found {
		t.Fatalf("expected HasFileInZip to miss unknown entry, found=%v err=%v", found, err)
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

func TestFileIDRoundTripNormalizesRepeatedLeadingSlashes(t *testing.T) {
	coord := Coord{Group: "com.example", Artifact: "demo", Version: "1.0.0"}
	fileID := FormatFileID(coord, "///nested/path/File.kt")
	if fileID != "com.example:demo:1.0.0!/nested/path/File.kt" {
		t.Fatalf("FormatFileID() = %q", fileID)
	}

	parsedCoord, inner, err := ParseFileID("com.example:demo:1.0.0!////nested/path/File.kt")
	if err != nil {
		t.Fatalf("ParseFileID() error = %v", err)
	}
	if parsedCoord != coord {
		t.Fatalf("parsed coord = %#v, want %#v", parsedCoord, coord)
	}
	if inner != "nested/path/File.kt" {
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

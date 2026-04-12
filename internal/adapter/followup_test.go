package adapter

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/respawn-app/ksrc/internal/resolve"
)

func TestFindFollowupFileIDLocationFallsBackWhenFileIDCacheLookupFails(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	brokenCacheRoot := filepath.Join(t.TempDir(), "cache-root-file")
	if err := os.WriteFile(brokenCacheRoot, []byte("not a directory"), 0o644); err != nil {
		t.Fatalf("write broken cache root: %v", err)
	}
	t.Setenv("KSRC_FILEID_CACHE_DIR", brokenCacheRoot)

	coord := resolve.Coord{Group: "com.example", Artifact: "demo", Version: "1.0.0"}
	inner := "com/example/Demo.kt"
	jarDir := filepath.Join(home, ".gradle", "caches", "modules-2", "files-2.1", coord.Group, coord.Artifact, coord.Version, "hash")
	if err := os.MkdirAll(jarDir, 0o755); err != nil {
		t.Fatalf("mkdir jar dir: %v", err)
	}
	jarPath := filepath.Join(jarDir, coord.Artifact+"-"+coord.Version+"-sources.jar")
	if err := writeTestJar(jarPath, inner, "class Demo\n"); err != nil {
		t.Fatalf("write jar: %v", err)
	}

	location, found, err := FindFollowupFileIDLocation(resolve.FormatFileID(coord, inner))
	if err != nil {
		t.Fatalf("FindFollowupFileIDLocation error: %v", err)
	}
	if !found {
		t.Fatal("expected follow-up file-id lookup to fall back to Gradle cache")
	}
	if location.Source.Path != jarPath {
		t.Fatalf("unexpected jar path: %q", location.Source.Path)
	}
	if location.InnerPath != inner {
		t.Fatalf("unexpected inner path: %q", location.InnerPath)
	}
	if location.FileID != resolve.FormatFileID(coord, inner) {
		t.Fatalf("unexpected file id: %q", location.FileID)
	}
}

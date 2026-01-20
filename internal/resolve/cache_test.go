package resolve

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindCachedSourcesUsesDottedGroupDir(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	cacheDir := filepath.Join(home, ".gradle", "caches", "modules-2", "files-2.1")
	jarDir := filepath.Join(cacheDir, "org.jetbrains.kotlinx", "kotlinx-datetime", "1.2.3", "hash")
	if err := os.MkdirAll(jarDir, 0o755); err != nil {
		t.Fatalf("mkdir cache dir: %v", err)
	}
	jar := filepath.Join(jarDir, "kotlinx-datetime-1.2.3-sources.jar")
	if err := os.WriteFile(jar, []byte{}, 0o644); err != nil {
		t.Fatalf("write jar: %v", err)
	}

	sources, err := FindCachedSources("org.jetbrains.kotlinx", "kotlinx-datetime", "1.2.3")
	if err != nil {
		t.Fatalf("FindCachedSources error: %v", err)
	}
	if len(sources) != 1 {
		t.Fatalf("expected 1 source, got %d", len(sources))
	}
	if sources[0].Path != jar {
		t.Fatalf("expected jar path %q, got %q", jar, sources[0].Path)
	}
}

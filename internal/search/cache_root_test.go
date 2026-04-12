package search

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSearchExtractCacheRootEnvPathIsAbsolute(t *testing.T) {
	t.Setenv(extractCacheDirEnv, filepath.Join("relative", "cache"))
	root, err := searchExtractCacheRoot()
	if err != nil {
		t.Fatalf("searchExtractCacheRoot() error = %v", err)
	}
	if !filepath.IsAbs(root) {
		t.Fatalf("expected absolute cache root, got %q", root)
	}
	if filepath.Base(root) != "cache" {
		t.Fatalf("unexpected cache root %q", root)
	}
}

func TestSearchExtractCacheRootFallsBackToTempDirWhenUserCacheDirFails(t *testing.T) {
	original := userCacheDir
	userCacheDir = func() (string, error) {
		return "", filepath.ErrBadPattern
	}
	t.Cleanup(func() {
		userCacheDir = original
	})

	t.Setenv(extractCacheDirEnv, "")
	root, err := searchExtractCacheRoot()
	if err != nil {
		t.Fatalf("searchExtractCacheRoot() error = %v", err)
	}
	want := filepath.Join(os.TempDir(), extractCacheRootName)
	if root != want {
		t.Fatalf("unexpected fallback cache root: want %q got %q", want, root)
	}
}

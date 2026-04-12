package search

import (
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

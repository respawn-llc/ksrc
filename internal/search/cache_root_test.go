package search

import (
	"os"
	"path/filepath"
	"runtime"
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
	want := tempExtractCacheRoot()
	if root != want {
		t.Fatalf("unexpected fallback cache root: want %q got %q", want, root)
	}
}

func TestPrepareExtractCacheRootFallsBackToTempDirWhenConfiguredParentIsUnwritable(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows directory access uses ACLs, not chmod write bits")
	}
	blockedParent := filepath.Join(t.TempDir(), "blocked")
	if err := os.MkdirAll(blockedParent, 0o755); err != nil {
		t.Fatalf("mkdir blocked parent: %v", err)
	}
	if err := os.Chmod(blockedParent, 0o555); err != nil {
		t.Fatalf("chmod blocked parent: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chmod(blockedParent, 0o755)
	})

	t.Setenv(extractCacheDirEnv, filepath.Join(blockedParent, "cache"))
	root, err := prepareExtractCacheRoot()
	if err != nil {
		t.Fatalf("prepareExtractCacheRoot() error = %v", err)
	}
	if root != tempExtractCacheRoot() {
		t.Fatalf("expected temp fallback root, got %q", root)
	}
}

func TestPrepareExtractCacheRootFallsBackToTempDirWhenConfiguredRootIsUnwritable(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows directory access uses ACLs, not chmod write bits")
	}
	blockedRoot := filepath.Join(t.TempDir(), "cache")
	if err := os.MkdirAll(blockedRoot, 0o755); err != nil {
		t.Fatalf("mkdir blocked root: %v", err)
	}
	if err := os.Chmod(blockedRoot, 0o555); err != nil {
		t.Fatalf("chmod blocked root: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chmod(blockedRoot, 0o755)
	})

	t.Setenv(extractCacheDirEnv, blockedRoot)
	root, err := prepareExtractCacheRoot()
	if err != nil {
		t.Fatalf("prepareExtractCacheRoot() error = %v", err)
	}
	if root != tempExtractCacheRoot() {
		t.Fatalf("expected temp fallback root, got %q", root)
	}
}

func TestPrepareExtractCacheRootFallsBackWhenPathIsBlocked(t *testing.T) {
	for _, blockParent := range []bool{false, true} {
		t.Run(map[bool]string{false: "root", true: "parent"}[blockParent], func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "blocked")
			if err := os.WriteFile(path, []byte("not a directory"), 0o644); err != nil {
				t.Fatal(err)
			}
			if blockParent {
				path = filepath.Join(path, "cache")
			}
			t.Setenv(extractCacheDirEnv, path)
			root, err := prepareExtractCacheRoot()
			if err != nil {
				t.Fatal(err)
			}
			if root != tempExtractCacheRoot() {
				t.Fatalf("expected fallback root, got %q", root)
			}
		})
	}
}

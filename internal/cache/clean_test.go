package cache

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func writeCacheFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestInspectAndDelete(t *testing.T) {
	root := t.TempDir()
	first := filepath.Join(root, "ksrc")
	writeCacheFile(t, filepath.Join(first, "old-version", "file"), "123")
	writeCacheFile(t, filepath.Join(first, "current-version", "file"), "12345")
	outside := filepath.Join(root, "gradle", "keep")
	writeCacheFile(t, outside, "untouched")
	folders, err := Inspect(context.Background(), []string{first, first, filepath.Join(first, "old-version"), filepath.Join(root, "absent")})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(folders, []Folder{{Path: first, Bytes: 8}}) {
		t.Fatalf("folders = %+v", folders)
	}
	if _, err := os.Stat(first); err != nil {
		t.Fatalf("inspection mutated cache: %v", err)
	}
	if err := Delete(context.Background(), folders); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(first); !os.IsNotExist(err) {
		t.Fatalf("cache remains: %v", err)
	}
	if data, err := os.ReadFile(outside); err != nil || string(data) != "untouched" {
		t.Fatalf("outside data changed: %q, %v", data, err)
	}
}

func TestCleanDoesNotFollowSymlinks(t *testing.T) {
	base := t.TempDir()
	outside := filepath.Join(base, "outside")
	writeCacheFile(t, filepath.Join(outside, "keep"), "untouched")
	for _, rootLink := range []bool{false, true} {
		path := filepath.Join(t.TempDir(), "cache")
		link := path
		if !rootLink {
			if err := os.Mkdir(path, 0o700); err != nil {
				t.Fatal(err)
			}
			link = filepath.Join(path, "link")
		}
		if err := os.Symlink(outside, link); err != nil {
			t.Skipf("symlinks unavailable: %v", err)
		}
		folders, err := Inspect(context.Background(), []string{path})
		if err != nil {
			t.Fatal(err)
		}
		if len(folders) != 1 || folders[0].Bytes != 0 {
			t.Fatalf("followed symlink: %+v", folders)
		}
		if err := Delete(context.Background(), folders); err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(filepath.Join(outside, "keep")); err != nil {
			t.Fatal(err)
		}
	}
}

func TestInspectRejectsSharedDirectories(t *testing.T) {
	home, _ := os.UserHomeDir()
	userCache, _ := os.UserCacheDir()
	cwd, _ := os.Getwd()
	for _, root := range []string{home, userCache, cwd, os.TempDir(), filepath.VolumeName(cwd) + string(filepath.Separator)} {
		if _, err := Inspect(context.Background(), []string{root}); err == nil {
			t.Errorf("accepted shared root %q", root)
		}
	}
}

func TestInspectAndDeleteCancellation(t *testing.T) {
	root := filepath.Join(t.TempDir(), "cache")
	writeCacheFile(t, filepath.Join(root, "keep"), "123")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := Inspect(ctx, []string{root}); err == nil {
		t.Fatal("inspection ignored cancellation")
	}
	if err := Delete(ctx, []Folder{{Path: root}}); err == nil {
		t.Fatal("deletion ignored cancellation")
	}
	if _, err := os.Stat(root); err != nil {
		t.Fatal(err)
	}
}

func TestRootsIncludeDefaultsAndOverrides(t *testing.T) {
	base, err := os.UserCacheDir()
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv(ExtractDirEnv, "custom-extract")
	t.Setenv(FileIDDirEnv, "custom-fileids")
	roots, err := Roots()
	if err != nil {
		t.Fatal(err)
	}
	want := []string{filepath.Dir(TempExtractRoot()), filepath.Join(base, "ksrc"), "custom-extract", "custom-fileids"}
	if !reflect.DeepEqual(roots, want) {
		t.Fatalf("roots = %q; want %q", roots, want)
	}
}

func TestDeleteReportsFailuresAndContinues(t *testing.T) {
	base := t.TempDir()
	blocked := filepath.Join(base, "not-a-directory")
	writeCacheFile(t, blocked, "keep")
	good := filepath.Join(base, "cache")
	writeCacheFile(t, filepath.Join(good, "remove"), "123")
	bad := filepath.Join(blocked, "cache")
	err := Delete(context.Background(), []Folder{{Path: bad}, {Path: good}})
	if err == nil || !strings.Contains(err.Error(), bad) {
		t.Fatalf("missing failed path: %v", err)
	}
	if _, err := os.Stat(good); !os.IsNotExist(err) {
		t.Fatalf("did not continue after failure: %v", err)
	}
}

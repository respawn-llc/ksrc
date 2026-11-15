package cache

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
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
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	first := filepath.Join(root, "ksrc")
	writeCacheFile(t, filepath.Join(first, "old-version", "file"), "123")
	writeCacheFile(t, filepath.Join(first, "current-version", "file"), "12345")
	sibling := first + "-other"
	writeCacheFile(t, filepath.Join(sibling, "file"), "123456")
	outside := filepath.Join(root, "gradle", "keep")
	writeCacheFile(t, outside, "untouched")
	folders, err := Inspect(context.Background(), []string{first, first, filepath.Join(first, "old-version"), sibling, filepath.Join(root, "absent")})
	if err != nil {
		t.Fatal(err)
	}
	if len(folders) != 2 || folders[0].Path != first || folders[0].Bytes != 8 || folders[1].Path != sibling || folders[1].Bytes != 6 {
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
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	base := filepath.Join(root, `with\backslash`)
	blocked := filepath.Join(base, "blocked-parent")
	bad := filepath.Join(blocked, "cache")
	if err := os.MkdirAll(bad, 0o700); err != nil {
		t.Fatal(err)
	}
	good := filepath.Join(base, "cache")
	writeCacheFile(t, filepath.Join(good, "remove"), "123")
	folders, err := Inspect(context.Background(), []string{bad, good})
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS == "windows" {
		// Windows cannot delete a file opened without delete sharing.
		held, err := os.Create(filepath.Join(bad, "held-open"))
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = held.Close() })
	} else {
		if err := os.Remove(bad); err != nil {
			t.Fatal(err)
		}
		if err := os.Remove(blocked); err != nil {
			t.Fatal(err)
		}
		writeCacheFile(t, blocked, "keep")
	}
	err = Delete(context.Background(), folders)
	if err == nil || !strings.Contains(err.Error(), strconv.Quote(bad)) {
		t.Fatalf("missing failed path: %v", err)
	}
	if _, err := os.Stat(good); !os.IsNotExist(err) {
		t.Fatalf("did not continue after failure: %v", err)
	}
}

func TestInspectRejectsSharedDirectoryThroughSymlinkedParent(t *testing.T) {
	home := filepath.Join(t.TempDir(), "user", "home")
	if err := os.MkdirAll(home, 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	link := filepath.Join(t.TempDir(), "link")
	if err := os.Symlink(filepath.Dir(home), link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	path := filepath.Join(link, filepath.Base(home))
	if _, err := Inspect(context.Background(), []string{path}); err == nil {
		t.Fatal("accepted home directory through a symlinked parent")
	}
}

func TestCleanUsesPreviewedResolvedLocation(t *testing.T) {
	base := t.TempDir()
	first, second := filepath.Join(base, "first"), filepath.Join(base, "second")
	for _, parent := range []string{first, second} {
		writeCacheFile(t, filepath.Join(parent, "cache", "keep"), "123")
	}
	link := filepath.Join(base, "link")
	if err := os.Symlink(first, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	folders, err := Inspect(context.Background(), []string{filepath.Join(link, "cache")})
	if err != nil {
		t.Fatal(err)
	}
	previewed, err := filepath.EvalSymlinks(filepath.Join(first, "cache"))
	if err != nil {
		t.Fatal(err)
	}
	if len(folders) != 1 || folders[0].Path != previewed {
		t.Fatalf("preview did not show resolved location: %+v", folders)
	}
	if err := os.Remove(link); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(second, link); err != nil {
		t.Fatal(err)
	}
	if err := Delete(context.Background(), folders); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(previewed); !os.IsNotExist(err) {
		t.Fatalf("previewed cache remains: %v", err)
	}
	if _, err := os.Stat(filepath.Join(second, "cache", "keep")); err != nil {
		t.Fatalf("deleted unconfirmed files: %v", err)
	}
}

func TestCleanFallbackWhenUserCacheDirUnavailable(t *testing.T) {
	temp := t.TempDir()
	t.Setenv("TMPDIR", temp)
	t.Setenv("TMP", temp)
	t.Setenv("TEMP", temp)
	switch runtime.GOOS {
	case "windows":
		t.Setenv("LocalAppData", "")
	case "darwin":
		t.Setenv("HOME", "")
	default:
		t.Setenv("HOME", "")
		t.Setenv("XDG_CACHE_HOME", "")
	}
	if _, err := os.UserCacheDir(); err == nil {
		t.Skip("platform can resolve user cache without environment")
	}
	t.Setenv(ExtractDirEnv, "")
	t.Setenv(FileIDDirEnv, "")
	file := filepath.Join(TempExtractRoot(), "entry")
	writeCacheFile(t, file, "123")
	roots, _ := Roots()
	folders, err := Inspect(context.Background(), roots)
	if err != nil {
		t.Fatal(err)
	}
	if len(folders) != 1 || folders[0].Bytes != 3 {
		t.Fatalf("fallback missing from preview: %+v", folders)
	}
	if err := Delete(context.Background(), folders); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(file); !os.IsNotExist(err) {
		t.Fatalf("fallback not cleaned: %v", err)
	}
}

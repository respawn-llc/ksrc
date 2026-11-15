package cat

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestReadSourceFileFromDirectory(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "android", "app", "Activity.java")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	body := "public class Activity {}\nline two\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := ReadSourceFile(root, "android/app/Activity.java", nil)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != body {
		t.Fatalf("body = %q", got)
	}
	ranged, err := ReadSourceFile(root, "android/app/Activity.java", &LineRange{Start: 2, End: 2})
	if err != nil {
		t.Fatal(err)
	}
	if string(ranged) != "line two\n" {
		t.Fatalf("range = %q", ranged)
	}
	if _, err := ReadSourceFile(root, "android/../Activity.java", nil); err == nil {
		t.Fatal("expected traversal to fail")
	}
	ok, err := HasSourceFile(root, "android/app/Activity.java")
	if err != nil || !ok {
		t.Fatalf("has file ok=%v err=%v", ok, err)
	}
	ok, err = HasSourceFile(root, "missing.java")
	if err != nil || ok {
		t.Fatalf("missing ok=%v err=%v", ok, err)
	}
	if strings.Contains(string(got), "Stub!") {
		t.Fatal("unexpected stub text")
	}
}

func TestDirectorySourceRejectsEscapingSymlinks(t *testing.T) {
	for _, directoryLink := range []bool{false, true} {
		t.Run(map[bool]string{false: "file", true: "directory"}[directoryLink], func(t *testing.T) {
			root, outside := t.TempDir(), t.TempDir()
			secret := filepath.Join(outside, "Secret.java")
			if err := os.WriteFile(secret, []byte("outside source root"), 0o644); err != nil {
				t.Fatal(err)
			}
			target, inner := secret, "link"
			if directoryLink {
				target, inner = outside, "link/Secret.java"
			}
			makeSourceSymlink(t, target, filepath.Join(root, "link"))
			if ok, err := HasSourceFile(root, inner); err == nil || ok {
				t.Fatalf("escaping link accepted: ok=%v err=%v", ok, err)
			}
			if content, err := ReadSourceFile(root, inner, nil); err == nil {
				t.Fatalf("read escaped source tree: %q", content)
			}
		})
	}
}

func TestDirectorySourceRechecksSymlinkOnRead(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "Inside.java"), []byte("inside"), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "link")
	makeSourceSymlink(t, "Inside.java", link)
	if ok, err := HasSourceFile(root, "link"); err != nil || !ok {
		t.Fatalf("internal link rejected: ok=%v err=%v", ok, err)
	}
	if content, err := ReadSourceFile(root, "link", nil); err != nil || string(content) != "inside" {
		t.Fatalf("internal link unreadable: %q err=%v", content, err)
	}
	outside := filepath.Join(t.TempDir(), "Outside.java")
	if err := os.WriteFile(outside, []byte("outside"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(link); err != nil {
		t.Fatal(err)
	}
	makeSourceSymlink(t, outside, link)
	if content, err := ReadSourceFile(root, "link", nil); err == nil {
		t.Fatalf("changed link escaped source tree: %q", content)
	}
}

func makeSourceSymlink(t *testing.T, target, path string) {
	t.Helper()
	if err := os.Symlink(target, path); err != nil {
		if runtime.GOOS == "windows" {
			t.Skipf("Windows symlink creation unavailable: %v", err)
		}
		t.Fatal(err)
	}
}

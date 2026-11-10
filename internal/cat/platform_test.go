package cat

import (
	"os"
	"path/filepath"
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

package cat

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
)

func TestParseLineRange(t *testing.T) {
	cases := []struct {
		in    string
		start int
		end   int
	}{
		{"1,3", 1, 3},
		{"1:3", 1, 3},
		{"1-3", 1, 3},
		{"1 3", 1, 3},
		{"1..3", 1, 3},
		{"1;3", 1, 3},
	}
	for _, tc := range cases {
		lr, err := ParseLineRange(tc.in)
		if err != nil {
			t.Fatalf("unexpected error for %q: %v", tc.in, err)
		}
		if lr.Start != tc.start || lr.End != tc.end {
			t.Fatalf("unexpected range for %q: %+v", tc.in, lr)
		}
	}
	if _, err := ParseLineRange("1 3 4"); err == nil {
		t.Fatal("expected error for invalid range")
	}
}

func TestReadFileFromZipRange(t *testing.T) {
	tmp := t.TempDir()
	zipPath := filepath.Join(tmp, "test.jar")
	inner := "a/b.txt"

	f, err := os.Create(zipPath)
	if err != nil {
		t.Fatalf("create zip: %v", err)
	}
	zw := zip.NewWriter(f)
	w, err := zw.Create(inner)
	if err != nil {
		t.Fatalf("create entry: %v", err)
	}
	_, _ = w.Write([]byte("line1\nline2\nline3\n"))
	_ = zw.Close()
	_ = f.Close()

	lr := &LineRange{Start: 2, End: 3}
	data, err := ReadFileFromZip(zipPath, inner, lr)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(data) != "line2\nline3\n" {
		t.Fatalf("unexpected data: %q", string(data))
	}
}

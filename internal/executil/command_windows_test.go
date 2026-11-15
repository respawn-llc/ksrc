package executil

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOSRunnerBatchArguments(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "project & spaces %PATH% !")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "gradlew.bat")
	if err := os.WriteFile(path, []byte("@echo off\r\n"+
		"echo \"%~1\"\r\necho \"%~2\"\r\necho diagnostic 1>&2\r\nexit /b 7\r\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, stderr, err := (OSRunner{}).Run(context.Background(), dir, path, `a & b %PATH% !`, `C:\cache with spaces\`)
	if err == nil || err.Error() != "exit status 7" {
		t.Fatalf("expected exit 7, got %v", err)
	}
	if strings.ReplaceAll(out, "\r\n", "\n") != "\"a & b %PATH% !\"\n\"C:\\cache with spaces\\\"\n" {
		t.Fatalf("arguments changed: %q", out)
	}
	if strings.TrimSpace(stderr) != "diagnostic" {
		t.Fatalf("stderr lost: %q", stderr)
	}
}

func TestBatchRejectsUnrepresentableArguments(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.cmd")
	if err := os.WriteFile(path, []byte("@echo off\r\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, arg := range []string{"a\"b", "a\nb", "a\x00b"} {
		if _, err := command(context.Background(), path, arg); err == nil {
			t.Fatalf("expected invalid argument %q to fail", arg)
		}
	}
}

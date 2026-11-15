package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/respawn-llc/ksrc/internal/cache"
)

func isolateCleanCaches(t *testing.T) string {
	t.Helper()
	base, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_CACHE_HOME", base)
	t.Setenv("HOME", base)
	t.Setenv("LocalAppData", base)
	t.Setenv("TMPDIR", base)
	t.Setenv("TMP", base)
	t.Setenv("TEMP", base)
	t.Setenv("KSRC_EXTRACT_CACHE_DIR", filepath.Join(base, "extract"))
	t.Setenv("KSRC_FILEID_CACHE_DIR", filepath.Join(base, "fileids"))
	path := filepath.Join(base, "extract", "file")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("1234"), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestCleanCommandNonInteractive(t *testing.T) {
	for _, tc := range []struct {
		name    string
		args    []string
		deletes bool
	}{
		{name: "default"},
		{name: "dry-run", args: []string{"--dry-run"}},
		{name: "yes", args: []string{"--yes"}, deletes: true},
		{name: "short yes", args: []string{"-y"}, deletes: true},
		{name: "dry-run overrides yes", args: []string{"--yes", "--dry-run"}},
		{name: "false dry-run still safe", args: []string{"--dry-run=false"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := isolateCleanCaches(t)
			cmd := NewRootCommand(NewApp())
			var out, stderr bytes.Buffer
			cmd.SetOut(&out)
			cmd.SetErr(&stderr)
			cmd.SetIn(strings.NewReader("yes\n"))
			cmd.SetArgs(append([]string{"clean"}, tc.args...))
			if err := cmd.Execute(); err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(out.String(), filepath.Dir(path)) || !strings.Contains(out.String(), "4 B") {
				t.Fatalf("missing folder/size summary: %s", &out)
			}
			if strings.Contains(out.String(), path) || strings.Contains(stderr.String(), "[y/N]") {
				t.Fatalf("unexpected file listing or prompt: %s %s", &out, &stderr)
			}
			_, err := os.Stat(path)
			if tc.deletes != os.IsNotExist(err) {
				t.Fatalf("deletes = %v, stat error = %v", tc.deletes, err)
			}
		})
	}
}

func TestCleanInteractiveConfirmation(t *testing.T) {
	for _, tc := range []struct {
		answer  string
		deletes bool
		wantErr bool
	}{
		{answer: "y\n", deletes: true},
		{answer: "YES\n", deletes: true},
		{answer: "\n"},
		{answer: "no\n"},
		{answer: "", wantErr: true},
	} {
		t.Run(strings.TrimSpace(tc.answer), func(t *testing.T) {
			path := isolateCleanCaches(t)
			cmd := newCleanCmd()
			cmd.SetContext(context.Background())
			cmd.SetIn(strings.NewReader(tc.answer))
			var out, stderr bytes.Buffer
			cmd.SetOut(&out)
			cmd.SetErr(&stderr)
			err := cleanCaches(cmd, false, false, true)
			if (err != nil) != tc.wantErr {
				t.Fatalf("error = %v", err)
			}
			if !strings.Contains(stderr.String(), "[y/N]") || !strings.Contains(out.String(), filepath.Dir(path)) {
				t.Fatalf("missing preview or confirmation: %s %s", &out, &stderr)
			}
			_, err = os.Stat(path)
			if tc.deletes != os.IsNotExist(err) {
				t.Fatalf("deletes = %v, stat = %v", tc.deletes, err)
			}
		})
	}
}

func TestCleanDeletesOldDefaultAndFallbackCachesWithOverrides(t *testing.T) {
	custom := isolateCleanCaches(t)
	userCache, err := os.UserCacheDir()
	if err != nil {
		t.Fatal(err)
	}
	paths := []string{
		filepath.Join(userCache, "ksrc", "search-extracted-v0", "old"),
		filepath.Join(filepath.Dir(cache.TempExtractRoot()), "search-extracted-v0", "old"),
	}
	for _, path := range paths {
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("123"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	cmd := newCleanCmd()
	cmd.SetArgs([]string{"--yes"})
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	for _, path := range append(paths, custom) {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("cache %q remains: %v", path, err)
		}
	}
}

func TestCleanRejectsExtraArguments(t *testing.T) {
	cmd := newCleanCmd()
	cmd.SetArgs([]string{"some-directory"})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	if err := cmd.Execute(); err == nil {
		t.Fatal("accepted an arbitrary cleanup path")
	}
}

func TestCleanWarnsAndCleansFallbackWithoutUserCacheDirectory(t *testing.T) {
	custom := isolateCleanCaches(t)
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
	fallback := filepath.Join(cache.TempExtractRoot(), "entry")
	if err := os.MkdirAll(filepath.Dir(fallback), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fallback, []byte("123"), 0o600); err != nil {
		t.Fatal(err)
	}
	cmd := newCleanCmd()
	cmd.SetArgs([]string{"--yes"})
	var out, stderr bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&stderr)
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stderr.String(), "user cache directory unavailable") {
		t.Fatalf("missing diagnostic: %s", &stderr)
	}
	for _, path := range []string{custom, fallback} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("cache %q remains: %v", path, err)
		}
	}
}

package scripts

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestTapUpdateRemovesOldBottles(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("release script runs on Unix")
	}
	for _, gap := range []string{"", "\n"} {
		t.Run("gap="+gap, func(t *testing.T) {
			tap := t.TempDir()
			formulaDir := filepath.Join(tap, "Formula")
			if err := os.Mkdir(formulaDir, 0755); err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(formulaDir, "ksrc.rb")
			before := "class Ksrc < Formula\n  url \"old\"\n  sha256 \"old\"\n\n  bottle do\n    sha256 arm64_tahoe: \"old-bottle\"\n  end\n" + gap + "  depends_on \"ripgrep\"\nend\n"
			if err := os.WriteFile(path, []byte(before), 0644); err != nil {
				t.Fatal(err)
			}
			bin := t.TempDir()
			// Return deterministic archive bytes without contacting GitHub.
			if err := os.WriteFile(filepath.Join(bin, "curl"), []byte("#!/bin/sh\nwhile [ \"$1\" != '-o' ]; do shift; done\nshift\nprintf archive > \"$1\"\n"), 0755); err != nil {
				t.Fatal(err)
			}
			cmd := exec.Command("bash", "update-brew-tap.sh", "--version", "v1.2.3", "--tap", tap)
			cmd.Env = append(os.Environ(), "PATH="+bin+string(os.PathListSeparator)+os.Getenv("PATH"))
			if out, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("%v\n%s", err, out)
			}
			after, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(after), "bottle") || !strings.Contains(string(after), "v1.2.3.tar.gz") || !strings.Contains(string(after), "depends_on \"ripgrep\"") {
				t.Fatalf("incorrect updated formula:\n%s", after)
			}
		})
	}
}

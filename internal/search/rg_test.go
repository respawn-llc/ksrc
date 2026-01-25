package search

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/respawn-app/ksrc/internal/resolve"
)

func TestParseRgLine(t *testing.T) {
	line := "/tmp/lib.jar:com/foo/Bar.kt:12:3:match text"
	m, ok := parseRgLine(line)
	if !ok {
		t.Fatal("expected parse ok")
	}
	if m.File != "/tmp/lib.jar:com/foo/Bar.kt" || m.Line != 12 || m.Column != 3 || m.Text != "match text" {
		t.Fatalf("unexpected match: %+v", m)
	}
}

func TestParseRgContextLine(t *testing.T) {
	line := "/tmp/foo-bar/baz.kt-7-context line"
	m, ok := parseRgLine(line)
	if !ok {
		t.Fatal("expected parse ok")
	}
	if m.File != "/tmp/foo-bar/baz.kt" || m.Line != 7 || m.Column != 0 || m.Text != "context line" {
		t.Fatalf("unexpected match: %+v", m)
	}
}

func TestRunUsesZipSearchWhenSupported(t *testing.T) {
	resetZipSupportCacheForTests()
	jarPath := filepath.Join(t.TempDir(), "foo.jar")
	coord := resolve.Coord{Group: "com.example", Artifact: "foo", Version: "1.0.0"}
	runner := &fakeRunner{jarPath: jarPath}

	matches, err := Run(context.Background(), runner, Options{
		Pattern: "Needle",
		Jars:    []resolve.SourceJar{{Coord: coord, Path: jarPath}},
		WorkDir: ".",
	})
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if len(matches) != 1 {
		t.Fatalf("expected 1 match, got %d", len(matches))
	}
	if matches[0].FileID != "com.example:foo:1.0.0!/com/foo/Bar.kt" {
		t.Fatalf("unexpected file-id: %s", matches[0].FileID)
	}
	if !runner.usedSearchZip {
		t.Fatalf("expected rg to be invoked with --search-zip")
	}
}

func TestRunTreatsExitCodeOneAsNoMatches(t *testing.T) {
	resetZipSupportCacheForTests()
	jarPath := filepath.Join(t.TempDir(), "foo.jar")
	coord := resolve.Coord{Group: "com.example", Artifact: "foo", Version: "1.0.0"}
	runner := &exitCodeRunner{jarPath: jarPath, exitCode: 1}

	matches, err := Run(context.Background(), runner, Options{
		Pattern: "Needle",
		Jars:    []resolve.SourceJar{{Coord: coord, Path: jarPath}},
		WorkDir: ".",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(matches) != 0 {
		t.Fatalf("expected no matches, got %d", len(matches))
	}
}

func TestParseRgOutputMapsFileIDs(t *testing.T) {
	coord := resolve.Coord{Group: "com.example", Artifact: "demo", Version: "1.0.0"}
	stdout := "/tmp/root/demo.kt:7:2:match\n"
	roots := map[string]resolve.Coord{"/tmp/root": coord}

	matches := parseRgOutput(stdout, func(filePath string) (resolve.Coord, string, bool) {
		return mapToCoord(roots, filePath)
	})
	if len(matches) != 1 {
		t.Fatalf("expected 1 match, got %d", len(matches))
	}
	if matches[0].FileID != "com.example:demo:1.0.0!/demo.kt" {
		t.Fatalf("unexpected file-id: %s", matches[0].FileID)
	}
}

type fakeRunner struct {
	jarPath       string
	usedSearchZip bool
}

func (f *fakeRunner) Run(_ context.Context, _ string, name string, args ...string) (string, string, error) {
	if name != "rg" {
		return "", "", fmt.Errorf("unexpected command: %s", name)
	}
	for _, arg := range args {
		if arg == "--search-zip" {
			f.usedSearchZip = true
			break
		}
	}
	if containsArg(args, "ksrc-zip-probe") {
		path := args[len(args)-1]
		return fmt.Sprintf("%s:probe.txt:1:1:ksrc-zip-probe\n", path), "", nil
	}
	if containsArg(args, "Needle") {
		return fmt.Sprintf("%s:com/foo/Bar.kt:12:3:Needle\n", f.jarPath), "", nil
	}
	return "", "", nil
}

func (f *fakeRunner) LookPath(string) (string, error) {
	return "rg", nil
}

func containsArg(args []string, needle string) bool {
	for _, arg := range args {
		if arg == needle {
			return true
		}
	}
	return false
}

type exitCodeRunner struct {
	jarPath  string
	exitCode int
}

func (e *exitCodeRunner) Run(_ context.Context, _ string, name string, args ...string) (string, string, error) {
	if name != "rg" {
		return "", "", fmt.Errorf("unexpected command: %s", name)
	}
	if containsArg(args, "ksrc-zip-probe") {
		path := args[len(args)-1]
		return fmt.Sprintf("%s:probe.txt:1:1:ksrc-zip-probe\n", path), "", nil
	}
	if containsArg(args, "Needle") {
		return "", "", exitError{code: e.exitCode}
	}
	return "", "", nil
}

func (e *exitCodeRunner) LookPath(string) (string, error) {
	return "rg", nil
}

type exitError struct {
	code int
}

func (e exitError) Error() string {
	return fmt.Sprintf("exit status %d", e.code)
}

func (e exitError) ExitCode() int {
	return e.code
}

package search

import (
	"archive/zip"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/respawn-app/ksrc/internal/resolve"
)

func TestParseRgLine(t *testing.T) {
	line := rgEventLine(t, "match", "/tmp/lib.jar:com/foo/Bar.kt", "match text\n", 12, 2)
	m, ok := parseRgLine(line)
	if !ok {
		t.Fatal("expected parse ok")
	}
	if m.File != "/tmp/lib.jar:com/foo/Bar.kt" || m.Line != 12 || m.Column != 3 || m.Text != "match text" {
		t.Fatalf("unexpected match: %+v", m)
	}
}

func TestParseRgContextLine(t *testing.T) {
	line := rgEventLine(t, "context", "/tmp/foo-bar/baz.kt", "context line\n", 7, -1)
	m, ok := parseRgLine(line)
	if !ok {
		t.Fatal("expected parse ok")
	}
	if m.File != "/tmp/foo-bar/baz.kt" || m.Line != 7 || m.Column != 0 || m.Text != "context line" {
		t.Fatalf("unexpected match: %+v", m)
	}
}

func TestSearchKeepsMatchTextContainingColon(t *testing.T) {
	coord := resolve.Coord{Group: "com.example", Artifact: "demo", Version: "1.0.0"}
	stdout := rgEventLine(t, "match", "/tmp/root/demo.kt", "prefix:Needle:suffix\n", 7, 7)
	roots := map[string]resolve.SourceJar{"/tmp/root": {Coord: coord, Path: "/tmp/demo-sources.jar"}}

	matches := parseRgOutput(stdout, func(filePath string) (resolve.SourceJar, string, bool) {
		return mapToSource(roots, filePath)
	})
	if len(matches.Matches) != 1 {
		t.Fatalf("expected 1 match, got %d", len(matches.Matches))
	}
	if matches.Matches[0].Text != "prefix:Needle:suffix" {
		t.Fatalf("unexpected text: %q", matches.Matches[0].Text)
	}
}

func TestRunUsesExtractedSearch(t *testing.T) {
	cacheDir := filepath.Join(t.TempDir(), "cache")
	t.Setenv(extractCacheDirEnv, cacheDir)
	jarPath := filepath.Join(t.TempDir(), "foo.jar")
	writeTestJar(t, jarPath, map[string]string{"com/foo/Bar.kt": "Needle\n"})
	coord := resolve.Coord{Group: "com.example", Artifact: "foo", Version: "1.0.0"}
	runner := &fakeRunner{}

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
	if runner.usedSearchZip {
		t.Fatal("did not expect rg to be invoked with --search-zip")
	}
}

func TestRunTreatsExitCodeOneAsNoMatches(t *testing.T) {
	cacheDir := filepath.Join(t.TempDir(), "cache")
	t.Setenv(extractCacheDirEnv, cacheDir)
	jarPath := filepath.Join(t.TempDir(), "foo.jar")
	writeTestJar(t, jarPath, map[string]string{"com/foo/Bar.kt": "Needle\n"})
	coord := resolve.Coord{Group: "com.example", Artifact: "foo", Version: "1.0.0"}
	runner := &exitCodeRunner{exitCode: 1}

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
	stdout := rgEventLine(t, "match", "/tmp/root/demo.kt", "match\n", 7, 1)
	roots := map[string]resolve.SourceJar{"/tmp/root": {Coord: coord, Path: "/tmp/demo-sources.jar"}}

	matches := parseRgOutput(stdout, func(filePath string) (resolve.SourceJar, string, bool) {
		return mapToSource(roots, filePath)
	})
	if len(matches.Matches) != 1 {
		t.Fatalf("expected 1 match, got %d", len(matches.Matches))
	}
	if matches.Matches[0].FileID != "com.example:demo:1.0.0!/demo.kt" {
		t.Fatalf("unexpected file-id: %s", matches.Matches[0].FileID)
	}
}

func TestExtractJarCachedReusesDirectory(t *testing.T) {
	cacheDir := filepath.Join(t.TempDir(), "cache")
	t.Setenv(extractCacheDirEnv, cacheDir)
	jarPath := filepath.Join(t.TempDir(), "foo.jar")
	writeTestJar(t, jarPath, map[string]string{"com/foo/Bar.kt": "Needle\n"})

	first, err := extractJarCached(jarPath)
	if err != nil {
		t.Fatalf("extract first: %v", err)
	}
	second, err := extractJarCached(jarPath)
	if err != nil {
		t.Fatalf("extract second: %v", err)
	}
	if first != second {
		t.Fatalf("expected cache reuse, got %q and %q", first, second)
	}
	if _, err := os.Stat(filepath.Join(first, extractCacheReady)); err != nil {
		t.Fatalf("expected ready marker: %v", err)
	}
}

func TestExtractJarCachedReplacesStalePartialDirectory(t *testing.T) {
	cacheDir := filepath.Join(t.TempDir(), "cache")
	t.Setenv(extractCacheDirEnv, cacheDir)
	jarPath := filepath.Join(t.TempDir(), "foo.jar")
	writeTestJar(t, jarPath, map[string]string{"com/foo/Bar.kt": "Needle\n"})

	key, err := extractCacheKey(jarPath)
	if err != nil {
		t.Fatalf("extract cache key: %v", err)
	}
	staleDir := filepath.Join(cacheDir, key)
	if err := os.MkdirAll(filepath.Join(staleDir, "stale"), 0o755); err != nil {
		t.Fatalf("mkdir stale dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(staleDir, "stale", "leftover.txt"), []byte("bad\n"), 0o644); err != nil {
		t.Fatalf("write stale file: %v", err)
	}

	dir, err := extractJarCached(jarPath)
	if err != nil {
		t.Fatalf("extract cached: %v", err)
	}
	if dir != staleDir {
		t.Fatalf("expected cache dir %q, got %q", staleDir, dir)
	}
	if _, err := os.Stat(filepath.Join(dir, extractCacheReady)); err != nil {
		t.Fatalf("expected ready marker: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "stale", "leftover.txt")); !os.IsNotExist(err) {
		t.Fatalf("expected stale file removed, stat err: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "com", "foo", "Bar.kt")); err != nil {
		t.Fatalf("expected extracted file: %v", err)
	}
}

func TestExtractJarCachedReplacesEmptyDirectoryWithoutReadyMarker(t *testing.T) {
	cacheDir := filepath.Join(t.TempDir(), "cache")
	t.Setenv(extractCacheDirEnv, cacheDir)
	jarPath := filepath.Join(t.TempDir(), "foo.jar")
	writeTestJar(t, jarPath, map[string]string{"com/foo/Bar.kt": "Needle\n"})

	key, err := extractCacheKey(jarPath)
	if err != nil {
		t.Fatalf("extract cache key: %v", err)
	}
	staleDir := filepath.Join(cacheDir, key)
	if err := os.MkdirAll(staleDir, 0o755); err != nil {
		t.Fatalf("mkdir stale dir: %v", err)
	}

	dir, err := extractJarCached(jarPath)
	if err != nil {
		t.Fatalf("extract cached: %v", err)
	}
	if dir != staleDir {
		t.Fatalf("expected cache dir %q, got %q", staleDir, dir)
	}
	if _, err := os.Stat(filepath.Join(dir, extractCacheReady)); err != nil {
		t.Fatalf("expected ready marker: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "com", "foo", "Bar.kt")); err != nil {
		t.Fatalf("expected extracted file: %v", err)
	}
}

func TestExtractJarRejectsParentTraversalEntry(t *testing.T) {
	tmp := t.TempDir()
	jarPath := filepath.Join(tmp, "malicious-parent.jar")
	writeTestJar(t, jarPath, map[string]string{
		"../escaped.kt": "bad",
	})

	dest := filepath.Join(tmp, "jar-0")
	err := extractJar(jarPath, dest)
	if err == nil {
		t.Fatal("expected extractJar to reject parent traversal entry")
	}
	if !strings.Contains(err.Error(), "invalid path in archive") {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(tmp, "escaped.kt")); !os.IsNotExist(statErr) {
		t.Fatalf("expected escaped file to be absent, stat err: %v", statErr)
	}
}

func TestExtractJarRejectsSiblingPrefixEscape(t *testing.T) {
	tmp := t.TempDir()
	jarPath := filepath.Join(tmp, "malicious-sibling.jar")
	writeTestJar(t, jarPath, map[string]string{
		"../jar-1-sibling/escaped.kt": "bad",
	})

	dest := filepath.Join(tmp, "jar-1")
	err := extractJar(jarPath, dest)
	if err == nil {
		t.Fatal("expected extractJar to reject sibling-prefix escape")
	}
	if !strings.Contains(err.Error(), "invalid path in archive") {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(tmp, "jar-1-sibling", "escaped.kt")); !os.IsNotExist(statErr) {
		t.Fatalf("expected sibling escape file to be absent, stat err: %v", statErr)
	}
}

func writeTestJar(t *testing.T, path string, entries map[string]string) {
	t.Helper()

	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create zip: %v", err)
	}
	zw := zip.NewWriter(f)
	for name, contents := range entries {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatalf("create entry %q: %v", name, err)
		}
		if _, err := w.Write([]byte(contents)); err != nil {
			t.Fatalf("write entry %q: %v", name, err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("close zip writer: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("close zip file: %v", err)
	}
}

type fakeRunner struct {
	usedSearchZip bool
	searchDirs    []string
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
	if containsArg(args, "Needle") {
		searchDir := args[len(args)-1]
		f.searchDirs = append(f.searchDirs, searchDir)
		return rgEventLine(nil, "match", filepath.Join(searchDir, "com/foo/Bar.kt"), "Needle\n", 12, 2), "", nil
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
	exitCode int
}

func (e *exitCodeRunner) Run(_ context.Context, _ string, name string, args ...string) (string, string, error) {
	if name != "rg" {
		return "", "", fmt.Errorf("unexpected command: %s", name)
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

func rgEventLine(t *testing.T, eventType string, path string, text string, lineNumber int, submatchStart int) string {
	type textValue struct {
		Text string `json:"text"`
	}
	type submatch struct {
		Start int       `json:"start"`
		End   int       `json:"end"`
		Match textValue `json:"match"`
	}
	type eventData struct {
		Path       textValue  `json:"path"`
		Lines      textValue  `json:"lines"`
		LineNumber int        `json:"line_number"`
		Submatches []submatch `json:"submatches"`
	}
	type event struct {
		Type string    `json:"type"`
		Data eventData `json:"data"`
	}
	data := eventData{
		Path:       textValue{Text: path},
		Lines:      textValue{Text: text},
		LineNumber: lineNumber,
		Submatches: []submatch{},
	}
	if submatchStart >= 0 {
		data.Submatches = []submatch{{Start: submatchStart, End: submatchStart + 1, Match: textValue{Text: "x"}}}
	}
	encoded, err := json.Marshal(event{Type: eventType, Data: data})
	if err != nil {
		if t != nil {
			t.Fatalf("marshal rg event: %v", err)
		}
		panic(err)
	}
	return string(encoded) + "\n"
}

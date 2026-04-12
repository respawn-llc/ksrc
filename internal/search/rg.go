package search

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/respawn-app/ksrc/internal/executil"
	"github.com/respawn-app/ksrc/internal/resolve"
)

type Match struct {
	FileID  string
	JarPath string
	File    string
	Line    int
	Column  int
	Text    string
}

type Options struct {
	Pattern string
	Jars    []resolve.SourceJar
	RGArgs  []string
	WorkDir string
	Report  func(ExecPlan)
}

type ExecPlan struct {
	Cmd      string
	Args     []string
	JarCount int
	Mode     string
	WorkDir  string
}

func Run(ctx context.Context, runner executil.Runner, opts Options) ([]Match, error) {
	if opts.Pattern == "" {
		return nil, fmt.Errorf("pattern is required")
	}
	if len(opts.Jars) == 0 {
		return nil, fmt.Errorf("no source jars to search")
	}
	if _, err := runner.LookPath("rg"); err != nil {
		return nil, fmt.Errorf("rg not found on PATH")
	}

	strategy := selectStrategy(ctx, runner)
	return strategy.run(ctx, runner, opts)
}

type rgText struct {
	Text  *string `json:"text,omitempty"`
	Bytes *string `json:"bytes,omitempty"`
}

func (v rgText) decode() (string, bool) {
	if v.Text != nil {
		return *v.Text, true
	}
	if v.Bytes != nil {
		decoded, err := base64.StdEncoding.DecodeString(*v.Bytes)
		if err != nil {
			return "", false
		}
		return string(decoded), true
	}
	return "", false
}

type rgSubmatch struct {
	Start int `json:"start"`
}

type rgEventType string

const (
	rgEventMatch   rgEventType = "match"
	rgEventContext rgEventType = "context"
)

type rgMessage struct {
	Type rgEventType `json:"type"`
	Data struct {
		Path       rgText       `json:"path"`
		Lines      rgText       `json:"lines"`
		LineNumber int          `json:"line_number"`
		Submatches []rgSubmatch `json:"submatches"`
	} `json:"data"`
}

func parseRgLine(line string) (Match, bool) {
	var msg rgMessage
	if err := json.Unmarshal([]byte(line), &msg); err != nil {
		return Match{}, false
	}
	if msg.Type != rgEventMatch && msg.Type != rgEventContext {
		return Match{}, false
	}
	file, ok := msg.Data.Path.decode()
	if !ok || file == "" || msg.Data.LineNumber <= 0 {
		return Match{}, false
	}
	text, ok := msg.Data.Lines.decode()
	if !ok {
		return Match{}, false
	}
	match := Match{
		File:   file,
		Line:   msg.Data.LineNumber,
		Column: 0,
		Text:   strings.TrimRight(text, "\r\n"),
	}
	if msg.Type == rgEventMatch && len(msg.Data.Submatches) > 0 {
		match.Column = msg.Data.Submatches[0].Start + 1
	}
	return match, true
}

type sourceMapper func(string) (resolve.SourceJar, string, bool)

type rgParseResult struct {
	Matches []Match
	Parsed  int
	Mapped  int
}

func parseRgOutput(stdout string, mapper sourceMapper) rgParseResult {
	matches := []Match{}
	parsed := 0
	mapped := 0
	for _, line := range strings.Split(stdout, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		m, ok := parseRgLine(line)
		if !ok {
			continue
		}
		parsed++
		source, inner, ok := mapper(m.File)
		if !ok {
			continue
		}
		mapped++
		m.FileID = resolve.FormatFileID(source.Coord, inner)
		m.JarPath = source.Path
		matches = append(matches, m)
	}
	return rgParseResult{Matches: matches, Parsed: parsed, Mapped: mapped}
}

func mapToSource(roots map[string]resolve.SourceJar, filePath string) (resolve.SourceJar, string, bool) {
	for root, source := range roots {
		rel, err := filepath.Rel(root, filePath)
		if err != nil || strings.HasPrefix(rel, "..") {
			continue
		}
		rel = filepath.ToSlash(rel)
		return source, rel, true
	}
	return resolve.SourceJar{}, "", false
}

type searchStrategy struct {
	mode string
	run  func(context.Context, executil.Runner, Options) ([]Match, error)
}

func selectStrategy(context.Context, executil.Runner) searchStrategy {
	return searchStrategy{mode: "extract", run: runExtractSearch}
}

func runExtractSearch(ctx context.Context, runner executil.Runner, opts Options) ([]Match, error) {
	extractRoots := make(map[string]resolve.SourceJar)
	searchDirs := make([]string, 0, len(opts.Jars))
	seenDirs := make(map[string]struct{}, len(opts.Jars))
	for _, j := range opts.Jars {
		dir, err := extractJarCached(j.Path)
		if err != nil {
			return nil, err
		}
		extractRoots[dir] = j
		if _, ok := seenDirs[dir]; ok {
			continue
		}
		seenDirs[dir] = struct{}{}
		searchDirs = append(searchDirs, dir)
	}

	args := []string{"--json", "--color=never"}
	args = append(args, opts.RGArgs...)
	args = append(args, "--")
	args = append(args, opts.Pattern)
	args = append(args, searchDirs...)

	if opts.Report != nil {
		opts.Report(ExecPlan{
			Cmd:      "rg",
			Args:     args,
			JarCount: len(opts.Jars),
			Mode:     "extract",
			WorkDir:  opts.WorkDir,
		})
	}
	stdout, stderr, err := runner.Run(ctx, opts.WorkDir, "rg", args...)
	if err != nil {
		if isNoMatches(err) {
			return nil, nil
		}
		if strings.TrimSpace(stdout) == "" {
			return nil, fmt.Errorf("rg failed: %w\n%s", err, strings.TrimSpace(stderr))
		}
	}

	parsed := parseRgOutput(stdout, func(filePath string) (resolve.SourceJar, string, bool) {
		return mapToSource(extractRoots, filePath)
	})
	return parsed.Matches, nil
}

type exitCoder interface {
	ExitCode() int
}

func isNoMatches(err error) bool {
	if err == nil {
		return false
	}
	if code, ok := err.(exitCoder); ok {
		return code.ExitCode() == 1
	}
	return false
}

const (
	extractCacheDirEnv   = "KSRC_EXTRACT_CACHE_DIR"
	extractCacheReady    = ".ksrc-ready"
	extractCacheRootName = "ksrc/search-extracted-v1"
)

var extractCacheLocks sync.Map

var userCacheDir = os.UserCacheDir

func extractJarCached(src string) (string, error) {
	cacheRoot, err := searchExtractCacheRoot()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(cacheRoot, 0o755); err != nil {
		return "", err
	}
	cacheKey, err := extractCacheKey(src)
	if err != nil {
		return "", err
	}
	dir := filepath.Join(cacheRoot, cacheKey)
	if isExtractedJarReady(dir) {
		return dir, nil
	}
	lock := extractCacheLock(dir)
	lock.Lock()
	defer lock.Unlock()
	if isExtractedJarReady(dir) {
		return dir, nil
	}
	tempDir, err := os.MkdirTemp(cacheRoot, cacheKey+".tmp-")
	if err != nil {
		return "", err
	}
	cleanupTemp := func() {
		_ = os.RemoveAll(tempDir)
	}
	if err := extractJar(src, tempDir); err != nil {
		cleanupTemp()
		return "", err
	}
	if err := os.WriteFile(filepath.Join(tempDir, extractCacheReady), []byte("ok\n"), 0o644); err != nil {
		cleanupTemp()
		return "", err
	}
	if err := os.Rename(tempDir, dir); err != nil {
		if isExtractedJarReady(dir) {
			cleanupTemp()
			return dir, nil
		}
		if removeErr := os.RemoveAll(dir); removeErr == nil {
			if retryErr := os.Rename(tempDir, dir); retryErr == nil {
				return dir, nil
			}
		}
		cleanupTemp()
		return "", err
	}
	return dir, nil
}

func searchExtractCacheRoot() (string, error) {
	if dir := strings.TrimSpace(os.Getenv(extractCacheDirEnv)); dir != "" {
		abs, err := filepath.Abs(dir)
		if err != nil {
			return "", err
		}
		return filepath.Clean(abs), nil
	}
	base, err := userCacheDir()
	if err != nil {
		return filepath.Join(os.TempDir(), extractCacheRootName), nil
	}
	return filepath.Join(base, extractCacheRootName), nil
}

func extractCacheKey(src string) (string, error) {
	abs, err := filepath.Abs(src)
	if err != nil {
		return "", err
	}
	// Gradle artifact caches are checksum-addressed, so a canonical path is the
	// production identity we care about here. This keeps cache lookup at stat-free
	// path normalization cost instead of re-hashing jar contents on every search.
	// Tradeoff: if some non-Gradle workflow overwrites a jar in place at the same
	// path, this cache will intentionally keep reusing the existing extraction.
	clean := filepath.Clean(abs)
	sum := sha256.Sum256([]byte(clean))
	return fmt.Sprintf("%x", sum), nil
}

func isExtractedJarReady(dir string) bool {
	info, err := os.Stat(filepath.Join(dir, extractCacheReady))
	return err == nil && !info.IsDir()
}

func extractCacheLock(key string) *sync.Mutex {
	lock, _ := extractCacheLocks.LoadOrStore(key, &sync.Mutex{})
	return lock.(*sync.Mutex)
}

func extractJar(src, dest string) error {
	zr, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer zr.Close()

	for _, f := range zr.File {
		if f.FileInfo().IsDir() {
			continue
		}
		path, err := archiveChildPath(dest, f.Name)
		if err != nil {
			return fmt.Errorf("invalid path in archive: %s", f.Name)
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		out, err := os.Create(path)
		if err != nil {
			_ = rc.Close()
			return err
		}
		if _, err := io.Copy(out, rc); err != nil {
			_ = out.Close()
			_ = rc.Close()
			return err
		}
		_ = out.Close()
		_ = rc.Close()
	}
	return nil
}

func archiveChildPath(root, name string) (string, error) {
	root = filepath.Clean(root)
	path := filepath.Clean(filepath.Join(root, filepath.FromSlash(name)))
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return "", err
	}
	if rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path escapes root")
	}
	return path, nil
}

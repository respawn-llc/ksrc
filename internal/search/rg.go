package search

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/respawn-app/ksrc/internal/executil"
	"github.com/respawn-app/ksrc/internal/resolve"
)

type Match struct {
	FileID string
	File   string
	Line   int
	Column int
	Text   string
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

func parseRgLine(line string) (Match, bool) {
	if m, ok := parseRgMatchLine(line); ok {
		return m, true
	}
	return parseRgContextLine(line)
}

func parseRgMatchLine(line string) (Match, bool) {
	// file:line:col:match
	last := strings.LastIndex(line, ":")
	if last <= 0 {
		return Match{}, false
	}
	second := strings.LastIndex(line[:last], ":")
	if second <= 0 {
		return Match{}, false
	}
	third := strings.LastIndex(line[:second], ":")
	if third <= 0 {
		return Match{}, false
	}
	file := line[:third]
	lineStr := line[third+1 : second]
	colStr := line[second+1 : last]
	text := line[last+1:]
	ln, err := strconv.Atoi(lineStr)
	if err != nil {
		return Match{}, false
	}
	col, err := strconv.Atoi(colStr)
	if err != nil {
		return Match{}, false
	}
	return Match{File: file, Line: ln, Column: col, Text: text}, true
}

func parseRgContextLine(line string) (Match, bool) {
	// file-line-text (rg -C with --no-heading --line-number --column)
	last := strings.LastIndex(line, "-")
	if last <= 0 {
		return Match{}, false
	}
	second := strings.LastIndex(line[:last], "-")
	if second <= 0 {
		return Match{}, false
	}
	file := line[:second]
	lineStr := line[second+1 : last]
	text := line[last+1:]
	ln, err := strconv.Atoi(lineStr)
	if err != nil {
		return Match{}, false
	}
	return Match{File: file, Line: ln, Column: 0, Text: text}, true
}

type coordMapper func(string) (resolve.Coord, string, bool)

func parseRgOutput(stdout string, mapper coordMapper) []Match {
	matches := []Match{}
	for _, line := range strings.Split(stdout, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		m, ok := parseRgLine(line)
		if !ok {
			continue
		}
		coord, inner, ok := mapper(m.File)
		if !ok {
			continue
		}
		m.FileID = coord.String() + "!/" + inner
		matches = append(matches, m)
	}
	return matches
}

func mapToCoord(roots map[string]resolve.Coord, filePath string) (resolve.Coord, string, bool) {
	for root, coord := range roots {
		rel, err := filepath.Rel(root, filePath)
		if err != nil || strings.HasPrefix(rel, "..") {
			continue
		}
		rel = filepath.ToSlash(rel)
		return coord, rel, true
	}
	return resolve.Coord{}, "", false
}

func mapToCoordFromJarPath(jarPaths map[string]resolve.Coord, filePath string) (resolve.Coord, string, bool) {
	for jar, coord := range jarPaths {
		prefix := jar + ":"
		if !strings.HasPrefix(filePath, prefix) {
			continue
		}
		inner := strings.TrimPrefix(filePath, prefix)
		inner = strings.TrimPrefix(inner, "/")
		return coord, inner, true
	}
	return resolve.Coord{}, "", false
}

type searchStrategy struct {
	mode string
	run  func(context.Context, executil.Runner, Options) ([]Match, error)
}

func selectStrategy(ctx context.Context, runner executil.Runner) searchStrategy {
	if supportsZipSearch(ctx, runner) {
		return searchStrategy{mode: "zip", run: runZipSearch}
	}
	return searchStrategy{mode: "extract", run: runExtractSearch}
}

func runZipSearch(ctx context.Context, runner executil.Runner, opts Options) ([]Match, error) {
	jarPaths := make(map[string]resolve.Coord, len(opts.Jars))
	searchJars := make([]string, 0, len(opts.Jars))
	for _, j := range opts.Jars {
		jarPaths[j.Path] = j.Coord
		searchJars = append(searchJars, j.Path)
	}

	args := []string{"--search-zip", "--no-heading", "--line-number", "--column", "--color=never", "--with-filename", "-g", "*.kt"}
	args = append(args, opts.RGArgs...)
	args = append(args, "--")
	args = append(args, opts.Pattern)
	args = append(args, searchJars...)

	if opts.Report != nil {
		opts.Report(ExecPlan{
			Cmd:      "rg",
			Args:     args,
			JarCount: len(opts.Jars),
			Mode:     "zip",
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

	return parseRgOutput(stdout, func(filePath string) (resolve.Coord, string, bool) {
		return mapToCoordFromJarPath(jarPaths, filePath)
	}), nil
}

func runExtractSearch(ctx context.Context, runner executil.Runner, opts Options) ([]Match, error) {
	root, err := os.MkdirTemp("", "ksrc-search-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(root)

	extractRoots := make(map[string]resolve.Coord)
	searchDirs := make([]string, 0, len(opts.Jars))
	for i, j := range opts.Jars {
		dir := filepath.Join(root, fmt.Sprintf("jar-%d", i))
		if err := extractJar(j.Path, dir); err != nil {
			return nil, err
		}
		extractRoots[dir] = j.Coord
		searchDirs = append(searchDirs, dir)
	}

	args := []string{"--no-heading", "--line-number", "--column", "--color=never", "--with-filename", "-g", "*.kt"}
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

	return parseRgOutput(stdout, func(filePath string) (resolve.Coord, string, bool) {
		return mapToCoord(extractRoots, filePath)
	}), nil
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

var zipSupport = &zipSupportCache{}

type zipSupportCache struct {
	once      sync.Once
	supported bool
}

func (c *zipSupportCache) supports(ctx context.Context, runner executil.Runner) bool {
	c.once.Do(func() {
		c.supported = probeZipSearch(ctx, runner)
	})
	return c.supported
}

func supportsZipSearch(ctx context.Context, runner executil.Runner) bool {
	return zipSupport.supports(ctx, runner)
}

func resetZipSupportCacheForTests() {
	zipSupport = &zipSupportCache{}
}

func probeZipSearch(ctx context.Context, runner executil.Runner) bool {
	file, err := os.CreateTemp("", "ksrc-rg-probe-*.zip")
	if err != nil {
		return false
	}
	path := file.Name()
	zw := zip.NewWriter(file)
	w, err := zw.Create("probe.txt")
	if err == nil {
		_, _ = w.Write([]byte("ksrc-zip-probe"))
	}
	_ = zw.Close()
	_ = file.Close()
	defer os.Remove(path)

	args := []string{"--search-zip", "--no-heading", "--line-number", "--column", "--color=never", "-g", "*.txt", "ksrc-zip-probe", path}
	stdout, _, err := runner.Run(ctx, "", "rg", args...)
	if err != nil {
		return false
	}
	for _, line := range strings.Split(stdout, "\n") {
		m, ok := parseRgLine(strings.TrimSpace(line))
		if !ok {
			continue
		}
		if strings.HasPrefix(m.File, path+":") {
			return true
		}
	}
	return false
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
		path := filepath.Join(dest, filepath.FromSlash(f.Name))
		clean := filepath.Clean(path)
		if !strings.HasPrefix(clean, dest) {
			return fmt.Errorf("invalid path in archive: %s", f.Name)
		}
		if err := os.MkdirAll(filepath.Dir(clean), 0o755); err != nil {
			return err
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		out, err := os.Create(clean)
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

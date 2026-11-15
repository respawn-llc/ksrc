package cat

import (
	"archive/zip"
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type LineRange struct {
	Start int
	End   int
}

func ParseLineRange(value string) (*LineRange, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	matches := lineRangeRe.FindStringSubmatch(value)
	if matches == nil {
		return nil, fmt.Errorf("invalid line range: %q", value)
	}
	start, err := parsePositive(matches[1])
	if err != nil {
		return nil, err
	}
	end, err := parsePositive(matches[2])
	if err != nil {
		return nil, err
	}
	if end < start {
		return nil, fmt.Errorf("invalid line range: %q", value)
	}
	return &LineRange{Start: start, End: end}, nil
}

func parsePositive(s string) (int, error) {
	var n int
	_, err := fmt.Sscanf(strings.TrimSpace(s), "%d", &n)
	if err != nil || n <= 0 {
		return 0, fmt.Errorf("invalid line range value: %q", s)
	}
	return n, nil
}

var lineRangeRe = regexp.MustCompile(`^\s*(\d+)\s*(?:,|:|-|\.{2}|;|\s)\s*(\d+)\s*$`)

func normalizeArchivePath(path string) string {
	path = strings.ReplaceAll(path, "\\", "/")
	return strings.TrimPrefix(path, "/")
}

// HasSourceFile reports whether root, a directory or a zip, contains innerPath.
func HasSourceFile(root, innerPath string) (bool, error) {
	info, err := os.Stat(root)
	if err != nil {
		return false, err
	}
	if !info.IsDir() {
		return HasFileInZip(root, innerPath)
	}
	name, err := sourceRelativePath(innerPath)
	if err != nil {
		return false, err
	}
	tree, err := os.OpenRoot(root)
	if err != nil {
		return false, err
	}
	defer tree.Close()
	fileInfo, err := tree.Stat(name)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return fileInfo.Mode().IsRegular(), nil
}

// ReadSourceFile reads innerPath from a directory tree or a zip/jar.
func ReadSourceFile(root, innerPath string, lr *LineRange) ([]byte, error) {
	info, err := os.Stat(root)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return ReadFileFromZip(root, innerPath, lr)
	}
	name, err := sourceRelativePath(innerPath)
	if err != nil {
		return nil, err
	}
	tree, err := os.OpenRoot(root)
	if err != nil {
		return nil, err
	}
	defer tree.Close()
	file, err := tree.Open(name)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("file not found in source root: %s", normalizeArchivePath(innerPath))
		}
		return nil, err
	}
	defer file.Close()
	if lr == nil {
		return io.ReadAll(file)
	}
	return readRange(file, lr)
}

func sourceRelativePath(innerPath string) (string, error) {
	innerPath = normalizeArchivePath(innerPath)
	if innerPath == "" {
		return "", fmt.Errorf("invalid source path: %q", innerPath)
	}
	for _, part := range strings.Split(innerPath, "/") {
		if part == ".." {
			return "", fmt.Errorf("invalid source path: %s", innerPath)
		}
	}
	return filepath.FromSlash(innerPath), nil
}

// HasFileInZip reports whether a zip/jar contains the requested inner path.
// It only inspects directory metadata and never opens the entry body.
func HasFileInZip(zipPath, innerPath string) (bool, error) {
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		return false, err
	}
	defer zr.Close()

	innerPath = normalizeArchivePath(innerPath)
	for _, f := range zr.File {
		if f.Name == innerPath {
			return true, nil
		}
	}
	return false, nil
}

// ReadFileFromZip reads a file from a zip/jar and optionally slices by line range.
func ReadFileFromZip(zipPath, innerPath string, lr *LineRange) ([]byte, error) {
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, err
	}
	defer zr.Close()

	innerPath = normalizeArchivePath(innerPath)
	for _, f := range zr.File {
		if f.Name != innerPath {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return nil, err
		}
		defer rc.Close()
		if lr == nil {
			return io.ReadAll(rc)
		}
		return readRange(rc, lr)
	}
	return nil, fmt.Errorf("file not found in archive: %s", innerPath)
}

func readRange(r io.Reader, lr *LineRange) ([]byte, error) {
	scanner := bufio.NewScanner(r)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)
	var out bytes.Buffer
	line := 0
	for scanner.Scan() {
		line++
		if line < lr.Start {
			continue
		}
		if line > lr.End {
			break
		}
		out.WriteString(scanner.Text())
		out.WriteByte('\n')
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

package cat

import (
	"archive/zip"
	"bufio"
	"bytes"
	"fmt"
	"io"
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

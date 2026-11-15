package cache

import (
	"os"
	"path/filepath"
	"strings"
)

const (
	ExtractDirEnv = "KSRC_EXTRACT_CACHE_DIR"
	FileIDDirEnv  = "KSRC_FILEID_CACHE_DIR"
)

func ExtractRoot(userCacheDir func() (string, error)) (string, error) {
	if dir := strings.TrimSpace(os.Getenv(ExtractDirEnv)); dir != "" {
		return filepath.Abs(dir)
	}
	base, err := userCacheDir()
	if err != nil {
		return TempExtractRoot(), nil
	}
	return filepath.Join(base, "ksrc", "search-extracted-v1"), nil
}

func TempExtractRoot() string {
	return filepath.Join(os.TempDir(), "ksrc", "search-extracted-v1")
}

func FileIDRoot() (string, error) {
	if dir := strings.TrimSpace(os.Getenv(FileIDDirEnv)); dir != "" {
		return filepath.Abs(dir)
	}
	base, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "ksrc", "file-id-locations-v1"), nil
}

// Roots includes older cache versions and the extraction fallback, even when
// environment overrides redirect current writes elsewhere.
func Roots() ([]string, error) {
	roots := []string{filepath.Dir(TempExtractRoot())}
	base, err := os.UserCacheDir()
	if err != nil {
		return nil, err
	}
	roots = append(roots, filepath.Join(base, "ksrc"))
	for _, env := range []string{ExtractDirEnv, FileIDDirEnv} {
		if dir := strings.TrimSpace(os.Getenv(env)); dir != "" {
			roots = append(roots, dir)
		}
	}
	return roots, nil
}

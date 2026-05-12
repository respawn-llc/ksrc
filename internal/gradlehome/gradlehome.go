package gradlehome

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const EnvName = "GRADLE_USER_HOME"

type Source string

const (
	SourceExplicit Source = "explicit"
	SourceEnv      Source = "env"
	SourceDefault  Source = "default"
)

type Effective struct {
	Raw    string
	Path   string
	Source Source
}

func Resolve(explicit string, workDir string) (Effective, error) {
	if raw := strings.TrimSpace(explicit); raw != "" {
		path, err := cleanRelativeToWorkDir(raw, workDir)
		if err != nil {
			return Effective{}, err
		}
		return Effective{Raw: raw, Path: path, Source: SourceExplicit}, nil
	}
	if raw := strings.TrimSpace(os.Getenv(EnvName)); raw != "" {
		path, err := cleanRelativeToWorkDir(raw, workDir)
		if err != nil {
			return Effective{}, err
		}
		return Effective{Raw: raw, Path: path, Source: SourceEnv}, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return Effective{}, err
	}
	return Effective{Path: filepath.Join(home, ".gradle"), Source: SourceDefault}, nil
}

func ModulesCacheDir(explicit string, workDir string) (string, error) {
	home, err := Resolve(explicit, workDir)
	if err != nil {
		return "", err
	}
	return filepath.Join(home.Path, "caches", "modules-2", "files-2.1"), nil
}

func cleanRelativeToWorkDir(path string, workDir string) (string, error) {
	if filepath.IsAbs(path) {
		return filepath.Clean(path), nil
	}
	base := strings.TrimSpace(workDir)
	if base == "" {
		base = "."
	}
	absBase, err := filepath.Abs(base)
	if err != nil {
		return "", fmt.Errorf("resolve work dir %q: %w", workDir, err)
	}
	return filepath.Clean(filepath.Join(absBase, path)), nil
}

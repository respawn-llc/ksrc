package resolve

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func GradleCacheDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".gradle", "caches", "modules-2", "files-2.1"), nil
}

func FindCachedSources(group, artifact, version string) ([]SourceJar, error) {
	cacheDir, err := GradleCacheDir()
	if err != nil {
		return nil, err
	}
	groupPath := filepath.Join(cacheDir, filepath.FromSlash(strings.ReplaceAll(group, ".", "/")), artifact)
	if version == "" {
		version, err = HighestCachedVersion(groupPath)
		if err != nil {
			return nil, err
		}
	}
	versionDir := filepath.Join(groupPath, version)
	paths, err := findSourceJars(versionDir)
	if err != nil {
		return nil, err
	}
	if len(paths) == 0 {
		return nil, errors.New("sources not found in cache")
	}
	out := make([]SourceJar, 0, len(paths))
	for _, p := range paths {
		out = append(out, SourceJar{Coord: Coord{Group: group, Artifact: artifact, Version: version}, Path: p})
	}
	return out, nil
}

func FindAllCachedSources() ([]SourceJar, error) {
	cacheDir, err := GradleCacheDir()
	if err != nil {
		return nil, err
	}
	var out []SourceJar
	err = filepath.WalkDir(cacheDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(d.Name(), "-sources.jar") {
			return nil
		}
		rel, err := filepath.Rel(cacheDir, path)
		if err != nil {
			return err
		}
		parts := strings.Split(rel, string(filepath.Separator))
		if len(parts) < 5 {
			return nil
		}
		version := parts[len(parts)-3]
		artifact := parts[len(parts)-4]
		groupParts := parts[:len(parts)-4]
		if len(groupParts) == 0 {
			return nil
		}
		group := strings.Join(groupParts, ".")
		out = append(out, SourceJar{
			Coord: Coord{Group: group, Artifact: artifact, Version: version},
			Path:  path,
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func HighestCachedVersion(groupArtifactDir string) (string, error) {
	entries, err := os.ReadDir(groupArtifactDir)
	if err != nil {
		return "", err
	}
	versions := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			versions = append(versions, e.Name())
		}
	}
	if len(versions) == 0 {
		return "", fmt.Errorf("no cached versions found")
	}
	highest := versions[0]
	for _, v := range versions[1:] {
		if CompareVersion(v, highest) > 0 {
			highest = v
		}
	}
	return highest, nil
}

func findSourceJars(versionDir string) ([]string, error) {
	var jars []string
	err := filepath.WalkDir(versionDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if strings.HasSuffix(d.Name(), "-sources.jar") {
			jars = append(jars, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return jars, nil
}

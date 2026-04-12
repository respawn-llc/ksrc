package resolve

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

var errCachedSourcesNotFound = errors.New("sources not found in cache")

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
	groupPath := filepath.Join(cacheDir, group, artifact)
	if version == "" {
		version, err = HighestCachedSourceVersion(groupPath)
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
		return nil, errCachedSourcesNotFound
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
	versions, err := cachedVersionDirs(groupArtifactDir)
	if err != nil {
		return "", err
	}
	if len(versions) == 0 {
		return "", fmt.Errorf("no cached versions found")
	}
	sortCachedVersionsDesc(versions)
	return versions[0], nil
}

func HighestCachedSourceVersion(groupArtifactDir string) (string, error) {
	versions, err := cachedVersionDirs(groupArtifactDir)
	if err != nil {
		return "", err
	}
	sortCachedVersionsDesc(versions)
	for _, version := range versions {
		paths, err := findSourceJars(filepath.Join(groupArtifactDir, version))
		if err != nil {
			return "", err
		}
		if len(paths) > 0 {
			return version, nil
		}
	}
	return "", errCachedSourcesNotFound
}

func cachedVersionDirs(groupArtifactDir string) ([]string, error) {
	entries, err := os.ReadDir(groupArtifactDir)
	if err != nil {
		return nil, err
	}
	versions := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			versions = append(versions, e.Name())
		}
	}
	return versions, nil
}

func sortCachedVersionsDesc(versions []string) {
	sort.SliceStable(versions, func(i, j int) bool {
		cmp := CompareVersion(versions[i], versions[j])
		if cmp != 0 {
			return cmp > 0
		}
		if len(versions[i]) != len(versions[j]) {
			return len(versions[i]) < len(versions[j])
		}
		return versions[i] < versions[j]
	})
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

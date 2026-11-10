package resolve

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	// PlatformGroup is the synthetic group for Android SDK platform sources.
	PlatformGroup = "sdk.android"
	// PlatformArtifact is the synthetic artifact for Android SDK platform sources.
	PlatformArtifact = "platform"
)

// IsPlatformCoord reports whether coord addresses Android SDK platform sources.
func IsPlatformCoord(coord Coord) bool {
	return coord.Group == PlatformGroup && coord.Artifact == PlatformArtifact
}

// NormalizePlatformID accepts android-36, android-36.1, android-Baklava, or a numeric API such as 36.
func NormalizePlatformID(version string) (string, error) {
	version = strings.TrimSpace(version)
	if version == "" {
		return "", fmt.Errorf("platform version is required, for example sdk.android:platform:android-36")
	}
	if strings.Contains(version, "/") || strings.Contains(version, "\\") || strings.Contains(version, "..") {
		return "", fmt.Errorf("invalid platform version %q", version)
	}
	if strings.HasPrefix(version, "android-") && len(version) > len("android-") {
		return version, nil
	}
	if isNumericPlatform(version) {
		return "android-" + version, nil
	}
	return "", fmt.Errorf("invalid platform version %q; use android-36 or 36", version)
}

func isNumericPlatform(value string) bool {
	if value == "" {
		return false
	}
	dot := false
	for i, r := range value {
		switch {
		case r >= '0' && r <= '9':
		case r == '.' && i > 0 && !dot:
			dot = true
		default:
			return false
		}
	}
	return !strings.HasSuffix(value, ".")
}

// ExactPlatformCoord reports whether the selector is exactly sdk.android:platform[:version].
// Globs are left to Gradle resolution. A platform selector without a version is an error.
func ExactPlatformCoord(module, group, artifact, version string) (Coord, bool, error) {
	if strings.ContainsAny(module, "*?[]") || strings.ContainsAny(group, "*?[]") || strings.ContainsAny(artifact, "*?[]") || strings.ContainsAny(version, "*?[]") {
		return Coord{}, false, nil
	}
	coord, ok := SelectorToCoord(module, group, artifact, version)
	if !ok || !IsPlatformCoord(coord) {
		return Coord{}, false, nil
	}
	id, err := NormalizePlatformID(coord.Version)
	if err != nil {
		return Coord{}, true, err
	}
	coord.Version = id
	return coord, true, nil
}

// PlatformInstallCommand is the Android CLI command that installs full platform sources.
func PlatformInstallCommand(platformID string) string {
	return "android sdk install sources/" + platformID
}

// FindAndroidSDK locates an Android SDK directory.
// An explicit path wins. Otherwise ksrc reads sdk.dir, then android.dir, from the nearest
// local.properties at or above projectDir, then ANDROID_HOME, then ANDROID_SDK_ROOT.
func FindAndroidSDK(explicit, projectDir string) (string, error) {
	explicit = strings.TrimSpace(explicit)
	if explicit != "" {
		if !isDir(explicit) {
			return "", fmt.Errorf("Android SDK directory %q does not exist", explicit)
		}
		return explicit, nil
	}
	if dir, ok := sdkFromLocalProperties(projectDir); ok {
		return dir, nil
	}
	for _, env := range []string{"ANDROID_HOME", "ANDROID_SDK_ROOT"} {
		dir := strings.TrimSpace(os.Getenv(env))
		if dir != "" && isDir(dir) {
			return dir, nil
		}
	}
	return "", fmt.Errorf("Android SDK not found. Pass --android-home or set ANDROID_HOME")
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func sdkFromLocalProperties(start string) (string, bool) {
	if strings.TrimSpace(start) == "" {
		start = "."
	}
	dir, err := filepath.Abs(start)
	if err != nil {
		return "", false
	}
	for {
		props := filepath.Join(dir, "local.properties")
		if value, ok := readPropertiesValue(props, "sdk.dir"); ok {
			if resolved, ok := resolveSDKPath(dir, value); ok {
				return resolved, true
			}
		}
		if value, ok := readPropertiesValue(props, "android.dir"); ok {
			if resolved, ok := resolveSDKPath(dir, value); ok {
				return resolved, true
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}
		dir = parent
	}
}

func resolveSDKPath(base, value string) (string, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", false
	}
	if !filepath.IsAbs(value) {
		value = filepath.Join(base, value)
	}
	if !isDir(value) {
		return "", false
	}
	return value, true
}

func readPropertiesValue(path, key string) (string, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", false
	}
	prefix := key + "="
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(strings.TrimRight(line, "\r"))
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "!") {
			continue
		}
		if !strings.HasPrefix(line, prefix) {
			continue
		}
		value := strings.TrimSpace(strings.TrimPrefix(line, prefix))
		value = strings.Trim(value, `"'`)
		if value == "" {
			return "", false
		}
		return value, true
	}
	return "", false
}

// PlatformLoad is the readable roots for one Android platform.
type PlatformLoad struct {
	Coord    Coord
	Sources  []SourceJar
	Warnings []string
}

// LoadPlatform attaches full sources and the platform stub jar when they exist.
// Full sources are listed first so file lookup prefers them.
func LoadPlatform(sdkRoot, platformID string) (PlatformLoad, error) {
	id, err := NormalizePlatformID(platformID)
	if err != nil {
		return PlatformLoad{}, err
	}
	coord := Coord{Group: PlatformGroup, Artifact: PlatformArtifact, Version: id}
	sourcesDir := filepath.Join(sdkRoot, "sources", id)
	stubJar := filepath.Join(sdkRoot, "platforms", id, "android-stubs-src.jar")
	var sources []SourceJar
	if isDir(sourcesDir) {
		sources = append(sources, SourceJar{Coord: coord, Path: sourcesDir})
	}
	if info, err := os.Stat(stubJar); err == nil && info.Mode().IsRegular() {
		sources = append(sources, SourceJar{Coord: coord, Path: stubJar, Stub: true})
	}
	load := PlatformLoad{Coord: coord, Sources: sources}
	switch {
	case len(sources) == 0:
		load.Warnings = append(load.Warnings, fmt.Sprintf("Android platform sources for %s are not installed. Install them with: %s", id, PlatformInstallCommand(id)))
	case len(sources) == 1 && sources[0].Stub:
		load.Warnings = append(load.Warnings, fmt.Sprintf("Android platform %s is using stub sources at %s. Install full sources to read implementations: %s", id, stubJar, PlatformInstallCommand(id)))
	}
	return load, nil
}

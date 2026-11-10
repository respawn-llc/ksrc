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
// An explicit path wins. Otherwise ksrc reads sdk.dir, then android.dir, from
// local.properties in projectDir only, then ANDROID_HOME, then ANDROID_SDK_ROOT.
func FindAndroidSDK(explicit, projectDir string) (string, error) {
	explicit = strings.TrimSpace(explicit)
	if explicit != "" {
		if !isDir(explicit) {
			return "", fmt.Errorf("android SDK directory %q does not exist", explicit)
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
	return "", fmt.Errorf("android SDK not found. Pass --android-home or set ANDROID_HOME")
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func sdkFromLocalProperties(projectDir string) (string, bool) {
	if strings.TrimSpace(projectDir) == "" {
		projectDir = "."
	}
	dir, err := filepath.Abs(projectDir)
	if err != nil {
		return "", false
	}
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
	return "", false
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

// PlatformPackageRevision reads Pkg.Revision from source.properties next to a
// platform jar. Android replaces that jar in place and bumps the revision,
// so the same path can hold different bytes.
func PlatformPackageRevision(jarPath string) (string, bool) {
	dir := filepath.Dir(jarPath)
	if filepath.Base(filepath.Dir(dir)) != "platforms" || !strings.HasPrefix(filepath.Base(dir), "android-") {
		return "", false
	}
	value, ok := readPropertiesValue(filepath.Join(dir, "source.properties"), "Pkg.Revision")
	value = strings.TrimSpace(value)
	if !ok || value == "" {
		return "", false
	}
	return value, true
}

func readPropertiesValue(path, key string) (string, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", false
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(strings.TrimRight(line, "\r"))
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "!") {
			continue
		}
		sep := strings.IndexAny(line, "=:")
		if sep <= 0 {
			continue
		}
		if strings.TrimSpace(line[:sep]) != key {
			continue
		}
		value := unescapeProperties(strings.TrimSpace(line[sep+1:]))
		value = strings.Trim(value, `"'`)
		if value == "" {
			return "", false
		}
		return value, true
	}
	return "", false
}

func unescapeProperties(value string) string {
	var b strings.Builder
	b.Grow(len(value))
	for i := 0; i < len(value); i++ {
		if value[i] != '\\' || i+1 == len(value) {
			b.WriteByte(value[i])
			continue
		}
		switch value[i+1] {
		case '\\', ':', '=', ' ':
			b.WriteByte(value[i+1])
			i++
		default:
			b.WriteByte(value[i])
		}
	}
	return b.String()
}

// PlatformLoad is the readable roots for one Android platform.
type PlatformLoad struct {
	Coord    Coord
	Sources  []SourceJar
	Warnings []string
}

// LoadPlatform chooses one readable root for the platform.
// The sources directory wins. The platform stub jar is used only when that directory is absent.
func LoadPlatform(sdkRoot, platformID string) (PlatformLoad, error) {
	id, err := NormalizePlatformID(platformID)
	if err != nil {
		return PlatformLoad{}, err
	}
	coord := Coord{Group: PlatformGroup, Artifact: PlatformArtifact, Version: id}
	sourcesDir := filepath.Join(sdkRoot, "sources", id)
	stubJar := filepath.Join(sdkRoot, "platforms", id, "android-stubs-src.jar")
	load := PlatformLoad{Coord: coord}
	if isDir(sourcesDir) {
		load.Sources = []SourceJar{{Coord: coord, Path: sourcesDir}}
		return load, nil
	}
	if info, err := os.Stat(stubJar); err == nil && info.Mode().IsRegular() {
		load.Sources = []SourceJar{{Coord: coord, Path: stubJar}}
		load.Warnings = []string{fmt.Sprintf("Android platform %s is using stub sources at %s. Install full sources to read implementations: %s", id, stubJar, PlatformInstallCommand(id))}
		return load, nil
	}
	load.Warnings = []string{fmt.Sprintf("Android platform sources for %s are not installed. Install them with: %s", id, PlatformInstallCommand(id))}
	return load, nil
}

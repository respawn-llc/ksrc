package resolve

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/magiconair/properties"
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
		dir, err := filepath.Abs(explicit)
		if err != nil {
			return "", fmt.Errorf("resolve android SDK directory %q: %w", explicit, err)
		}
		if !isDir(dir) {
			return "", fmt.Errorf("android SDK directory %q does not exist", explicit)
		}
		return dir, nil
	}
	if dir, ok, err := sdkFromLocalProperties(projectDir); err != nil {
		return "", err
	} else if ok {
		return dir, nil
	}
	for _, env := range []string{"ANDROID_HOME", "ANDROID_SDK_ROOT"} {
		dir := strings.TrimSpace(os.Getenv(env))
		if dir == "" {
			continue
		}
		absolute, err := filepath.Abs(dir)
		if err != nil {
			return "", fmt.Errorf("resolve android SDK directory from %s: %w", env, err)
		}
		if isDir(absolute) {
			return absolute, nil
		}
	}
	return "", fmt.Errorf("android SDK not found. Pass --android-home or set ANDROID_HOME")
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func sdkFromLocalProperties(projectDir string) (string, bool, error) {
	if strings.TrimSpace(projectDir) == "" {
		projectDir = "."
	}
	dir, err := filepath.Abs(projectDir)
	if err != nil {
		return "", false, fmt.Errorf("resolve project directory %q: %w", projectDir, err)
	}
	props, err := readProperties(filepath.Join(dir, "local.properties"))
	if errors.Is(err, os.ErrNotExist) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	for _, key := range []string{"sdk.dir", "android.dir"} {
		if value, ok := props.Get(key); ok {
			if resolved, ok := resolveSDKPath(dir, value); ok {
				return resolved, true, nil
			}
		}
	}
	return "", false, nil
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
	props, err := readProperties(filepath.Join(dir, "source.properties"))
	if err != nil {
		return "", false
	}
	value, ok := props.Get("Pkg.Revision")
	value = strings.TrimSpace(value)
	if !ok || value == "" {
		return "", false
	}
	return value, true
}

func readProperties(path string) (*properties.Properties, error) {
	loader := properties.Loader{Encoding: properties.UTF8, DisableExpansion: true}
	props, err := loader.LoadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read properties %q: %w", path, err)
	}
	return props, nil
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

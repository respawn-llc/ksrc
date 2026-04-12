package resolve

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindCachedSourcesUsesDottedGroupDir(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	cacheDir := filepath.Join(home, ".gradle", "caches", "modules-2", "files-2.1")
	jarDir := filepath.Join(cacheDir, "org.jetbrains.kotlinx", "kotlinx-datetime", "1.2.3", "hash")
	if err := os.MkdirAll(jarDir, 0o755); err != nil {
		t.Fatalf("mkdir cache dir: %v", err)
	}
	jar := filepath.Join(jarDir, "kotlinx-datetime-1.2.3-sources.jar")
	if err := os.WriteFile(jar, []byte{}, 0o644); err != nil {
		t.Fatalf("write jar: %v", err)
	}

	sources, err := FindCachedSources("org.jetbrains.kotlinx", "kotlinx-datetime", "1.2.3")
	if err != nil {
		t.Fatalf("FindCachedSources error: %v", err)
	}
	if len(sources) != 1 {
		t.Fatalf("expected 1 source, got %d", len(sources))
	}
	if sources[0].Path != jar {
		t.Fatalf("expected jar path %q, got %q", jar, sources[0].Path)
	}
}

func TestFindCachedSourcesPrefersHighestQualifiedReleaseWhenVersionOmitted(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	cacheDir := filepath.Join(home, ".gradle", "caches", "modules-2", "files-2.1")
	versions := []string{"1.2.0-alpha01", "1.2.0-rc1", "1.2.0"}
	for _, version := range versions {
		jarDir := filepath.Join(cacheDir, "org.jetbrains.kotlinx", "kotlinx-datetime", version, "hash")
		if err := os.MkdirAll(jarDir, 0o755); err != nil {
			t.Fatalf("mkdir cache dir: %v", err)
		}
		jar := filepath.Join(jarDir, "kotlinx-datetime-"+version+"-sources.jar")
		if err := os.WriteFile(jar, []byte{}, 0o644); err != nil {
			t.Fatalf("write jar: %v", err)
		}
	}

	sources, err := FindCachedSources("org.jetbrains.kotlinx", "kotlinx-datetime", "")
	if err != nil {
		t.Fatalf("FindCachedSources error: %v", err)
	}
	if len(sources) != 1 {
		t.Fatalf("expected 1 source, got %d", len(sources))
	}
	if sources[0].Coord.Version != "1.2.0" {
		t.Fatalf("expected highest release version, got %q", sources[0].Coord.Version)
	}
}

func TestFindCachedSourcesSkipsHigherVersionsWithoutSourcesWhenVersionOmitted(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	cacheDir := filepath.Join(home, ".gradle", "caches", "modules-2", "files-2.1")
	emptyDir := filepath.Join(cacheDir, "org.jetbrains.kotlinx", "kotlinx-datetime", "1.2.0", "hash")
	if err := os.MkdirAll(emptyDir, 0o755); err != nil {
		t.Fatalf("mkdir empty cache dir: %v", err)
	}

	jarDir := filepath.Join(cacheDir, "org.jetbrains.kotlinx", "kotlinx-datetime", "1.2.0-rc1", "hash")
	if err := os.MkdirAll(jarDir, 0o755); err != nil {
		t.Fatalf("mkdir cache dir: %v", err)
	}
	jar := filepath.Join(jarDir, "kotlinx-datetime-1.2.0-rc1-sources.jar")
	if err := os.WriteFile(jar, []byte{}, 0o644); err != nil {
		t.Fatalf("write jar: %v", err)
	}

	sources, err := FindCachedSources("org.jetbrains.kotlinx", "kotlinx-datetime", "")
	if err != nil {
		t.Fatalf("FindCachedSources error: %v", err)
	}
	if len(sources) != 1 {
		t.Fatalf("expected 1 source, got %d", len(sources))
	}
	if sources[0].Coord.Version != "1.2.0-rc1" {
		t.Fatalf("expected highest source-bearing version, got %q", sources[0].Coord.Version)
	}
	if sources[0].Path != jar {
		t.Fatalf("expected jar path %q, got %q", jar, sources[0].Path)
	}
}

func TestFindCachedSourcesTreatsUnderscoreQualifierAsPrereleaseWhenVersionOmitted(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	cacheDir := filepath.Join(home, ".gradle", "caches", "modules-2", "files-2.1")
	versions := []string{"1.0_rc1", "1.0"}
	for _, version := range versions {
		jarDir := filepath.Join(cacheDir, "com.example", "demo", version, "hash")
		if err := os.MkdirAll(jarDir, 0o755); err != nil {
			t.Fatalf("mkdir cache dir: %v", err)
		}
		jar := filepath.Join(jarDir, "demo-"+version+"-sources.jar")
		if err := os.WriteFile(jar, []byte{}, 0o644); err != nil {
			t.Fatalf("write jar: %v", err)
		}
	}

	sources, err := FindCachedSources("com.example", "demo", "")
	if err != nil {
		t.Fatalf("FindCachedSources error: %v", err)
	}
	if len(sources) != 1 {
		t.Fatalf("expected 1 source, got %d", len(sources))
	}
	if sources[0].Coord.Version != "1.0" {
		t.Fatalf("expected final release version, got %q", sources[0].Coord.Version)
	}
}

func TestHighestCachedSourceVersionPrefersSemanticReleaseAndSkipsEmptyHigherDir(t *testing.T) {
	groupArtifactDir := t.TempDir()

	for _, version := range []string{"1.0_rc1", "1.0", "1.1"} {
		versionDir := filepath.Join(groupArtifactDir, version, "hash")
		if err := os.MkdirAll(versionDir, 0o755); err != nil {
			t.Fatalf("mkdir cache dir: %v", err)
		}
	}

	jar := filepath.Join(groupArtifactDir, "1.0", "hash", "demo-1.0-sources.jar")
	if err := os.WriteFile(jar, []byte{}, 0o644); err != nil {
		t.Fatalf("write jar: %v", err)
	}

	got, err := HighestCachedSourceVersion(groupArtifactDir)
	if err != nil {
		t.Fatalf("HighestCachedSourceVersion error: %v", err)
	}
	if got != "1.0" {
		t.Fatalf("expected final release version, got %q", got)
	}
}

func TestHighestCachedSourceVersionPrefersDotNumericOverPlusQualifiedVersion(t *testing.T) {
	groupArtifactDir := t.TempDir()

	for _, version := range []string{"1.0+1", "1.0.1"} {
		versionDir := filepath.Join(groupArtifactDir, version, "hash")
		if err := os.MkdirAll(versionDir, 0o755); err != nil {
			t.Fatalf("mkdir cache dir: %v", err)
		}
		jar := filepath.Join(versionDir, "demo-"+version+"-sources.jar")
		if err := os.WriteFile(jar, []byte{}, 0o644); err != nil {
			t.Fatalf("write jar: %v", err)
		}
	}

	got, err := HighestCachedSourceVersion(groupArtifactDir)
	if err != nil {
		t.Fatalf("HighestCachedSourceVersion error: %v", err)
	}
	if got != "1.0.1" {
		t.Fatalf("expected dot numeric version, got %q", got)
	}
}

func TestHighestCachedSourceVersionMissingDirectoryIsCacheMiss(t *testing.T) {
	groupArtifactDir := filepath.Join(t.TempDir(), "missing")

	_, err := HighestCachedSourceVersion(groupArtifactDir)
	if !IsCachedSourcesNotFound(err) {
		t.Fatalf("expected cache miss error, got %v", err)
	}
}

func TestHighestCachedSourceVersionSeparatorParity(t *testing.T) {
	testCases := []struct {
		name     string
		versions []string
		want     string
	}{
		{
			name:     "plus stays below dot numeric",
			versions: []string{"1.0+1", "1.0.1", "1.1"},
			want:     "1.0.1",
		},
		{
			name:     "underscore normalizes to dot numeric",
			versions: []string{"1.0_1", "1.0.1", "1.1"},
			want:     "1.0.1",
		},
		{
			name:     "mixed separators prefer canonical dot numeric",
			versions: []string{"1.0+1", "1.0_1", "1.0.1", "1.1"},
			want:     "1.0.1",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			groupArtifactDir := t.TempDir()

			for _, version := range testCase.versions {
				versionDir := filepath.Join(groupArtifactDir, version, "hash")
				if err := os.MkdirAll(versionDir, 0o755); err != nil {
					t.Fatalf("mkdir cache dir: %v", err)
				}
				if version == "1.1" {
					continue
				}
				jar := filepath.Join(versionDir, "demo-"+version+"-sources.jar")
				if err := os.WriteFile(jar, []byte{}, 0o644); err != nil {
					t.Fatalf("write jar: %v", err)
				}
			}

			got, err := HighestCachedSourceVersion(groupArtifactDir)
			if err != nil {
				t.Fatalf("HighestCachedSourceVersion error: %v", err)
			}
			if got != testCase.want {
				t.Fatalf("expected %q, got %q", testCase.want, got)
			}
		})
	}
}

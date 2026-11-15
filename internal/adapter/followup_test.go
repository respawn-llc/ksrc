package adapter

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/respawn-app/ksrc/internal/fileidcache"
	"github.com/respawn-app/ksrc/internal/resolve"
)

func TestPlatformFollowupReselectsSourcesFromCachedStub(t *testing.T) {
	t.Setenv("KSRC_FILEID_CACHE_DIR", t.TempDir())
	t.Setenv("ANDROID_HOME", "")
	t.Setenv("ANDROID_SDK_ROOT", "")
	sdk := t.TempDir()
	coord := resolve.Coord{Group: resolve.PlatformGroup, Artifact: resolve.PlatformArtifact, Version: "android-36"}
	inner := "android/app/Activity.java"
	fileID := resolve.FormatFileID(coord, inner)
	stub := filepath.Join(sdk, "platforms", coord.Version, "android-stubs-src.jar")
	if err := os.MkdirAll(filepath.Dir(stub), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := writeTestJar(stub, inner, "class Activity { /* Stub! */ }\n"); err != nil {
		t.Fatal(err)
	}
	if err := fileidcache.Register(fileID, stub); err != nil {
		t.Fatal(err)
	}
	assertLocation := func(want string) {
		t.Helper()
		location, found, err := FindFollowupFileIDLocation(fileID)
		if err != nil || !found {
			t.Fatalf("follow-up found=%v, err=%v", found, err)
		}
		if location.Source.Path != want || location.Source.Coord != coord || location.InnerPath != inner || location.FileID != fileID {
			t.Fatalf("follow-up = %+v, want source %q", location, want)
		}
	}
	assertLocation(stub)
	full := filepath.Join(sdk, "sources", coord.Version)
	if err := os.MkdirAll(filepath.Dir(filepath.Join(full, inner)), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(full, inner), []byte("class Activity { void implementation() {} }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	assertLocation(full)
	// A subsequent search records the full tree. Its removal must still allow
	// the same SDK's remaining stub package to be selected.
	if err := fileidcache.Register(fileID, full); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(full, full+"-uninstalled"); err != nil {
		t.Fatal(err)
	}
	assertLocation(stub)
}

func TestMavenFollowupDiscardsMissingTrackedJar(t *testing.T) {
	t.Setenv("KSRC_FILEID_CACHE_DIR", t.TempDir())
	t.Setenv("GRADLE_USER_HOME", t.TempDir())
	fileID := "com.example:demo:1.0.0!/Demo.kt"
	if err := fileidcache.Register(fileID, filepath.Join(t.TempDir(), "missing.jar")); err != nil {
		t.Fatal(err)
	}
	if _, found, err := FindFollowupFileIDLocation(fileID); err != nil || found {
		t.Fatalf("missing Maven jar should be a cache miss: found=%v err=%v", found, err)
	}
}

func TestPlatformFollowupUsesCachedFullSourcesWithoutSDKDiscovery(t *testing.T) {
	t.Setenv("KSRC_FILEID_CACHE_DIR", t.TempDir())
	t.Setenv("ANDROID_HOME", "")
	t.Setenv("ANDROID_SDK_ROOT", "")
	coord := resolve.Coord{Group: resolve.PlatformGroup, Artifact: resolve.PlatformArtifact, Version: "36"}
	fileID := resolve.FormatFileID(coord, "android/app/Activity.java")
	full := filepath.Join(t.TempDir(), "sources", "android-36")
	if err := os.MkdirAll(full, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := fileidcache.Register(fileID, full); err != nil {
		t.Fatal(err)
	}
	location, found, err := FindFollowupFileIDLocation(fileID)
	if err != nil || !found {
		t.Fatalf("follow-up found=%v, err=%v", found, err)
	}
	if location.Source.Path != full || location.Source.Coord.Version != "android-36" ||
		location.FileID != "sdk.android:platform:android-36!/android/app/Activity.java" {
		t.Fatalf("follow-up = %+v", location)
	}
}

func TestPlatformFollowupRejectsNoncanonicalCachedRoot(t *testing.T) {
	for _, relative := range []string{
		"sources/android-35",
		"other/android-36",
		"platforms/android-36/other.jar",
		"platforms/android-35/android-stubs-src.jar",
	} {
		t.Run(relative, func(t *testing.T) {
			t.Setenv("KSRC_FILEID_CACHE_DIR", t.TempDir())
			t.Setenv("GRADLE_USER_HOME", t.TempDir())
			path := filepath.Join(t.TempDir(), filepath.FromSlash(relative))
			if err := os.MkdirAll(path, 0o755); err != nil {
				t.Fatal(err)
			}
			fileID := "sdk.android:platform:android-36!/android/app/Activity.java"
			if err := fileidcache.Register(fileID, path); err != nil {
				t.Fatal(err)
			}
			_, found, err := FindFollowupFileIDLocation(fileID)
			if err != nil || found {
				t.Fatalf("noncanonical cached root: found=%v, err=%v", found, err)
			}
		})
	}
}

func TestMavenFollowupUsesCachedJarWithoutGradleCache(t *testing.T) {
	t.Setenv("KSRC_FILEID_CACHE_DIR", t.TempDir())
	t.Setenv("GRADLE_USER_HOME", t.TempDir())
	coord := resolve.Coord{Group: "com.example", Artifact: "demo", Version: "1.0.0"}
	inner := "com/example/Demo.kt"
	fileID := resolve.FormatFileID(coord, inner)
	jar := filepath.Join(t.TempDir(), "demo-sources.jar")
	if err := writeTestJar(jar, inner, "class Demo\n"); err != nil {
		t.Fatal(err)
	}
	if err := fileidcache.Register(fileID, jar); err != nil {
		t.Fatal(err)
	}
	location, found, err := FindFollowupFileIDLocation(fileID)
	if err != nil || !found {
		t.Fatalf("follow-up found=%v, err=%v", found, err)
	}
	if location.Source.Path != jar || location.Source.Coord != coord || location.FileID != fileID || location.InnerPath != inner {
		t.Fatalf("follow-up = %+v", location)
	}
}

func TestFindFollowupFileIDLocationFallsBackWhenFileIDCacheLookupFails(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("GRADLE_USER_HOME", "")

	brokenCacheRoot := filepath.Join(t.TempDir(), "cache-root-file")
	if err := os.WriteFile(brokenCacheRoot, []byte("not a directory"), 0o644); err != nil {
		t.Fatalf("write broken cache root: %v", err)
	}
	t.Setenv("KSRC_FILEID_CACHE_DIR", brokenCacheRoot)

	coord := resolve.Coord{Group: "com.example", Artifact: "demo", Version: "1.0.0"}
	inner := "com/example/Demo.kt"
	jarDir := filepath.Join(home, ".gradle", "caches", "modules-2", "files-2.1", coord.Group, coord.Artifact, coord.Version, "hash")
	if err := os.MkdirAll(jarDir, 0o755); err != nil {
		t.Fatalf("mkdir jar dir: %v", err)
	}
	jarPath := filepath.Join(jarDir, coord.Artifact+"-"+coord.Version+"-sources.jar")
	if err := writeTestJar(jarPath, inner, "class Demo\n"); err != nil {
		t.Fatalf("write jar: %v", err)
	}

	location, found, err := FindFollowupFileIDLocation(resolve.FormatFileID(coord, inner))
	if err != nil {
		t.Fatalf("FindFollowupFileIDLocation error: %v", err)
	}
	if !found {
		t.Fatal("expected follow-up file-id lookup to fall back to Gradle cache")
	}
	if location.Source.Path != jarPath {
		t.Fatalf("unexpected jar path: %q", location.Source.Path)
	}
	if location.InnerPath != inner {
		t.Fatalf("unexpected inner path: %q", location.InnerPath)
	}
	if location.FileID != resolve.FormatFileID(coord, inner) {
		t.Fatalf("unexpected file id: %q", location.FileID)
	}
}

func TestFindFollowupFileIDLocationUsesCacheOptions(t *testing.T) {
	defaultHome := t.TempDir()
	customHome := filepath.Join(t.TempDir(), "custom-gradle")
	t.Setenv("HOME", defaultHome)
	t.Setenv("USERPROFILE", defaultHome)
	t.Setenv("KSRC_FILEID_CACHE_DIR", filepath.Join(t.TempDir(), "fileid-cache"))

	coord := resolve.Coord{Group: "com.example", Artifact: "demo", Version: "1.0.0"}
	inner := "com/example/Demo.kt"
	jarDir := filepath.Join(customHome, "caches", "modules-2", "files-2.1", coord.Group, coord.Artifact, coord.Version, "hash")
	if err := os.MkdirAll(jarDir, 0o755); err != nil {
		t.Fatalf("mkdir jar dir: %v", err)
	}
	jarPath := filepath.Join(jarDir, coord.Artifact+"-"+coord.Version+"-sources.jar")
	if err := writeTestJar(jarPath, inner, "class Demo\n"); err != nil {
		t.Fatalf("write jar: %v", err)
	}

	location, found, err := FindFollowupFileIDLocationWithOptions(resolve.FormatFileID(coord, inner), resolve.CacheOptions{GradleUserHome: customHome})
	if err != nil {
		t.Fatalf("FindFollowupFileIDLocationWithOptions error: %v", err)
	}
	if !found {
		t.Fatal("expected follow-up file-id lookup to find custom Gradle cache")
	}
	if location.Source.Path != jarPath {
		t.Fatalf("unexpected jar path: %q", location.Source.Path)
	}
}

package search

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExtractJarCachedRefreshesAndroidPlatformRevision(t *testing.T) {
	cacheDir := filepath.Join(t.TempDir(), "cache")
	t.Setenv(extractCacheDirEnv, cacheDir)
	platform := filepath.Join(t.TempDir(), "platforms", "android-36")
	if err := os.MkdirAll(platform, 0o755); err != nil {
		t.Fatal(err)
	}
	jarPath := filepath.Join(platform, "android-stubs-src.jar")
	props := filepath.Join(platform, "source.properties")
	if err := os.WriteFile(props, []byte("Pkg.Revision=1\nAndroidVersion.ApiLevel=36\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	writeTestJar(t, jarPath, map[string]string{"android/app/Notification.java": "revision one\n"})

	first, err := extractJarCached(jarPath)
	if err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(filepath.Join(first, "android", "app", "Notification.java"))
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "revision one\n" {
		t.Fatalf("first extract = %q", body)
	}

	again, err := extractJarCached(jarPath)
	if err != nil {
		t.Fatal(err)
	}
	if again != first {
		t.Fatalf("same revision extracted twice: %q and %q", first, again)
	}

	writeTestJar(t, jarPath, map[string]string{"android/app/Notification.java": "revision six\n"})
	if err := os.WriteFile(props, []byte("Pkg.Revision=6\nAndroidVersion.ApiLevel=36\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	sixth, err := extractJarCached(jarPath)
	if err != nil {
		t.Fatal(err)
	}
	if sixth == first {
		t.Fatal("revision 6 reused the revision 1 extraction")
	}
	body, err = os.ReadFile(filepath.Join(sixth, "android", "app", "Notification.java"))
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "revision six\n" {
		t.Fatalf("sixth extract = %q", body)
	}
}

package depupdate

import (
	"path/filepath"
	"testing"
)

func TestSyncGradleFixtureDepsUsesCatalogVersion(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	catalog := filepath.Join(dir, "libs.versions.toml")
	fixture := filepath.Join(dir, "build.gradle")
	writeFile(t, catalog, "[versions]\nkotlinx-datetime = \"0.9.0\"\n")
	writeFile(t, fixture, "dependencies {\n    api 'org.jetbrains.kotlinx:kotlinx-datetime:0.1.0'\n}\n")

	if err := SyncGradleFixtureDeps(catalog, fixture); err != nil {
		t.Fatalf("SyncGradleFixtureDeps error: %v", err)
	}
	want := "dependencies {\n    api 'org.jetbrains.kotlinx:kotlinx-datetime:0.9.0'\n}\n"
	if got := readFile(t, fixture); got != want {
		t.Fatalf("unexpected fixture content:\n%s", got)
	}
}

func TestKotlinxDatetimeVersionRequiresCatalogEntry(t *testing.T) {
	t.Parallel()

	if _, err := KotlinxDatetimeVersion("libs.versions.toml", "[versions]\n"); err == nil {
		t.Fatal("expected missing version error")
	}
}

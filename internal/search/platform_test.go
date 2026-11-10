package search

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/respawn-app/ksrc/internal/resolve"
)

func TestSuppressPlatformStubMatchesDropsPathPresentInFullSources(t *testing.T) {
	root := t.TempDir()
	full := filepath.Join(root, "android", "app", "Activity.java")
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte("class Activity {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	coord := resolve.Coord{Group: resolve.PlatformGroup, Artifact: resolve.PlatformArtifact, Version: "android-36"}
	stubPath := filepath.Join(root, "android-stubs-src.jar")
	matches := []Match{
		{JarPath: filepath.Join(root, "sources"), Coord: coord, InnerPath: "android/app/Activity.java", Text: "class Activity {}"},
		{JarPath: stubPath, Coord: coord, InnerPath: "android/app/Activity.java", Text: "Stub!"},
		{JarPath: stubPath, Coord: coord, InnerPath: "android/app/OnlyStub.java", Text: "only stub"},
	}
	jars := []resolve.SourceJar{
		{Coord: coord, Path: root},
		{Coord: coord, Path: stubPath, Stub: true},
	}
	got := suppressPlatformStubMatches(matches, jars)
	if len(got) != 2 {
		t.Fatalf("kept %#v", got)
	}
	if got[0].Text != "class Activity {}" || got[1].InnerPath != "android/app/OnlyStub.java" {
		t.Fatalf("kept %#v", got)
	}
}

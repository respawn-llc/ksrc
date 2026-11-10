package resolve

import (
	"archive/zip"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNormalizePlatformID(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in   string
		want string
	}{
		{in: "android-36", want: "android-36"},
		{in: "36", want: "android-36"},
		{in: "android-36.1", want: "android-36.1"},
		{in: "36.1", want: "android-36.1"},
		{in: "android-Baklava", want: "android-Baklava"},
	}
	for _, tc := range cases {
		got, err := NormalizePlatformID(tc.in)
		if err != nil {
			t.Fatalf("NormalizePlatformID(%q) error = %v", tc.in, err)
		}
		if got != tc.want {
			t.Fatalf("NormalizePlatformID(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
	if _, err := NormalizePlatformID(""); err == nil {
		t.Fatal("expected empty platform version to fail")
	}
	if _, err := NormalizePlatformID("../android-36"); err == nil {
		t.Fatal("expected path platform version to fail")
	}
}

func TestExactPlatformCoordRequiresVersion(t *testing.T) {
	t.Parallel()
	if _, ok, err := ExactPlatformCoord("org.example:lib:1.0.0", "", "", ""); err != nil || ok {
		t.Fatalf("maven selector ok=%v err=%v", ok, err)
	}
	coord, ok, err := ExactPlatformCoord("sdk.android:platform:36", "", "", "")
	if err != nil || !ok {
		t.Fatalf("platform selector ok=%v err=%v", ok, err)
	}
	if coord.String() != "sdk.android:platform:android-36" {
		t.Fatalf("coord = %s", coord)
	}
	if _, ok, err := ExactPlatformCoord("sdk.android:platform", "", "", ""); err == nil || !ok {
		t.Fatalf("missing version ok=%v err=%v", ok, err)
	}
	if _, ok, err := ExactPlatformCoord("sdk.android:*", "", "", ""); err != nil || ok {
		t.Fatalf("glob ok=%v err=%v", ok, err)
	}
}

func TestLoadPlatformPrefersFullSourcesAndFallsBackToStubs(t *testing.T) {
	sdk := t.TempDir()
	writePlatformFile(t, filepath.Join(sdk, "sources", "android-36", "android", "app", "Activity.java"), "class Activity {}\n")
	writeStubJar(t, filepath.Join(sdk, "platforms", "android-36", "android-stubs-src.jar"))

	both, err := LoadPlatform(sdk, "android-36")
	if err != nil {
		t.Fatalf("LoadPlatform both: %v", err)
	}
	if len(both.Warnings) != 0 {
		t.Fatalf("warnings = %#v", both.Warnings)
	}
	if len(both.Sources) != 2 || both.Sources[0].Stub || !both.Sources[1].Stub {
		t.Fatalf("sources = %#v", both.Sources)
	}

	if err := os.RemoveAll(filepath.Join(sdk, "sources")); err != nil {
		t.Fatal(err)
	}
	stubs, err := LoadPlatform(sdk, "36")
	if err != nil {
		t.Fatalf("LoadPlatform stubs: %v", err)
	}
	if len(stubs.Sources) != 1 || !stubs.Sources[0].Stub {
		t.Fatalf("stub sources = %#v", stubs.Sources)
	}
	if len(stubs.Warnings) != 1 || !strings.Contains(stubs.Warnings[0], "android sdk install sources/android-36") || !strings.Contains(stubs.Warnings[0], "stub sources") {
		t.Fatalf("stub warning = %q", stubs.Warnings)
	}

	if err := os.Remove(stubs.Sources[0].Path); err != nil {
		t.Fatal(err)
	}
	missing, err := LoadPlatform(sdk, "android-36")
	if err != nil {
		t.Fatalf("LoadPlatform missing: %v", err)
	}
	if len(missing.Sources) != 0 || len(missing.Warnings) != 1 || !strings.Contains(missing.Warnings[0], "not installed") || !strings.Contains(missing.Warnings[0], "android sdk install sources/android-36") {
		t.Fatalf("missing = %#v", missing)
	}
}

func TestFindAndroidSDKExplicitAndEnv(t *testing.T) {
	sdk := t.TempDir()
	project := t.TempDir()
	if err := os.WriteFile(filepath.Join(project, "local.properties"), []byte("sdk.dir="+sdk+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	other := t.TempDir()
	got, err := FindAndroidSDK(other, project)
	if err != nil || got != other {
		t.Fatalf("explicit = %q err=%v", got, err)
	}
	got, err = FindAndroidSDK("", project)
	if err != nil || got != sdk {
		t.Fatalf("local.properties = %q err=%v", got, err)
	}

	t.Setenv("ANDROID_HOME", sdk)
	t.Setenv("ANDROID_SDK_ROOT", other)
	got, err = FindAndroidSDK("", t.TempDir())
	if err != nil || got != sdk {
		t.Fatalf("ANDROID_HOME = %q err=%v", got, err)
	}
}

func writePlatformFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeStubJar(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	zw := zip.NewWriter(file)
	entry, err := zw.Create("android/app/Activity.java")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := entry.Write([]byte("public class Activity { throw new RuntimeException(\"Stub!\"); }\n")); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
}

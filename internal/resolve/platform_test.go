package resolve

import (
	"archive/zip"
	"os"
	"path/filepath"
	"strconv"
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
	if len(both.Sources) != 1 || both.Sources[0].Path != filepath.Join(sdk, "sources", "android-36") {
		t.Fatalf("sources = %#v", both.Sources)
	}

	if err := os.RemoveAll(filepath.Join(sdk, "sources")); err != nil {
		t.Fatal(err)
	}
	stubs, err := LoadPlatform(sdk, "36")
	if err != nil {
		t.Fatalf("LoadPlatform stubs: %v", err)
	}
	if len(stubs.Sources) != 1 || stubs.Sources[0].Path != filepath.Join(sdk, "platforms", "android-36", "android-stubs-src.jar") {
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
	if err := os.WriteFile(filepath.Join(project, "local.properties"), []byte("sdk.dir="+propertyPath(sdk)+"\n"), 0o644); err != nil {
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

	t.Setenv("ANDROID_HOME", "")
	bare := t.TempDir()
	got, err = FindAndroidSDK("", bare)
	if err != nil || got != other {
		t.Fatalf("ANDROID_SDK_ROOT = %q err=%v", got, err)
	}
}

func TestReadPropertiesSyntax(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "local.properties")
	body := "# comment\r\n\n" +
		"sdk.dir = /opt/android\n" +
		"win.dir=C\\:\\\\Users\\\\me\\\\sdk\n" +
		"other.dir:/opt/other\n" +
		`unicode.dir=C\:\\Users\\\u041d\u0438\u043a\u0438\u0442\u0430\\sdk` + "\n" +
		"duplicate=first\nduplicate=last\n" +
		"continued=/opt/an\\\n    droid\n" +
		"space.dir /opt/space\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	props, err := readProperties(path)
	if err != nil {
		t.Fatal(err)
	}
	for key, want := range map[string]string{
		"sdk.dir":     "/opt/android",
		"win.dir":     `C:\Users\me\sdk`,
		"other.dir":   "/opt/other",
		"unicode.dir": `C:\Users\Никита\sdk`,
		"duplicate":   "last",
		"continued":   "/opt/android",
		"space.dir":   "/opt/space",
	} {
		if got, ok := props.Get(key); !ok || got != want {
			t.Errorf("%s = %q, %v; want %q", key, got, ok, want)
		}
	}
}

func TestFindAndroidSDKAcceptsSpacedSDKDir(t *testing.T) {
	sdk := t.TempDir()
	project := t.TempDir()
	if err := os.WriteFile(filepath.Join(project, "local.properties"), []byte("sdk.dir = "+propertyPath(sdk)+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ANDROID_HOME", "")
	t.Setenv("ANDROID_SDK_ROOT", "")
	got, err := FindAndroidSDK("", project)
	if err != nil || got != sdk {
		t.Fatalf("spaced sdk.dir = %q err=%v", got, err)
	}
}

func TestFindAndroidSDKIgnoresParentLocalProperties(t *testing.T) {
	parentSDK := t.TempDir()
	projectSDK := t.TempDir()
	parent := t.TempDir()
	project := filepath.Join(parent, "app")
	if err := os.MkdirAll(project, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(parent, "local.properties"), []byte("sdk.dir="+propertyPath(parentSDK)+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ANDROID_HOME", projectSDK)
	t.Setenv("ANDROID_SDK_ROOT", "")
	got, err := FindAndroidSDK("", project)
	if err != nil || got != projectSDK {
		t.Fatalf("parent local.properties leaked: got %q err=%v", got, err)
	}
}

func TestFindAndroidSDKRelativeInvocationPaths(t *testing.T) {
	cwd := t.TempDir()
	t.Chdir(cwd)
	sdk := filepath.Join(cwd, "sdk")
	project := filepath.Join(cwd, "project")
	for _, dir := range []string{sdk, project} {
		if err := os.Mkdir(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for _, source := range []string{"explicit", "ANDROID_HOME", "ANDROID_SDK_ROOT"} {
		t.Run(source, func(t *testing.T) {
			t.Setenv("ANDROID_HOME", "")
			t.Setenv("ANDROID_SDK_ROOT", "")
			explicit := ""
			if source == "explicit" {
				explicit = "sdk"
			} else {
				t.Setenv(source, "sdk")
			}
			got, err := FindAndroidSDK(explicit, project)
			if err != nil || got != sdk {
				t.Fatalf("SDK = %q err=%v, want %q", got, err, sdk)
			}
		})
	}
}

func TestFindAndroidSDKLocalPropertiesSemantics(t *testing.T) {
	for _, tc := range []struct {
		body string
		sdk  string
	}{
		{"sdk.dir=missing\nsdk.dir=sdk\n", "sdk"},
		{"sdk.dir=s\\\n  dk\n", "sdk"},
		{"sdk.dir=missing\nandroid.dir=sdk\n", "sdk"},
		{"sdk.dir=sdk\nandroid.dir=other\n", "sdk"},
		{"sdk.dir=\\u0441dk\n", "сdk"},
		{"sdk.dir=${KSRC_LITERAL_SDK}\n", "${KSRC_LITERAL_SDK}"},
	} {
		t.Run(tc.body, func(t *testing.T) {
			project := t.TempDir()
			sdk := filepath.Join(project, tc.sdk)
			if err := os.Mkdir(sdk, 0o755); err != nil {
				t.Fatal(err)
			}
			writePlatformFile(t, filepath.Join(project, "local.properties"), tc.body)
			t.Setenv("ANDROID_HOME", t.TempDir())
			t.Setenv("ANDROID_SDK_ROOT", "")
			t.Setenv("KSRC_LITERAL_SDK", "missing")
			got, err := FindAndroidSDK("", project)
			if err != nil || got != sdk {
				t.Fatalf("SDK = %q err=%v, want %q", got, err, sdk)
			}
		})
	}
}

func TestFindAndroidSDKReportsPropertiesErrors(t *testing.T) {
	for _, kind := range []string{"malformed", "unreadable"} {
		t.Run(kind, func(t *testing.T) {
			project := filepath.Join(t.TempDir(), `with\backslash`)
			if err := os.MkdirAll(project, 0o755); err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(project, "local.properties")
			if kind == "malformed" {
				writePlatformFile(t, path, `sdk.dir=\uZZZZ`)
			} else if err := os.Mkdir(path, 0o755); err != nil {
				t.Fatal(err)
			}
			t.Setenv("ANDROID_HOME", t.TempDir())
			_, err := FindAndroidSDK("", project)
			if err == nil || !strings.Contains(err.Error(), strconv.Quote(path)) {
				t.Fatalf("expected diagnostic for %s, got %v", path, err)
			}
			explicit := t.TempDir()
			if got, err := FindAndroidSDK(explicit, project); err != nil || got != explicit {
				t.Fatalf("explicit override = %q err=%v, want %q", got, err, explicit)
			}
		})
	}
}

func propertyPath(path string) string {
	return strings.NewReplacer(`\`, `\\`, `:`, `\:`).Replace(path)
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

package cli

import (
	"archive/zip"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/respawn-app/ksrc/internal/gradle"
	"github.com/respawn-app/ksrc/internal/testutil"
)

func TestSearchAndCatIntegration(t *testing.T) {
	app := NewApp()
	if _, err := app.Runner.LookPath("rg"); err != nil {
		t.Skip("rg not available")
	}

	projectDir := filepath.Clean(filepath.Join("..", "..", "testdata", "fixture"))
	jarPath := filepath.Join(t.TempDir(), "kotlinx-datetime-sources.jar")
	inner := "kotlinx/datetime/LocalDate.kt"

	if err := writeTestJar(jarPath, inner, "before\npublic class LocalDate\nafter\n"); err != nil {
		t.Fatalf("write jar: %v", err)
	}

	setTestJarEnv(t, jarPath)

	searchOut, err := runCommand(app, []string{"search", "public class LocalDate", "--module", "org.jetbrains.kotlinx:kotlinx-datetime", "--project", projectDir})
	if err != nil {
		t.Fatalf("search error: %v", err)
	}
	if !strings.Contains(searchOut, "org.jetbrains.kotlinx:kotlinx-datetime:0.7.1!/"+inner) {
		t.Fatalf("unexpected search output: %s", searchOut)
	}

	// Extract file-id from search output
	fields := strings.Fields(searchOut)
	if len(fields) == 0 {
		t.Fatalf("no fields in search output")
	}
	fileID := fields[0]

	catOut, err := runCommand(app, []string{"cat", fileID, "--project", projectDir, "--lines", "2,2"})
	if err != nil {
		t.Fatalf("cat error: %v", err)
	}
	if strings.TrimSpace(catOut) != "public class LocalDate" {
		t.Fatalf("unexpected cat output: %q", catOut)
	}
}

func TestStaticFixtureUsesGradleInternalTaskName(t *testing.T) {
	content, err := os.ReadFile(filepath.Clean(filepath.Join("..", "..", "testdata", "fixture", "gradlew")))
	if err != nil {
		t.Fatalf("read fixture gradlew: %v", err)
	}
	if !strings.Contains(string(content), fmt.Sprintf(`"$arg" = "%s"`, gradle.KsrcGradleTaskName())) {
		t.Fatalf("fixture gradlew does not use internal task name %q", gradle.KsrcGradleTaskName())
	}
}

func TestSearchAndCatIntegrationReusesTrackedFileIDAcrossCWD(t *testing.T) {
	app := NewApp()
	if _, err := app.Runner.LookPath("rg"); err != nil {
		t.Skip("rg not available")
	}

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	root := filepath.Clean(filepath.Join(wd, "..", ".."))
	projectDir := filepath.Join(root, "testdata", "fixture")
	jarPath := filepath.Join(t.TempDir(), "kotlinx-datetime-sources.jar")
	inner := "kotlinx/datetime/LocalDate.kt"

	if err := writeTestJar(jarPath, inner, "before\npublic class LocalDate\nafter\n"); err != nil {
		t.Fatalf("write jar: %v", err)
	}
	home := filepath.Join(t.TempDir(), "home")
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("GRADLE_USER_HOME", "")

	setTestJarEnv(t, jarPath)
	t.Setenv("KSRC_FILEID_CACHE_DIR", filepath.Join(t.TempDir(), "fileid-cache"))

	previousWD := wd
	if err := os.Chdir(t.TempDir()); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer func() {
		_ = os.Chdir(previousWD)
	}()

	searchOut, err := runCommand(NewApp(), []string{"search", "public class LocalDate", "--module", "org.jetbrains.kotlinx:kotlinx-datetime", "--project", projectDir})
	if err != nil {
		t.Fatalf("search error: %v", err)
	}
	fields := strings.Fields(searchOut)
	if len(fields) == 0 {
		t.Fatalf("no fields in search output")
	}
	fileID := fields[0]

	catOut, err := runCommand(NewApp(), []string{"cat", fileID, "--lines", "2,2"})
	if err != nil {
		t.Fatalf("cat error: %v", err)
	}
	if strings.TrimSpace(catOut) != "public class LocalDate" {
		t.Fatalf("unexpected cat output: %q", catOut)
	}
}

func TestSearchAndWhereFileIDIntegrationReusesTrackedFileIDAcrossCWD(t *testing.T) {
	app := NewApp()
	if _, err := app.Runner.LookPath("rg"); err != nil {
		t.Skip("rg not available")
	}

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	root := filepath.Clean(filepath.Join(wd, "..", ".."))
	projectDir := filepath.Join(root, "testdata", "fixture")
	jarPath := filepath.Join(t.TempDir(), "kotlinx-datetime-sources.jar")
	inner := "kotlinx/datetime/LocalDate.kt"

	if err := writeTestJar(jarPath, inner, "before\npublic class LocalDate\nafter\n"); err != nil {
		t.Fatalf("write jar: %v", err)
	}

	setTestJarEnv(t, jarPath)
	t.Setenv("KSRC_FILEID_CACHE_DIR", filepath.Join(t.TempDir(), "fileid-cache"))

	previousWD := wd
	if err := os.Chdir(t.TempDir()); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer func() {
		_ = os.Chdir(previousWD)
	}()

	searchOut, err := runCommand(NewApp(), []string{"search", "public class LocalDate", "--module", "org.jetbrains.kotlinx:kotlinx-datetime", "--project", projectDir})
	if err != nil {
		t.Fatalf("search error: %v", err)
	}
	fields := strings.Fields(searchOut)
	if len(fields) == 0 {
		t.Fatalf("no fields in search output")
	}
	fileID := fields[0]

	whereOut, err := runCommand(NewApp(), []string{"where", fileID})
	if err != nil {
		t.Fatalf("where error: %v", err)
	}
	want := fileID + "|" + jarPath + "\n"
	if whereOut != want {
		t.Fatalf("unexpected where output:\nwant: %q\n got: %q", want, whereOut)
	}
}

func TestWhereFileIDIgnoresStaleTrackedLocationAndFallsBack(t *testing.T) {
	app := NewApp()
	if _, err := app.Runner.LookPath("rg"); err != nil {
		t.Skip("rg not available")
	}

	projectDir := t.TempDir()
	jarDir := t.TempDir()
	jarPath := filepath.Join(jarDir, "kotlinx-datetime-sources.jar")
	movedJarPath := filepath.Join(jarDir, "kotlinx-datetime-sources-moved.jar")
	inner := "kotlinx/datetime/LocalDate.kt"
	module := "com.example:fixture:1.0.0-test"

	if err := writeTestJar(jarPath, inner, "before\npublic class LocalDate\nafter\n"); err != nil {
		t.Fatalf("write jar: %v", err)
	}
	if err := writeFakeGradleProjectWithCoord(projectDir, "com.example", "fixture", "1.0.0-test"); err != nil {
		t.Fatalf("write fake project: %v", err)
	}

	setTestJarEnv(t, jarPath)
	t.Setenv("KSRC_FILEID_CACHE_DIR", filepath.Join(t.TempDir(), "fileid-cache"))

	searchOut, err := runCommand(NewApp(), []string{"search", "public class LocalDate", "--module", module, "--project", projectDir})
	if err != nil {
		t.Fatalf("search error: %v", err)
	}
	fields := strings.Fields(searchOut)
	if len(fields) == 0 {
		t.Fatalf("no fields in search output")
	}
	fileID := fields[0]

	if err := os.Rename(jarPath, movedJarPath); err != nil {
		t.Fatalf("move jar: %v", err)
	}
	t.Setenv("KSRC_TEST_JAR", movedJarPath)

	whereOut, err := runCommand(NewApp(), []string{"where", fileID, "--project", projectDir})
	if err != nil {
		t.Fatalf("where error after stale cache entry: %v", err)
	}
	want := fileID + "|" + movedJarPath + "\n"
	if whereOut != want {
		t.Fatalf("unexpected where output after stale cache entry:\nwant: %q\n got: %q", want, whereOut)
	}

	catOut, err := runCommand(NewApp(), []string{"cat", fileID, "--project", projectDir, "--lines", "2,2"})
	if err != nil {
		t.Fatalf("cat error after stale cache entry: %v", err)
	}
	if strings.TrimSpace(catOut) != "public class LocalDate" {
		t.Fatalf("unexpected cat output after stale cache entry: %q", catOut)
	}
}

func TestSearchContextAndPassThrough(t *testing.T) {
	app := NewApp()
	if _, err := app.Runner.LookPath("rg"); err != nil {
		t.Skip("rg not available")
	}

	projectDir := filepath.Clean(filepath.Join("..", "..", "testdata", "fixture"))
	jarPath := filepath.Join(t.TempDir(), "kotlinx-datetime-sources.jar")
	inner := "kotlinx/datetime/LocalDate.kt"

	if err := writeTestJar(jarPath, inner, "before\npublic class LocalDate\nafter\n"); err != nil {
		t.Fatalf("write jar: %v", err)
	}

	setTestJarEnv(t, jarPath)

	ctxOut, err := runCommand(app, []string{"search", "public class LocalDate", "--module", "org.jetbrains.kotlinx:kotlinx-datetime", "--project", projectDir, "--context", "1"})
	if err != nil {
		t.Fatalf("search error: %v", err)
	}
	fileID := "org.jetbrains.kotlinx:kotlinx-datetime:0.7.1!/" + inner
	if !strings.Contains(ctxOut, fileID+" 1:0:before") || !strings.Contains(ctxOut, fileID+" 3:0:after") {
		t.Fatalf("context lines missing: %s", ctxOut)
	}

	filteredOut, err := runCommand(app, []string{"search", "public class LocalDate", "--module", "org.jetbrains.kotlinx:kotlinx-datetime", "--project", projectDir, "--", "-g", "!*.kt"})
	if err != nil {
		t.Fatalf("unexpected error for filtered search: %v", err)
	}
	if strings.TrimSpace(filteredOut) != "" {
		t.Fatalf("expected filtered search to return no matches, got: %s", filteredOut)
	}

	dashPatternOut, err := runCommand(app, []string{"search", "--module", "org.jetbrains.kotlinx:kotlinx-datetime", "--project", projectDir, "--", "-X"})
	if err != nil {
		t.Fatalf("unexpected error for dash pattern: %v", err)
	}
	if strings.TrimSpace(dashPatternOut) != "" {
		t.Fatalf("expected dash pattern search to return no matches, got: %s", dashPatternOut)
	}
}

func TestSearchDefaultsToAll(t *testing.T) {
	app := NewApp()
	if _, err := app.Runner.LookPath("rg"); err != nil {
		t.Skip("rg not available")
	}

	projectDir := filepath.Clean(filepath.Join("..", "..", "testdata", "fixture"))
	jarPath := filepath.Join(t.TempDir(), "kotlinx-datetime-sources.jar")
	inner := "kotlinx/datetime/LocalDate.kt"

	if err := writeTestJar(jarPath, inner, "public class LocalDate\n"); err != nil {
		t.Fatalf("write jar: %v", err)
	}

	setTestJarEnv(t, jarPath)

	out, err := runCommand(app, []string{"search", "public class LocalDate", "--project", projectDir})
	if err != nil {
		t.Fatalf("search error: %v", err)
	}
	if !strings.Contains(out, "org.jetbrains.kotlinx:kotlinx-datetime:0.7.1!/"+inner) {
		t.Fatalf("unexpected search output: %s", out)
	}
}

func TestSearchIgnoresFileIDTrackingFailures(t *testing.T) {
	app := NewApp()
	if _, err := app.Runner.LookPath("rg"); err != nil {
		t.Skip("rg not available")
	}

	projectDir := filepath.Clean(filepath.Join("..", "..", "testdata", "fixture"))
	jarPath := filepath.Join(t.TempDir(), "kotlinx-datetime-sources.jar")
	inner := "kotlinx/datetime/LocalDate.kt"
	blockedPath := filepath.Join(t.TempDir(), "not-a-dir")

	if err := writeTestJar(jarPath, inner, "public class LocalDate\n"); err != nil {
		t.Fatalf("write jar: %v", err)
	}
	if err := os.WriteFile(blockedPath, []byte("blocked"), 0o644); err != nil {
		t.Fatalf("write blocked path: %v", err)
	}

	setTestJarEnv(t, jarPath)
	t.Setenv("KSRC_FILEID_CACHE_DIR", blockedPath)

	out, err := runCommand(app, []string{"search", "public class LocalDate", "--module", "org.jetbrains.kotlinx:kotlinx-datetime", "--project", projectDir})
	if err != nil {
		t.Fatalf("search error: %v", err)
	}
	if !strings.Contains(out, "org.jetbrains.kotlinx:kotlinx-datetime:0.7.1!/"+inner) {
		t.Fatalf("unexpected search output: %s", out)
	}
}

func TestSearchReusesPersistentExtractCache(t *testing.T) {
	app := NewApp()
	if _, err := app.Runner.LookPath("rg"); err != nil {
		t.Skip("rg not available")
	}

	projectDir := filepath.Clean(filepath.Join("..", "..", "testdata", "fixture"))
	jarPath := filepath.Join(t.TempDir(), "kotlinx-datetime-sources.jar")
	inner := "kotlinx/datetime/LocalDate.kt"
	cacheDir := filepath.Join(t.TempDir(), "extract-cache")

	if err := writeTestJar(jarPath, inner, "public class LocalDate\n"); err != nil {
		t.Fatalf("write jar: %v", err)
	}

	setTestJarEnv(t, jarPath)
	t.Setenv("KSRC_EXTRACT_CACHE_DIR", cacheDir)

	first, err := runCommand(app, []string{"search", "public class LocalDate", "--module", "org.jetbrains.kotlinx:kotlinx-datetime", "--project", projectDir})
	if err != nil {
		t.Fatalf("first search error: %v", err)
	}
	before, err := cacheEntries(cacheDir)
	if err != nil {
		t.Fatalf("list cache after first run: %v", err)
	}
	if len(before) != 1 {
		t.Fatalf("expected one cache entry after first run, got %v", before)
	}

	second, err := runCommand(app, []string{"search", "public class LocalDate", "--module", "org.jetbrains.kotlinx:kotlinx-datetime", "--project", projectDir})
	if err != nil {
		t.Fatalf("second search error: %v", err)
	}
	after, err := cacheEntries(cacheDir)
	if err != nil {
		t.Fatalf("list cache after second run: %v", err)
	}
	if len(after) != 1 || before[0] != after[0] {
		t.Fatalf("expected cache reuse, before=%v after=%v", before, after)
	}
	if first != second {
		t.Fatalf("expected identical search output, first=%q second=%q", first, second)
	}
}

func TestSearchShowExtractedPathUsesQuotedTabSeparatedOutput(t *testing.T) {
	app := NewApp()
	if _, err := app.Runner.LookPath("rg"); err != nil {
		t.Skip("rg not available")
	}

	projectDir := filepath.Clean(filepath.Join("..", "..", "testdata", "fixture"))
	jarPath := filepath.Join(t.TempDir(), "kotlinx-datetime-sources.jar")
	inner := "kotlinx/datetime/LocalDate.kt"
	cacheDir := filepath.Join(t.TempDir(), "extract-cache")

	if err := writeTestJar(jarPath, inner, "public class LocalDate\n"); err != nil {
		t.Fatalf("write jar: %v", err)
	}

	setTestJarEnv(t, jarPath)
	t.Setenv("KSRC_EXTRACT_CACHE_DIR", cacheDir)

	out, err := runCommand(app, []string{"search", "public class LocalDate", "--module", "org.jetbrains.kotlinx:kotlinx-datetime", "--project", projectDir, "--show-extracted-path"})
	if err != nil {
		t.Fatalf("search error: %v", err)
	}

	absJarPath, err := filepath.Abs(jarPath)
	if err != nil {
		t.Fatalf("abs jar path: %v", err)
	}
	cacheKey := fmt.Sprintf("%x", sha256.Sum256([]byte(filepath.Clean(absJarPath))))
	extractedPath := filepath.Join(cacheDir, cacheKey, filepath.FromSlash(inner))
	want := fmt.Sprintf("org.jetbrains.kotlinx:kotlinx-datetime:0.7.1!/%s\t%q\t1\t1\t%q\n", inner, extractedPath, "public class LocalDate")
	if out != want {
		t.Fatalf("unexpected show-extracted-path output:\nwant: %q\n got: %q", want, out)
	}
}

func TestWherePathEmitsFullyQualifiedFileID(t *testing.T) {
	app := NewApp()
	projectDir := filepath.Clean(filepath.Join("..", "..", "testdata", "fixture"))
	jarPath := filepath.Join(t.TempDir(), "kotlinx-datetime-sources.jar")
	inner := "kotlinx/datetime/LocalDate.kt"

	if err := writeTestJar(jarPath, inner, "before\npublic class LocalDate\nafter\n"); err != nil {
		t.Fatalf("write jar: %v", err)
	}

	setTestJarEnv(t, jarPath)

	whereOut, err := runCommand(app, []string{"where", inner, "--group", "org.jetbrains.kotlinx", "--artifact", "kotlinx-datetime", "--project", projectDir})
	if err != nil {
		t.Fatalf("where error: %v", err)
	}

	want := "org.jetbrains.kotlinx:kotlinx-datetime:0.7.1!/" + inner + "|" + jarPath + "\n"
	if whereOut != want {
		t.Fatalf("unexpected where output:\nwant: %q\n got: %q", want, whereOut)
	}

	fileID, _, ok := strings.Cut(strings.TrimSpace(whereOut), "|")
	if !ok {
		t.Fatalf("missing file-id separator in output: %q", whereOut)
	}

	catOut, err := runCommand(app, []string{"cat", fileID, "--project", projectDir, "--lines", "2,2"})
	if err != nil {
		t.Fatalf("cat error: %v", err)
	}
	if strings.TrimSpace(catOut) != "public class LocalDate" {
		t.Fatalf("unexpected cat output: %q", catOut)
	}
}

func TestWherePathWithModuleWithoutVersionEmitsResolvedFileID(t *testing.T) {
	app := NewApp()
	projectDir := filepath.Clean(filepath.Join("..", "..", "testdata", "fixture"))
	jarPath := filepath.Join(t.TempDir(), "kotlinx-datetime-sources.jar")
	inner := "kotlinx/datetime/LocalDate.kt"

	if err := writeTestJar(jarPath, inner, "before\npublic class LocalDate\nafter\n"); err != nil {
		t.Fatalf("write jar: %v", err)
	}

	setTestJarEnv(t, jarPath)

	whereOut, err := runCommand(app, []string{"where", inner, "--module", "org.jetbrains.kotlinx:kotlinx-datetime", "--project", projectDir})
	if err != nil {
		t.Fatalf("where error: %v", err)
	}

	want := "org.jetbrains.kotlinx:kotlinx-datetime:0.7.1!/" + inner + "|" + jarPath + "\n"
	if whereOut != want {
		t.Fatalf("unexpected where output:\nwant: %q\n got: %q", want, whereOut)
	}
}

func TestWherePathIgnoresFileIDTrackingFailures(t *testing.T) {
	app := NewApp()
	projectDir := filepath.Clean(filepath.Join("..", "..", "testdata", "fixture"))
	jarPath := filepath.Join(t.TempDir(), "kotlinx-datetime-sources.jar")
	inner := "kotlinx/datetime/LocalDate.kt"
	blockedPath := filepath.Join(t.TempDir(), "not-a-dir")

	if err := writeTestJar(jarPath, inner, "before\npublic class LocalDate\nafter\n"); err != nil {
		t.Fatalf("write jar: %v", err)
	}
	if err := os.WriteFile(blockedPath, []byte("blocked"), 0o644); err != nil {
		t.Fatalf("write blocked path: %v", err)
	}

	setTestJarEnv(t, jarPath)
	t.Setenv("KSRC_FILEID_CACHE_DIR", blockedPath)

	whereOut, err := runCommand(app, []string{"where", inner, "--group", "org.jetbrains.kotlinx", "--artifact", "kotlinx-datetime", "--project", projectDir})
	if err != nil {
		t.Fatalf("where error: %v", err)
	}
	want := "org.jetbrains.kotlinx:kotlinx-datetime:0.7.1!/" + inner + "|" + jarPath + "\n"
	if whereOut != want {
		t.Fatalf("unexpected where output:\nwant: %q\n got: %q", want, whereOut)
	}
}

func TestWherePathUsesZipDirectoryEntriesBeforeCatReadsContent(t *testing.T) {
	app := NewApp()
	projectDir := filepath.Clean(filepath.Join("..", "..", "testdata", "fixture"))
	jarPath := filepath.Join(t.TempDir(), "kotlinx-datetime-sources.jar")
	inner := "kotlinx/datetime/LocalDate.kt"

	if err := writeTestJar(jarPath, inner, "before\npublic class LocalDate\nafter\n"); err != nil {
		t.Fatalf("write jar: %v", err)
	}
	if err := testutil.CorruptZipEntryMethod(jarPath, inner, 99); err != nil {
		t.Fatalf("corrupt zip entry method: %v", err)
	}

	setTestJarEnv(t, jarPath)

	whereOut, err := runCommand(app, []string{"where", inner, "--module", "org.jetbrains.kotlinx:kotlinx-datetime", "--project", projectDir})
	if err != nil {
		t.Fatalf("where error: %v", err)
	}
	want := "org.jetbrains.kotlinx:kotlinx-datetime:0.7.1!/" + inner + "|" + jarPath + "\n"
	if whereOut != want {
		t.Fatalf("unexpected where output:\nwant: %q\n got: %q", want, whereOut)
	}

	_, err = runCommand(app, []string{"cat", inner, "--module", "org.jetbrains.kotlinx:kotlinx-datetime", "--project", projectDir})
	if err == nil {
		t.Fatal("expected cat path mode to fail when entry body cannot be read")
	}
	if !strings.Contains(err.Error(), "unsupported compression algorithm") {
		t.Fatalf("expected body read failure, got: %v", err)
	}
}

func TestCommandsUseGradleUserHomeEnvForGradleAndCacheFallback(t *testing.T) {
	app := NewApp()
	if _, err := app.Runner.LookPath("rg"); err != nil {
		t.Skip("rg not available")
	}

	projectDir := t.TempDir()
	defaultHome := t.TempDir()
	gradleHome := filepath.Join(t.TempDir(), "custom-gradle")
	group := "org.jetbrains.kotlinx"
	artifact := "kotlinx-datetime"
	version := "0.7.1"
	coord := group + ":" + artifact + ":" + version
	inner := "kotlinx/datetime/LocalDate.kt"
	cacheDir := filepath.Join(gradleHome, "caches", "modules-2", "files-2.1", group, artifact, version, "hash")
	jarPath := filepath.Join(cacheDir, artifact+"-"+version+"-sources.jar")

	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		t.Fatalf("mkdir cache dir: %v", err)
	}
	if err := writeTestJar(jarPath, inner, "before\npublic class LocalDate\nafter\n"); err != nil {
		t.Fatalf("write jar: %v", err)
	}
	if err := writeGradleUserHomeAwareProject(projectDir, group, artifact, version, gradleHome); err != nil {
		t.Fatalf("write fake project: %v", err)
	}

	t.Setenv("HOME", defaultHome)
	t.Setenv("USERPROFILE", defaultHome)
	t.Setenv("GRADLE_USER_HOME", gradleHome)
	t.Setenv("KSRC_FILEID_CACHE_DIR", filepath.Join(t.TempDir(), "fileid-cache"))

	searchOut, err := runCommand(app, []string{"search", "public class LocalDate", "--module", coord, "--project", projectDir})
	if err != nil {
		t.Fatalf("search error: %v", err)
	}
	if !strings.Contains(searchOut, coord+"!/"+inner) {
		t.Fatalf("unexpected search output: %s", searchOut)
	}

	whereOut, err := runCommand(app, []string{"where", coord, "--project", projectDir})
	if err != nil {
		t.Fatalf("where error: %v", err)
	}
	wantWhere := coord + "|" + jarPath + "\n"
	if whereOut != wantWhere {
		t.Fatalf("unexpected where output:\nwant: %q\n got: %q", wantWhere, whereOut)
	}

	catOut, err := runCommand(NewApp(), []string{"cat", coord + "!/" + inner, "--lines", "2,2"})
	if err != nil {
		t.Fatalf("cat error: %v", err)
	}
	if strings.TrimSpace(catOut) != "public class LocalDate" {
		t.Fatalf("unexpected cat output: %q", catOut)
	}
}

func TestCommandsUseExplicitGradleUserHomeFlagForGradleAndCacheFallback(t *testing.T) {
	app := NewApp()
	if _, err := app.Runner.LookPath("rg"); err != nil {
		t.Skip("rg not available")
	}

	projectDir := t.TempDir()
	defaultHome := t.TempDir()
	envGradleHome := filepath.Join(t.TempDir(), "env-gradle")
	flagGradleHome := filepath.Join(t.TempDir(), "flag-gradle")
	group := "org.jetbrains.kotlinx"
	artifact := "kotlinx-datetime"
	version := "0.7.1"
	coord := group + ":" + artifact + ":" + version
	inner := "kotlinx/datetime/LocalDate.kt"
	cacheDir := filepath.Join(flagGradleHome, "caches", "modules-2", "files-2.1", group, artifact, version, "hash")
	jarPath := filepath.Join(cacheDir, artifact+"-"+version+"-sources.jar")

	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		t.Fatalf("mkdir cache dir: %v", err)
	}
	if err := writeTestJar(jarPath, inner, "before\npublic class LocalDate\nafter\n"); err != nil {
		t.Fatalf("write jar: %v", err)
	}
	if err := writeGradleUserHomeFlagAwareProject(projectDir, group, artifact, version, flagGradleHome); err != nil {
		t.Fatalf("write fake project: %v", err)
	}

	t.Setenv("HOME", defaultHome)
	t.Setenv("USERPROFILE", defaultHome)
	t.Setenv("GRADLE_USER_HOME", envGradleHome)
	t.Setenv("KSRC_FILEID_CACHE_DIR", filepath.Join(t.TempDir(), "fileid-cache"))

	searchOut, err := runCommand(app, []string{"search", "public class LocalDate", "--module", coord, "--project", projectDir, "--gradle-user-home", flagGradleHome})
	if err != nil {
		t.Fatalf("search error: %v", err)
	}
	if !strings.Contains(searchOut, coord+"!/"+inner) {
		t.Fatalf("unexpected search output: %s", searchOut)
	}

	whereOut, err := runCommand(app, []string{"where", coord, "--project", projectDir, "--gradle-user-home", flagGradleHome})
	if err != nil {
		t.Fatalf("where error: %v", err)
	}
	wantWhere := coord + "|" + jarPath + "\n"
	if whereOut != wantWhere {
		t.Fatalf("unexpected where output:\nwant: %q\n got: %q", wantWhere, whereOut)
	}

	catOut, err := runCommand(app, []string{"cat", coord + "!/" + inner, "--project", projectDir, "--gradle-user-home", flagGradleHome, "--lines", "2,2"})
	if err != nil {
		t.Fatalf("cat error: %v", err)
	}
	if strings.TrimSpace(catOut) != "public class LocalDate" {
		t.Fatalf("unexpected cat output: %q", catOut)
	}
}

func TestExplicitProjectOverridesTrackedFileIDAcrossDuplicateCoords(t *testing.T) {
	app := NewApp()
	projectDirA := t.TempDir()
	projectDirB := t.TempDir()
	inner := "kotlinx/datetime/LocalDate.kt"
	firstJar := filepath.Join(t.TempDir(), "first-kotlinx-datetime-sources.jar")
	secondJar := filepath.Join(t.TempDir(), "second-kotlinx-datetime-sources.jar")

	if err := writeTestJar(firstJar, inner, "first jar\n"); err != nil {
		t.Fatalf("write first jar: %v", err)
	}
	if err := writeTestJar(secondJar, inner, "second jar\n"); err != nil {
		t.Fatalf("write second jar: %v", err)
	}
	if err := writeFakeGradleProject(projectDirA); err != nil {
		t.Fatalf("write first fake project: %v", err)
	}
	if err := writeFakeGradleProject(projectDirB); err != nil {
		t.Fatalf("write second fake project: %v", err)
	}

	t.Setenv("KSRC_TEST_JAR", firstJar)
	t.Setenv("KSRC_FILEID_CACHE_DIR", filepath.Join(t.TempDir(), "fileid-cache"))

	whereOut, err := runCommand(app, []string{"where", inner, "--module", "org.jetbrains.kotlinx:kotlinx-datetime", "--project", projectDirA})
	if err != nil {
		t.Fatalf("where error: %v", err)
	}
	wantWhere := "org.jetbrains.kotlinx:kotlinx-datetime:0.7.1!/" + inner + "|" + firstJar + "\n"
	if whereOut != wantWhere {
		t.Fatalf("unexpected where output:\nwant: %q\n got: %q", wantWhere, whereOut)
	}

	fileID, _, ok := strings.Cut(strings.TrimSpace(whereOut), "|")
	if !ok {
		t.Fatalf("missing file-id separator in output: %q", whereOut)
	}

	t.Setenv("KSRC_TEST_JAR", secondJar)

	whereBOut, err := runCommand(app, []string{"where", fileID, "--project", projectDirB})
	if err != nil {
		t.Fatalf("where with explicit project error: %v", err)
	}
	wantWhereB := fileID + "|" + secondJar + "\n"
	if whereBOut != wantWhereB {
		t.Fatalf("unexpected where output with explicit project:\nwant: %q\n got: %q", wantWhereB, whereBOut)
	}

	catOut, err := runCommand(app, []string{"cat", fileID, "--project", projectDirB, "--lines", "1,1"})
	if err != nil {
		t.Fatalf("cat error: %v", err)
	}
	if strings.TrimSpace(catOut) != "second jar" {
		t.Fatalf("unexpected cat output: %q", catOut)
	}
}

func TestOpenFileIDHonorsExplicitProjectAndStaleTrackedLocation(t *testing.T) {
	app := NewApp()
	projectDirA := t.TempDir()
	projectDirB := t.TempDir()
	inner := "kotlinx/datetime/LocalDate.kt"
	firstJar := filepath.Join(t.TempDir(), "first-kotlinx-datetime-sources.jar")
	secondJar := filepath.Join(t.TempDir(), "second-kotlinx-datetime-sources.jar")

	if err := writeTestJar(firstJar, inner, "first jar\n"); err != nil {
		t.Fatalf("write first jar: %v", err)
	}
	if err := writeTestJar(secondJar, inner, "second jar\n"); err != nil {
		t.Fatalf("write second jar: %v", err)
	}
	if err := writeFakeGradleProject(projectDirA); err != nil {
		t.Fatalf("write first fake project: %v", err)
	}
	if err := writeFakeGradleProject(projectDirB); err != nil {
		t.Fatalf("write second fake project: %v", err)
	}

	t.Setenv("KSRC_TEST_JAR", firstJar)
	t.Setenv("KSRC_FILEID_CACHE_DIR", filepath.Join(t.TempDir(), "fileid-cache"))
	t.Setenv("PAGER", "cat")

	whereOut, err := runCommand(app, []string{"where", inner, "--module", "org.jetbrains.kotlinx:kotlinx-datetime", "--project", projectDirA})
	if err != nil {
		t.Fatalf("where error: %v", err)
	}
	fileID, _, ok := strings.Cut(strings.TrimSpace(whereOut), "|")
	if !ok {
		t.Fatalf("missing file-id separator in output: %q", whereOut)
	}

	if err := os.Remove(firstJar); err != nil {
		t.Fatalf("remove tracked jar: %v", err)
	}
	t.Setenv("KSRC_TEST_JAR", secondJar)

	openOut, err := runCommand(app, []string{"open", fileID, "--project", projectDirB, "--lines", "1,1"})
	if err != nil {
		t.Fatalf("open error: %v", err)
	}
	if strings.TrimSpace(openOut) != "second jar" {
		t.Fatalf("unexpected open output: %q", openOut)
	}
}

func writeTestJar(path, inner, content string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	zw := zip.NewWriter(f)
	w, err := zw.Create(inner)
	if err != nil {
		_ = zw.Close()
		_ = f.Close()
		return err
	}
	if _, err := w.Write([]byte(content)); err != nil {
		_ = zw.Close()
		_ = f.Close()
		return err
	}
	if err := zw.Close(); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}

func writeFakeGradleProject(projectDir string) error {
	return writeFakeGradleProjectWithCoord(projectDir, "org.jetbrains.kotlinx", "kotlinx-datetime", "0.7.1")
}

func writeFakeGradleProjectWithCoord(projectDir string, group string, artifact string, version string) error {
	if err := os.WriteFile(filepath.Join(projectDir, "settings.gradle.kts"), []byte("rootProject.name = \"fixture\"\n"), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(projectDir, "build.gradle.kts"), []byte("\n"), 0o644); err != nil {
		return err
	}
	wrapper := fmt.Sprintf(`#!/bin/sh
for arg in "$@"; do
  if [ "$arg" = %q ]; then
    printf '%%s\n' 'KSRCJSON	{"type":"dep","group":"%s","artifact":"%s","version":"%s"}'
    if [ -n "$KSRC_TEST_JAR" ]; then
      printf 'KSRCJSON\t{"type":"source","group":"%s","artifact":"%s","version":"%s","path":"%%s"}\n' "$KSRC_TEST_JAR"
    fi
    if [ -n "$KSRC_TEST_JAR_2" ]; then
      printf 'KSRCJSON\t{"type":"source","group":"%s","artifact":"%s","version":"%s","path":"%%s"}\n' "$KSRC_TEST_JAR_2"
    fi
  fi
done
exit 0
`, gradle.KsrcGradleTaskName(), group, artifact, version, group, artifact, version, group, artifact, version)
	path := filepath.Join(projectDir, "gradlew")
	if err := os.WriteFile(path, []byte(wrapper), 0o755); err != nil {
		return err
	}
	return os.Chmod(path, 0o755)
}

func writeGradleUserHomeAwareProject(projectDir string, group string, artifact string, version string, gradleUserHome string) error {
	if err := os.WriteFile(filepath.Join(projectDir, "settings.gradle.kts"), []byte("rootProject.name = \"fixture\"\n"), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(projectDir, "build.gradle.kts"), []byte("\n"), 0o644); err != nil {
		return err
	}
	wrapper := fmt.Sprintf(`#!/bin/sh
if [ "$GRADLE_USER_HOME" != %q ]; then
  echo "unexpected GRADLE_USER_HOME: $GRADLE_USER_HOME" >&2
  exit 41
fi
for arg in "$@"; do
  if [ "$arg" = %q ]; then
    printf '%%s\n' 'KSRCJSON	{"type":"dep","group":"%s","artifact":"%s","version":"%s"}'
  fi
done
exit 0
`, gradleUserHome, gradle.KsrcGradleTaskName(), group, artifact, version)
	path := filepath.Join(projectDir, "gradlew")
	if err := os.WriteFile(path, []byte(wrapper), 0o755); err != nil {
		return err
	}
	return os.Chmod(path, 0o755)
}

func writeGradleUserHomeFlagAwareProject(projectDir string, group string, artifact string, version string, gradleUserHome string) error {
	if err := os.WriteFile(filepath.Join(projectDir, "settings.gradle.kts"), []byte("rootProject.name = \"fixture\"\n"), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(projectDir, "build.gradle.kts"), []byte("\n"), 0o644); err != nil {
		return err
	}
	wrapper := fmt.Sprintf(`#!/bin/sh
seen_gradle_user_home=0
emit_sources=0
while [ "$#" -gt 0 ]; do
  case "$1" in
    --gradle-user-home)
      shift
      if [ "$#" -eq 0 ]; then
        echo "missing --gradle-user-home value" >&2
        exit 42
      fi
      if [ "$1" != %q ]; then
        echo "unexpected --gradle-user-home: $1" >&2
        exit 43
      fi
      seen_gradle_user_home=1
      ;;
    %s)
      emit_sources=1
      ;;
  esac
  shift
done
if [ "$seen_gradle_user_home" != "1" ]; then
  echo "missing --gradle-user-home" >&2
  exit 44
fi
if [ "$emit_sources" = "1" ]; then
  printf '%%s\n' 'KSRCJSON	{"type":"dep","group":"%s","artifact":"%s","version":"%s"}'
fi
exit 0
`, gradleUserHome, gradle.KsrcGradleTaskName(), group, artifact, version)
	path := filepath.Join(projectDir, "gradlew")
	if err := os.WriteFile(path, []byte(wrapper), 0o755); err != nil {
		return err
	}
	return os.Chmod(path, 0o755)
}

func setTestJarEnv(t *testing.T, jarPath string) {
	t.Helper()
	t.Setenv("KSRC_TEST_JAR", jarPath)
	t.Setenv("KSRC_FILEID_CACHE_DIR", filepath.Join(t.TempDir(), "fileid-cache"))
}

func cacheEntries(root string) ([]string, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			out = append(out, entry.Name())
		}
	}
	sort.Strings(out)
	return out, nil
}

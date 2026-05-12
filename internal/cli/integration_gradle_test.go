package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/respawn-app/ksrc/internal/executil"
	"github.com/respawn-app/ksrc/internal/gradle"
)

var (
	integrationCatalogDatetimePattern = regexp.MustCompile(`(?m)^kotlinx-datetime\s*=\s*"([^"]+)"\s*$`)
	integrationBuildDatetimePattern   = regexp.MustCompile(`org\.jetbrains\.kotlinx:kotlinx-datetime:([^'"]+)`)
)

func TestIntegrationWithRealGradle(t *testing.T) {
	if os.Getenv("KSRC_INTEGRATION") != "1" {
		t.Skip("set KSRC_INTEGRATION=1 to run")
	}

	app := NewApp()
	if _, err := app.Runner.LookPath("rg"); err != nil {
		t.Skip("rg not available")
	}

	projectDir := filepath.Clean(filepath.Join("..", "..", "testdata", "integration"))
	datetimeVersion := expectedKotlinxDatetimeVersion(t, projectDir)

	out, err := runCommand(app, []string{"search", "LocalDate", "--module", "org.jetbrains.kotlinx:kotlinx-datetime", "--project", projectDir})
	if err != nil {
		t.Fatalf("search error: %v", err)
	}
	if !strings.Contains(out, "org.jetbrains.kotlinx:kotlinx-datetime:"+datetimeVersion+"!/") {
		t.Fatalf("unexpected search output: %s", out)
	}

	fileID, line := firstSearchHit(t, out)
	lineRange := fmt.Sprintf("%d,%d", line, line)
	catOut, err := runCommand(app, []string{"cat", fileID, "--project", projectDir, "--lines", lineRange})
	if err != nil {
		t.Fatalf("cat error: %v", err)
	}
	if !strings.Contains(catOut, "LocalDate") {
		t.Fatalf("unexpected cat output: %s", catOut)
	}
}

func firstSearchHit(t *testing.T, out string) (string, int) {
	t.Helper()

	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, " ", 2)
		if len(parts) != 2 {
			t.Fatalf("unexpected search line: %q", line)
		}
		location := strings.SplitN(parts[1], ":", 3)
		if len(location) != 3 {
			t.Fatalf("unexpected search location: %q", parts[1])
		}
		lineNumber, err := strconv.Atoi(location[0])
		if err != nil {
			t.Fatalf("parse search line number %q: %v", location[0], err)
		}
		return parts[0], lineNumber
	}
	t.Fatalf("empty search output")
	return "", 0
}

func expectedKotlinxDatetimeVersion(t *testing.T, projectDir string) string {
	t.Helper()

	catalogPath := filepath.Join(projectDir, "gradle", "libs.versions.toml")
	if catalog, err := os.ReadFile(catalogPath); err == nil {
		if matches := integrationCatalogDatetimePattern.FindSubmatch(catalog); len(matches) > 0 {
			return string(matches[1])
		}
		t.Fatalf("%s: missing kotlinx-datetime version", catalogPath)
	}
	buildPath := filepath.Join(projectDir, "build.gradle")
	build, err := os.ReadFile(buildPath)
	if err != nil {
		t.Fatalf("read %s: %v", buildPath, err)
	}
	matches := integrationBuildDatetimePattern.FindSubmatch(build)
	if len(matches) == 0 {
		t.Fatalf("%s: missing kotlinx-datetime dependency", buildPath)
	}
	return string(matches[1])
}

func TestIntegrationWithSampleKmp(t *testing.T) {
	if os.Getenv("KSRC_INTEGRATION") != "1" {
		t.Skip("set KSRC_INTEGRATION=1 to run")
	}

	app := NewApp()
	if _, err := app.Runner.LookPath("rg"); err != nil {
		t.Skip("rg not available")
	}

	projectDir := prepareSampleProject(t)
	datetimeVersion := expectedKotlinxDatetimeVersion(t, projectDir)

	out, err := runCommand(app, []string{"search", "LocalDate", "--module", "org.jetbrains.kotlinx:kotlinx-datetime", "--project", projectDir})
	if err != nil {
		t.Fatalf("search error: %v", err)
	}
	if !strings.Contains(out, "org.jetbrains.kotlinx:kotlinx-datetime:"+datetimeVersion+"!/") {
		t.Fatalf("unexpected search output: %s", out)
	}
}

func TestIntegrationKmpBaseModuleIncludesExternalVariantSources(t *testing.T) {
	if os.Getenv("KSRC_INTEGRATION") != "1" {
		t.Skip("set KSRC_INTEGRATION=1 to run")
	}

	projectDir := prepareSampleProject(t)
	datetimeVersion := expectedKotlinxDatetimeVersion(t, projectDir)

	out, err := runCommand(NewApp(), []string{"resolve", "--module", "org.jetbrains.kotlinx:kotlinx-datetime", "--project", projectDir})
	if err != nil {
		t.Fatalf("resolve error: %v\n%s", err, out)
	}
	if !strings.Contains(out, "org.jetbrains.kotlinx:kotlinx-datetime:"+datetimeVersion+"|") {
		t.Fatalf("expected base/common sources in output:\n%s", out)
	}
	if !strings.Contains(out, "org.jetbrains.kotlinx:kotlinx-datetime-jvm:"+datetimeVersion+"|") {
		t.Fatalf("expected JVM external-variant sources in output:\n%s", out)
	}
	assertNoDuplicateResolvePaths(t, out)
}

func TestIntegrationInitScriptReusesConfigurationCacheWithIsolatedProjects(t *testing.T) {
	if os.Getenv("KSRC_INTEGRATION") != "1" {
		t.Skip("set KSRC_INTEGRATION=1 to run")
	}

	projectDir := prepareSampleProject(t)
	scriptPath := filepath.Join(t.TempDir(), "ksrc-init.gradle")
	if err := os.WriteFile(scriptPath, []byte(gradle.InitScript()), 0o644); err != nil {
		t.Fatalf("write init script: %v", err)
	}

	first := runGradleKsrcSources(t, projectDir, scriptPath)
	if strings.Contains(first, "Configuration cache problems found") {
		t.Fatalf("first run reported configuration cache problems:\n%s", first)
	}
	if strings.Contains(first, "resolved during configuration time") {
		t.Fatalf("first run resolved configurations during configuration time:\n%s", first)
	}
	if !strings.Contains(first, "Configuration cache entry stored") {
		t.Fatalf("first run did not store configuration cache:\n%s", first)
	}
	if !strings.Contains(first, "KSRCJSON\t") {
		t.Fatalf("first run did not emit ksrc records:\n%s", first)
	}

	second := runGradleKsrcSources(t, projectDir, scriptPath)
	if strings.Contains(second, "Configuration cache problems found") {
		t.Fatalf("second run reported configuration cache problems:\n%s", second)
	}
	if strings.Contains(second, "resolved during configuration time") {
		t.Fatalf("second run resolved configurations during configuration time:\n%s", second)
	}
	if !strings.Contains(second, "Reusing configuration cache") {
		t.Fatalf("second run did not reuse configuration cache:\n%s", second)
	}
	if !strings.Contains(second, "KSRCJSON\t") {
		t.Fatalf("second run did not replay ksrc records:\n%s", second)
	}
}

func TestIntegrationInitScriptEmitsIncludedBuildsWhenRootProjectUnselected(t *testing.T) {
	if os.Getenv("KSRC_INTEGRATION") != "1" {
		t.Skip("set KSRC_INTEGRATION=1 to run")
	}

	projectDir := prepareCompositeProject(t)
	scriptPath := filepath.Join(t.TempDir(), "ksrc-init.gradle")
	if err := os.WriteFile(scriptPath, []byte(gradle.InitScript()), 0o644); err != nil {
		t.Fatalf("write init script: %v", err)
	}

	first := runGradleKsrcSourcesWithProps(t, projectDir, scriptPath, "-PksrcSubprojects=app")
	if strings.Contains(first, "Configuration cache problems found") {
		t.Fatalf("first run reported configuration cache problems:\n%s", first)
	}
	assertIncludedBuildRecord(t, first, filepath.Join(projectDir, "included"))

	second := runGradleKsrcSourcesWithProps(t, projectDir, scriptPath, "-PksrcSubprojects=app")
	if strings.Contains(second, "Configuration cache problems found") {
		t.Fatalf("second run reported configuration cache problems:\n%s", second)
	}
	if !strings.Contains(second, "Reusing configuration cache") {
		t.Fatalf("second run did not reuse configuration cache:\n%s", second)
	}
	assertIncludedBuildRecord(t, second, filepath.Join(projectDir, "included"))
}

func TestIntegrationCLIResolveReusesConfigurationCacheWithIsolatedProjects(t *testing.T) {
	if os.Getenv("KSRC_INTEGRATION") != "1" {
		t.Skip("set KSRC_INTEGRATION=1 to run")
	}
	if _, err := exec.LookPath("rg"); err != nil {
		t.Skip("rg not available")
	}

	projectDir := prepareSampleProject(t)
	datetimeVersion := expectedKotlinxDatetimeVersion(t, projectDir)
	runner := &recordingRunner{inner: executil.OSRunner{}}
	app := &App{Runner: runner, Verbose: true}
	args := []string{"resolve", "--module", "org.jetbrains.kotlinx:kotlinx-datetime", "--project", projectDir}

	firstOut, err := runCommand(app, args)
	if err != nil {
		t.Fatalf("first resolve error: %v\n%s", err, firstOut)
	}
	if !strings.Contains(firstOut, "org.jetbrains.kotlinx:kotlinx-datetime:"+datetimeVersion+"|") {
		t.Fatalf("unexpected first resolve output:\n%s", firstOut)
	}

	secondOut, err := runCommand(app, args)
	if err != nil {
		t.Fatalf("second resolve error: %v\n%s", err, secondOut)
	}
	if !strings.Contains(secondOut, "org.jetbrains.kotlinx:kotlinx-datetime:"+datetimeVersion+"|") {
		t.Fatalf("unexpected second resolve output:\n%s", secondOut)
	}
	if len(runner.calls) < 2 {
		t.Fatalf("expected at least 2 Gradle calls, got %d", len(runner.calls))
	}
	secondGradleOutput := runner.calls[len(runner.calls)-1].stdout + "\n" + runner.calls[len(runner.calls)-1].stderr
	if strings.Contains(secondGradleOutput, "Configuration cache problems found") {
		t.Fatalf("second resolve reported configuration cache problems:\n%s", secondGradleOutput)
	}
	if strings.Contains(secondGradleOutput, "resolved during configuration time") {
		t.Fatalf("second resolve resolved configurations during configuration time:\n%s", secondGradleOutput)
	}
	if !strings.Contains(secondGradleOutput, "Reusing configuration cache") {
		t.Fatalf("second resolve did not reuse configuration cache:\n%s", secondGradleOutput)
	}
}

func runGradleKsrcSources(t *testing.T, projectDir string, scriptPath string) string {
	t.Helper()

	return runGradleKsrcSourcesWithProps(t, projectDir, scriptPath)
}

func runGradleKsrcSourcesWithProps(t *testing.T, projectDir string, scriptPath string, props ...string) string {
	t.Helper()

	args := []string{
		"-I", scriptPath,
		"-PksrcModule=org.jetbrains.kotlinx:kotlinx-datetime",
		"-PksrcScope=compile",
		"-PksrcBuildscript=true",
		"-PksrcIncludeBuilds=true",
	}
	args = append(args, props...)
	args = append(args,
		gradle.KsrcGradleTaskName(),
		"--configuration-cache",
		"--info",
	)
	stdout, stderr, err := executil.OSRunner{}.Run(context.Background(), projectDir, filepath.Join(projectDir, "gradlew"), args...)
	if err != nil {
		t.Fatalf("gradle %s failed: %v\n%s\n%s", gradle.KsrcGradleTaskName(), err, stdout, stderr)
	}
	return stdout + "\n" + stderr
}

func assertIncludedBuildRecord(t *testing.T, out string, includedDir string) {
	t.Helper()

	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "KSRCJSON\t") {
			continue
		}
		var record struct {
			Type string `json:"type"`
			Path string `json:"path"`
		}
		if err := json.Unmarshal([]byte(strings.TrimPrefix(line, "KSRCJSON\t")), &record); err != nil {
			t.Fatalf("parse ksrc record %q: %v", line, err)
		}
		if record.Type == "include" && sameIntegrationPath(record.Path, includedDir) {
			return
		}
	}
	t.Fatalf("expected include record for %q in output:\n%s", includedDir, out)
}

func sameIntegrationPath(a string, b string) bool {
	return canonicalIntegrationPath(a) == canonicalIntegrationPath(b)
}

func canonicalIntegrationPath(path string) string {
	if abs, err := filepath.Abs(path); err == nil {
		path = abs
	}
	if realPath, err := filepath.EvalSymlinks(path); err == nil {
		path = realPath
	}
	return filepath.Clean(path)
}

func assertNoDuplicateResolvePaths(t *testing.T, out string) {
	t.Helper()

	seen := map[string]struct{}{}
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "WARN:") || strings.HasPrefix(line, "VERBOSE:") {
			continue
		}
		parts := strings.SplitN(line, "|", 2)
		if len(parts) != 2 {
			continue
		}
		path := strings.TrimSpace(parts[1])
		if path == "" {
			continue
		}
		if _, exists := seen[path]; exists {
			t.Fatalf("duplicate source jar path %q in output:\n%s", path, out)
		}
		seen[path] = struct{}{}
	}
}

type recordingRunner struct {
	inner executil.Runner
	calls []recordedRun
}

type recordedRun struct {
	stdout string
	stderr string
}

func (r *recordingRunner) Run(ctx context.Context, dir string, name string, args ...string) (string, string, error) {
	stdout, stderr, err := r.inner.Run(ctx, dir, name, args...)
	r.calls = append(r.calls, recordedRun{stdout: stdout, stderr: stderr})
	return stdout, stderr, err
}

func (r *recordingRunner) LookPath(file string) (string, error) {
	return r.inner.LookPath(file)
}

func prepareSampleProject(t *testing.T) string {
	t.Helper()

	src := filepath.Clean(filepath.Join("..", "..", "sample"))
	dst := t.TempDir()
	if err := copyDir(src, dst, map[string]struct{}{".gradle": {}, "build": {}}); err != nil {
		t.Fatalf("copy sample: %v", err)
	}
	if err := updateSampleVersions(dst); err != nil {
		t.Fatalf("update sample versions: %v", err)
	}
	if ver := strings.TrimSpace(os.Getenv("KSRC_SAMPLE_GRADLE_VERSION")); ver != "" {
		if err := runGradleWrapper(dst, ver); err != nil {
			t.Fatalf("update wrapper: %v", err)
		}
	}
	return dst
}

func prepareCompositeProject(t *testing.T) string {
	t.Helper()

	dst := t.TempDir()
	copyGradleWrapper(t, dst)
	writeTextFile(t, filepath.Join(dst, "gradle.properties"), strings.Join([]string{
		"org.gradle.configuration-cache=true",
		"org.gradle.unsafe.isolated-projects=true",
		"",
	}, "\n"))
	writeTextFile(t, filepath.Join(dst, "settings.gradle"), strings.Join([]string{
		"rootProject.name = 'ksrc-composite-root'",
		"include ':app'",
		"includeBuild 'included'",
		"",
	}, "\n"))
	writeTextFile(t, filepath.Join(dst, "build.gradle"), "tasks.register('ksrcSources')\n")
	writeTextFile(t, filepath.Join(dst, "app", "build.gradle"), strings.Join([]string{
		"plugins {",
		"    id 'java-library'",
		"}",
		"",
		"tasks.register('ksrcSources')",
		"",
	}, "\n"))
	writeTextFile(t, filepath.Join(dst, "included", "settings.gradle"), "rootProject.name = 'ksrc-included'\n")
	writeTextFile(t, filepath.Join(dst, "included", "build.gradle"), strings.Join([]string{
		"plugins {",
		"    id 'java-library'",
		"}",
		"",
	}, "\n"))
	return dst
}

func copyGradleWrapper(t *testing.T, dst string) {
	t.Helper()

	src := filepath.Clean(filepath.Join("..", "..", "sample"))
	for _, rel := range []string{
		"gradlew",
		filepath.Join("gradle", "wrapper", "gradle-wrapper.jar"),
		filepath.Join("gradle", "wrapper", "gradle-wrapper.properties"),
	} {
		if err := copyFile(filepath.Join(src, rel), filepath.Join(dst, rel)); err != nil {
			t.Fatalf("copy wrapper file %s: %v", rel, err)
		}
	}
}

func writeTextFile(t *testing.T, path string, contents string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func updateSampleVersions(projectDir string) error {
	catalogPath := filepath.Join(projectDir, "gradle", "libs.versions.toml")
	if ver := strings.TrimSpace(os.Getenv("KSRC_SAMPLE_AGP_VERSION")); ver != "" {
		if err := updateCatalogVersion(catalogPath, "agp", ver); err != nil {
			return err
		}
	}
	if ver := strings.TrimSpace(os.Getenv("KSRC_SAMPLE_KOTLIN_VERSION")); ver != "" {
		if err := updateCatalogVersion(catalogPath, "kotlin", ver); err != nil {
			return err
		}
	}
	return nil
}

func updateCatalogVersion(path string, key string, version string) error {
	input, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	lines := strings.Split(string(input), "\n")
	updated := false
	prefix := key + " = "
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, prefix) {
			lines[i] = fmt.Sprintf("%s = %q", key, version)
			updated = true
		}
	}
	if !updated {
		return fmt.Errorf("version key %q not found in %s", key, path)
	}
	return os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0o644)
}

func runGradleWrapper(projectDir string, version string) error {
	runner := executil.OSRunner{}
	if _, err := runner.LookPath("gradle"); err != nil {
		return fmt.Errorf("gradle not found on PATH: %w", err)
	}
	_, _, err := runner.Run(context.Background(), projectDir, "gradle", "wrapper", "--gradle-version", version)
	return err
}

func copyDir(src string, dst string, skip map[string]struct{}) error {
	return filepath.WalkDir(src, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if _, ok := skip[d.Name()]; ok {
				return filepath.SkipDir
			}
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		return copyFile(path, target)
	})
}

func copyFile(src string, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	info, err := in.Stat()
	if err != nil {
		return err
	}
	return os.Chmod(dst, info.Mode())
}

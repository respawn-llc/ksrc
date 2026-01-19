package cli

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
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

	out, err := runCommand(app, []string{"search", "LocalDate", "--module", "org.jetbrains.kotlinx:kotlinx-datetime", "--project", projectDir})
	if err != nil {
		t.Fatalf("search error: %v", err)
	}
	if !strings.Contains(out, "org.jetbrains.kotlinx:kotlinx-datetime:0.6.1!/") {
		t.Fatalf("unexpected search output: %s", out)
	}

	fileID := strings.Fields(out)[0]
	catOut, err := runCommand(app, []string{"cat", fileID, "--project", projectDir, "--lines", "1,5"})
	if err != nil {
		t.Fatalf("cat error: %v", err)
	}
	if !strings.Contains(catOut, "LocalDate") {
		t.Fatalf("unexpected cat output: %s", catOut)
	}
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

	out, err := runCommand(app, []string{"search", "LocalDate", "--module", "org.jetbrains.kotlinx:kotlinx-datetime", "--project", projectDir})
	if err != nil {
		t.Fatalf("search error: %v", err)
	}
	if !strings.Contains(out, "org.jetbrains.kotlinx:kotlinx-datetime:0.6.1!/") {
		t.Fatalf("unexpected search output: %s", out)
	}
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
	if _, err := exec.LookPath("gradle"); err != nil {
		return fmt.Errorf("gradle not found on PATH: %w", err)
	}
	cmd := exec.Command("gradle", "wrapper", "--gradle-version", version)
	cmd.Dir = projectDir
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	return cmd.Run()
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

package cli

import (
	"archive/zip"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
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

	t.Setenv("KSRC_TEST_JAR", jarPath)

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

	t.Setenv("KSRC_TEST_JAR", jarPath)

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

	t.Setenv("KSRC_TEST_JAR", jarPath)

	out, err := runCommand(app, []string{"search", "public class LocalDate", "--project", projectDir})
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

	t.Setenv("KSRC_TEST_JAR", jarPath)
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

func TestWherePathEmitsFullyQualifiedFileID(t *testing.T) {
	app := NewApp()
	projectDir := filepath.Clean(filepath.Join("..", "..", "testdata", "fixture"))
	jarPath := filepath.Join(t.TempDir(), "kotlinx-datetime-sources.jar")
	inner := "kotlinx/datetime/LocalDate.kt"

	if err := writeTestJar(jarPath, inner, "before\npublic class LocalDate\nafter\n"); err != nil {
		t.Fatalf("write jar: %v", err)
	}

	t.Setenv("KSRC_TEST_JAR", jarPath)

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

	t.Setenv("KSRC_TEST_JAR", jarPath)

	whereOut, err := runCommand(app, []string{"where", inner, "--module", "org.jetbrains.kotlinx:kotlinx-datetime", "--project", projectDir})
	if err != nil {
		t.Fatalf("where error: %v", err)
	}

	want := "org.jetbrains.kotlinx:kotlinx-datetime:0.7.1!/" + inner + "|" + jarPath + "\n"
	if whereOut != want {
		t.Fatalf("unexpected where output:\nwant: %q\n got: %q", want, whereOut)
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

package windowssmoke

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
	"time"
	"unicode/utf16"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// This opt-in black-box suite runs the installed binary, not CLI test doubles.
func TestWindowsSmoke(t *testing.T) {
	if runtime.GOOS != "windows" || os.Getenv("KSRC_WINDOWS_SMOKE") != "1" {
		t.Skip("run scripts/windows-smoke.ps1 on native Windows")
	}
	repo, err := filepath.Abs(filepath.Join("..", ".."))
	must(t, err)
	root := filepath.Join(t.TempDir(), "windows smoke")
	must(t, os.MkdirAll(root, 0o755))
	t.Setenv("KSRC_EXTRACT_CACHE_DIR", filepath.Join(root, "extracted sources"))
	t.Setenv("KSRC_FILEID_CACHE_DIR", filepath.Join(root, "file ids"))
	t.Setenv("GRADLE_USER_HOME", filepath.Join(root, "unused environment home"))
	// A persistent daemon holds its home open and prevents Windows temp cleanup.
	t.Setenv("GRADLE_OPTS", "-Dorg.gradle.daemon=false")
	t.Setenv("PAGER", "cat")

	versionBytes, err := os.ReadFile(filepath.Join(repo, "VERSION"))
	must(t, err)
	version := strings.TrimPrefix(strings.TrimSpace(string(versionBytes)), "v")
	binaryName := fmt.Sprintf("ksrc_%s_windows_%s.exe", version, runtime.GOARCH)
	binary := filepath.Join(root, binaryName)
	run(t, repo, "go", "build", "-trimpath", "-buildvcs=false",
		"-ldflags", "-X github.com/respawn-app/ksrc/internal/cli.Version="+version,
		"-o", binary, "./cmd/ksrc")
	archive := makeZip(t, binaryName, readFile(t, binary))
	archivePath := fmt.Sprintf("/v%s/ksrc_%s_windows_%s.zip", version, version, runtime.GOARCH)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != archivePath {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/zip")
		_, _ = w.Write(archive)
	}))
	defer server.Close()
	t.Setenv("KSRC_RELEASE_BASE", server.URL)
	prefix := filepath.Join(root, "installed tools")
	bash := os.Getenv("KSRC_WINDOWS_GIT_BASH")
	if bash == "" {
		t.Fatal("KSRC_WINDOWS_GIT_BASH must name the native Git Bash executable")
	}
	// Positional arguments preserve spaces without embedding paths in shell code.
	run(t, repo, bash, "-c",
		`exec sh scripts/install.sh --version "$1" --prefix "$(cygpath -u "$2")"`,
		"windows-smoke", version, prefix)
	installed := filepath.Join(prefix, "bin", "ksrc.exe")
	if info, err := os.Stat(installed); err != nil || info.IsDir() {
		t.Fatalf("installer must produce runnable ksrc.exe: %v", err)
	}
	out, _ := run(t, root, installed, "--version")
	if strings.TrimSpace(out) != version {
		t.Fatalf("installed version = %q, want %q", out, version)
	}
	out, _ = run(t, root, installed, "--help")
	requireContains(t, out, "Gradle dependency source search")

	project := filepath.Join(root, "real Gradle project")
	copyTree(t, filepath.Join(repo, "testdata", "integration"), project)
	home := filepath.Join(root, "Gradle user home")
	common := []string{"--project", project, "--gradle-user-home", home}
	module := "org.jetbrains.kotlinx:kotlinx-datetime"
	versionMatch := regexp.MustCompile(`org\.jetbrains\.kotlinx:kotlinx-datetime:([^'"]+)`).
		FindSubmatch(readFile(t, filepath.Join(project, "build.gradle")))
	if len(versionMatch) != 2 {
		t.Fatal("integration fixture is missing its kotlinx-datetime dependency")
	}
	coordinate := module + ":" + string(versionMatch[1])
	healthy := func(t *testing.T, args ...string) string {
		t.Helper()
		out, diagnostics := run(t, root, installed, append(args, common...)...)
		if strings.Contains(diagnostics, "WARN:") || strings.Contains(diagnostics, "cache-only") {
			t.Fatalf("Gradle success must not be masked by cache fallback:\n%s", diagnostics)
		}
		return out
	}

	t.Run("real Gradle and explicit home", func(t *testing.T) {
		// Fetch runs against the cold home, before doctor can download the sources.
		out := healthy(t, "fetch", coordinate)
		requireContains(t, out, coordinate+"|")
		requireContains(t, out, home)
		out = healthy(t, "doctor")
		requireContains(t, out, "rg: ok")
		requireContains(t, out, "gradle resolve: ok")
		requireContains(t, out, "gradlew.bat")
		out = healthy(t, "deps")
		requireContains(t, out, coordinate)
		requireContains(t, out, "[sources: yes]")
		out = healthy(t, "resolve", "--module", module)
		requireContains(t, out, coordinate+"|")
		requireContains(t, out, home)
		if _, err := os.Stat(filepath.Join(home, "caches", "modules-2", "files-2.1")); err != nil {
			t.Fatalf("explicit Gradle user home was not populated: %v", err)
		}
		if _, err := os.Stat(filepath.Join(os.Getenv("GRADLE_USER_HOME"), "caches")); !os.IsNotExist(err) {
			t.Fatalf("explicit home must override GRADLE_USER_HOME, stat error: %v", err)
		}
	})

	t.Run("search and file-id followups", func(t *testing.T) {
		out := healthy(t, "search", "LocalDate", "--module", module)
		fileID, line, text := searchHit(t, out, coordinate)
		lineRange := line + "," + line
		content := healthy(t, "cat", fileID, "--lines", lineRange)
		if strings.TrimRight(content, "\r\n") != text {
			t.Fatalf("cat returned %q, want search line %q", content, text)
		}
		opened := healthy(t, "open", fileID, "--lines", lineRange)
		if opened != content {
			t.Fatalf("open via PAGER=cat returned %q, want %q", opened, content)
		}
		t.Run("native default pager", func(t *testing.T) {
			t.Setenv("PAGER", "")
			requireContains(t, healthy(t, "open", fileID, "--lines", lineRange), text)
		})
		located := healthy(t, "where", fileID)
		requireContains(t, located, fileID+"|")
		requireContains(t, located, home)
		_, innerPath, _ := strings.Cut(fileID, "!/")
		byPath := healthy(t, "cat", innerPath, "--module", coordinate, "--lines", lineRange)
		if byPath != content {
			t.Fatalf("module/path cat returned %q, want %q", byPath, content)
		}
		if byPath := healthy(t, "where", innerPath, "--module", coordinate); byPath != located {
			t.Fatalf("module/path where returned %q, want %q", byPath, located)
		}
		// Exercise persisted file-id chaining without explicit project context too.
		chained, diagnostics := run(t, root, installed, "cat", fileID, "--lines", lineRange)
		if chained != content || strings.Contains(diagnostics, "WARN:") {
			t.Fatalf("context-free file-id chaining failed:\n%s\n%s", chained, diagnostics)
		}
	})

	t.Run("installed stdio MCP search cat and deps", func(t *testing.T) {
		// MCP must establish its own file-id mapping rather than reuse CLI state.
		t.Setenv("KSRC_FILEID_CACHE_DIR", filepath.Join(root, "MCP file ids"))
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		stderrPath := filepath.Join(t.TempDir(), "mcp-stderr.log")
		stderr, err := os.Create(stderrPath)
		must(t, err)
		// A file captures diagnostics without racing os/exec's stderr copier.
		// Read after session.Close so warnings cannot arrive after the assertion.
		defer func() {
			must(t, stderr.Close())
			diagnostics := string(readFile(t, stderrPath))
			t.Logf("installed MCP stderr:\n%s", diagnostics)
			if strings.Contains(diagnostics, "WARN:") || strings.Contains(diagnostics, "cache-only") {
				t.Errorf("MCP Gradle success must not be masked by cache fallback:\n%s", diagnostics)
			}
		}()
		cmd := exec.CommandContext(ctx, installed, "mcp", "-v")
		cmd.Dir, cmd.Stderr = root, stderr
		client := mcp.NewClient(&mcp.Implementation{Name: "windows-smoke", Version: "1.0"}, nil)
		session, err := client.Connect(ctx, &mcp.CommandTransport{Command: cmd}, nil)
		must(t, err)
		defer func() {
			if err := session.Close(); err != nil {
				t.Errorf("close installed MCP session: %v", err)
			}
		}()
		call := func(name string, arguments map[string]any) string {
			t.Helper()
			result, err := session.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: arguments})
			if err != nil {
				t.Fatalf("MCP %s failed: %v", name, err)
			}
			var output strings.Builder
			for _, content := range result.Content {
				text, ok := content.(*mcp.TextContent)
				if !ok {
					t.Fatalf("MCP %s returned non-plaintext content: %T", name, content)
				}
				output.WriteString(text.Text)
			}
			t.Logf("MCP %s output:\n%s", name, output.String())
			if result.IsError {
				t.Fatalf("MCP %s returned a tool error: %s", name, output.String())
			}
			return output.String()
		}
		out := call("search", map[string]any{
			"query": "LocalDate", "group": "org.jetbrains.kotlinx", "artifact": "kotlinx-datetime",
			"version": string(versionMatch[1]), "project": project, "gradleUserHome": home,
		})
		fileID, line, text := searchHit(t, out, coordinate)
		content := call("cat", map[string]any{"fileId": fileID, "lines": line + "," + line})
		if strings.TrimRight(content, "\r\n") != text {
			t.Fatalf("MCP cat returned %q, want search line %q", content, text)
		}
		out = call("deps", map[string]any{
			"group": "org.jetbrains.kotlinx", "artifact": "kotlinx-datetime",
			"project": project, "gradleUserHome": home,
		})
		requireContains(t, out, coordinate)
		requireContains(t, out, "[sources: yes]")
		requireContains(t, out, home)
	})

	t.Run("no matches", func(t *testing.T) {
		out := healthy(t, "search", "ksrc_windows_no_match_928d2d", "--module", module)
		if strings.TrimSpace(out) != "" {
			t.Fatalf("no-match search must return empty stdout, got %q", out)
		}
	})

	failedProject := filepath.Join(root, "broken Gradle project")
	must(t, os.MkdirAll(failedProject, 0o755))
	must(t, os.WriteFile(filepath.Join(failedProject, "settings.gradle"), nil, 0o644))
	must(t, os.WriteFile(filepath.Join(failedProject, "build.gradle"), nil, 0o644))
	must(t, os.WriteFile(filepath.Join(failedProject, "gradlew.bat"),
		[]byte("@echo off\r\necho intentional Windows smoke failure 1>&2\r\nexit /b 9\r\n"), 0o644))
	t.Run("cache-only fallback uses explicit home", func(t *testing.T) {
		out, diagnostics := run(t, root, installed, "search", "LocalDate",
			"--module", module, "--project", failedProject, "--gradle-user-home", home, "-v")
		requireContains(t, out, coordinate+"!/")
		requireContains(t, diagnostics, "Falling back to cache-only resolution")
		requireContains(t, diagnostics, "intentional Windows smoke failure")
	})

	t.Run("large dependency search", func(t *testing.T) {
		const count = 450
		largeHome := filepath.Join(root, "large dependency home")
		extracted := filepath.Join(root, "large extracted sources")
		t.Setenv("KSRC_EXTRACT_CACHE_DIR", extracted)
		for i := 0; i < count; i++ {
			artifact := fmt.Sprintf("library-%03d", i)
			jar := filepath.Join(largeHome, "caches", "modules-2", "files-2.1",
				"windows.smoke", artifact, "1.0", "hash", artifact+"-1.0-sources.jar")
			must(t, os.MkdirAll(filepath.Dir(jar), 0o755))
			must(t, os.WriteFile(jar, makeZip(t, "Smoke.kt",
				[]byte(fmt.Sprintf("val WindowsSmokeLargeDependency = %d\n", i))), 0o644))
		}
		out, diagnostics := run(t, root, installed, "search", "WindowsSmokeLargeDependency",
			"--all", "--project", failedProject, "--gradle-user-home", largeHome)
		requireContains(t, diagnostics, "Falling back to cache-only resolution")
		lines := strings.Split(strings.TrimSpace(out), "\n")
		if len(lines) != count {
			t.Fatalf("large search returned %d hits, want %d", len(lines), count)
		}
		seen := make(map[string]bool, count)
		for _, line := range lines {
			fields := strings.SplitN(strings.TrimSpace(line), " ", 2)
			if len(fields) != 2 {
				t.Fatalf("unparseable large-search hit: %q", line)
			}
			seen[fields[0]] = true
		}
		for i := 0; i < count; i++ {
			id := "windows.smoke:library-" + fmt.Sprintf("%03d", i) + ":1.0!/Smoke.kt"
			if !seen[id] {
				t.Fatalf("large search lost dependency %s", id)
			}
		}
		// Measure actual extraction directories, so this case cannot silently shrink
		// below Windows' CreateProcess limit when cache layout or temp paths change.
		entries, err := os.ReadDir(extracted)
		must(t, err)
		rawUnits, dirs := 0, 0
		for _, entry := range entries {
			if entry.IsDir() {
				rawUnits += len(utf16.Encode([]rune(filepath.Join(extracted, entry.Name())))) + 3
				dirs++
			}
		}
		if dirs != count || rawUnits <= 32767 {
			t.Fatalf("large-search coverage requires %d roots and >32767 UTF-16 units; got %d roots, %d units",
				count, dirs, rawUnits)
		}
		t.Logf("searched all %d dependencies; unchunked directory arguments would use %d UTF-16 units",
			count, rawUnits)
	})
}

func searchHit(t *testing.T, output, coordinate string) (fileID, line, text string) {
	t.Helper()
	hit := regexp.MustCompile(`(?m)^(` + regexp.QuoteMeta(coordinate) + `!/\S+) ([0-9]+):[0-9]+:([^\r\n]*)`).
		FindStringSubmatch(output)
	if len(hit) != 4 {
		t.Fatalf("search did not return a parseable file-id hit:\n%s", output)
	}
	return hit[1], hit[2], hit[3]
}

func run(t *testing.T, dir, executable string, args ...string) (string, string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, executable, args...)
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	t.Logf("%s %q\nstdout:\n%s\nstderr:\n%s", executable, args, stdout.String(), stderr.String())
	if err != nil {
		t.Fatalf("command failed: %v (context: %v)", err, ctx.Err())
	}
	return stdout.String(), stderr.String()
}

func makeZip(t *testing.T, name string, data []byte) []byte {
	t.Helper()
	var buffer bytes.Buffer
	w := zip.NewWriter(&buffer)
	entry, err := w.Create(name)
	must(t, err)
	_, err = entry.Write(data)
	must(t, err)
	must(t, w.Close())
	return buffer.Bytes()
}

func readFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	must(t, err)
	return data
}

func copyTree(t *testing.T, src, dst string) {
	t.Helper()
	must(t, filepath.WalkDir(src, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		// Local Gradle state is not fixture input, and must not warm the smoke home.
		if entry.IsDir() {
			if entry.Name() == ".gradle" || entry.Name() == "build" {
				return filepath.SkipDir
			}
			return os.MkdirAll(filepath.Join(dst, rel), 0o755)
		}
		input, err := os.Open(path)
		if err != nil {
			return err
		}
		defer input.Close()
		output, err := os.Create(filepath.Join(dst, rel))
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(output, input)
		closeErr := output.Close()
		if copyErr != nil {
			return copyErr
		}
		return closeErr
	}))
}

func requireContains(t *testing.T, text, want string) {
	t.Helper()
	if !strings.Contains(text, want) {
		t.Fatalf("expected %q in:\n%s", want, text)
	}
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

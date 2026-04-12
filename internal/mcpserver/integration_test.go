package mcpserver

import (
	"archive/zip"
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/respawn-app/ksrc/internal/cat"
	"github.com/respawn-app/ksrc/internal/testutil"
)

type toolSession interface {
	CallTool(context.Context, *mcp.CallToolParams) (*mcp.CallToolResult, error)
	Close() error
}

func TestMCPServerSearchAndCatIntegration(t *testing.T) {
	if _, err := exec.LookPath("rg"); err != nil {
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

	if err := writeZipFile(jarPath, inner, "before\npublic class LocalDate\nafter\n"); err != nil {
		t.Fatalf("write jar: %v", err)
	}
	if _, err := cat.ReadFileFromZip(jarPath, inner, nil); err != nil {
		t.Fatalf("jar missing test file: %v", err)
	}

	ctx, session, cleanup := startTestSession(t, root, projectDir, jarPath)
	defer cleanup()

	searchRes, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "search",
		Arguments: map[string]any{
			"query":       "public class LocalDate",
			"group":       "org.jetbrains.kotlinx",
			"artifact":    "kotlinx-datetime",
			"project":     projectDir,
			"subprojects": []string{},
		},
	})
	if err != nil {
		t.Fatalf("search tool: %v", err)
	}
	searchText := textFromResult(searchRes)
	expectedFileID := "org.jetbrains.kotlinx:kotlinx-datetime:0.7.1!/" + inner
	if !strings.Contains(searchText, expectedFileID) {
		t.Fatalf("unexpected search output: %s", searchText)
	}

	catRes, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "cat",
		Arguments: map[string]any{
			"fileId": expectedFileID,
			"lines":  "2,2",
		},
	})
	if err != nil {
		t.Fatalf("cat tool: %v", err)
	}
	catText := strings.TrimSpace(textFromResult(catRes))
	if catText != "public class LocalDate" {
		t.Fatalf("unexpected cat output: %q", catText)
	}
}

func TestMCPServerSearchAndCatIntegrationReusesTrackedFileIDAcrossCWD(t *testing.T) {
	if _, err := exec.LookPath("rg"); err != nil {
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

	if err := writeZipFile(jarPath, inner, "before\npublic class LocalDate\nafter\n"); err != nil {
		t.Fatalf("write jar: %v", err)
	}
	if _, err := cat.ReadFileFromZip(jarPath, inner, nil); err != nil {
		t.Fatalf("jar missing test file: %v", err)
	}

	ctx, session, cleanup := startTestSessionAt(t, root, t.TempDir(), jarPath)
	defer cleanup()

	searchRes, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "search",
		Arguments: map[string]any{
			"query":       "public class LocalDate",
			"group":       "org.jetbrains.kotlinx",
			"artifact":    "kotlinx-datetime",
			"project":     projectDir,
			"subprojects": []string{},
		},
	})
	if err != nil {
		t.Fatalf("search tool: %v", err)
	}
	searchText := textFromResult(searchRes)
	expectedFileID := "org.jetbrains.kotlinx:kotlinx-datetime:0.7.1!/" + inner
	if !strings.Contains(searchText, expectedFileID) {
		t.Fatalf("unexpected search output: %s", searchText)
	}

	catRes, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "cat",
		Arguments: map[string]any{
			"fileId": expectedFileID,
			"lines":  "2,2",
		},
	})
	if err != nil {
		t.Fatalf("cat tool: %v", err)
	}
	catText := strings.TrimSpace(textFromResult(catRes))
	if catText != "public class LocalDate" {
		t.Fatalf("unexpected cat output: %q", catText)
	}
}

func TestMCPServerWherePathEmitsFullyQualifiedFileID(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	root := filepath.Clean(filepath.Join(wd, "..", ".."))
	projectDir := filepath.Join(root, "testdata", "fixture")
	jarPath := filepath.Join(t.TempDir(), "kotlinx-datetime-sources.jar")
	inner := "kotlinx/datetime/LocalDate.kt"

	if err := writeZipFile(jarPath, inner, "before\npublic class LocalDate\nafter\n"); err != nil {
		t.Fatalf("write jar: %v", err)
	}

	ctx, session, cleanup := startTestSession(t, root, projectDir, jarPath)
	defer cleanup()

	whereRes, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "where",
		Arguments: map[string]any{
			"pathOrCoord": inner,
			"group":       "org.jetbrains.kotlinx",
			"artifact":    "kotlinx-datetime",
			"project":     projectDir,
			"subprojects": []string{},
		},
	})
	if err != nil {
		t.Fatalf("where tool: %v", err)
	}
	whereText := textFromResult(whereRes)
	want := "org.jetbrains.kotlinx:kotlinx-datetime:0.7.1!/" + inner + "|" + jarPath + "\n"
	if whereText != want {
		t.Fatalf("unexpected where output:\nwant: %q\n got: %q", want, whereText)
	}

	fileID, _, ok := strings.Cut(strings.TrimSpace(whereText), "|")
	if !ok {
		t.Fatalf("missing file-id separator in output: %q", whereText)
	}

	catRes, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "cat",
		Arguments: map[string]any{
			"fileId": fileID,
			"lines":  "2,2",
		},
	})
	if err != nil {
		t.Fatalf("cat tool: %v", err)
	}
	catText := strings.TrimSpace(textFromResult(catRes))
	if catText != "public class LocalDate" {
		t.Fatalf("unexpected cat output: %q", catText)
	}
}

func TestMCPServerWherePathUsesZipDirectoryEntriesBeforeCatReadsContent(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	root := filepath.Clean(filepath.Join(wd, "..", ".."))
	projectDir := filepath.Join(root, "testdata", "fixture")
	jarPath := filepath.Join(t.TempDir(), "kotlinx-datetime-sources.jar")
	inner := "kotlinx/datetime/LocalDate.kt"

	if err := writeZipFile(jarPath, inner, "before\npublic class LocalDate\nafter\n"); err != nil {
		t.Fatalf("write jar: %v", err)
	}
	if err := testutil.CorruptZipEntryMethod(jarPath, inner, 99); err != nil {
		t.Fatalf("corrupt zip entry method: %v", err)
	}

	ctx, session, cleanup := startTestSession(t, root, projectDir, jarPath)
	defer cleanup()

	whereRes, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "where",
		Arguments: map[string]any{
			"pathOrCoord": inner,
			"group":       "org.jetbrains.kotlinx",
			"artifact":    "kotlinx-datetime",
			"project":     projectDir,
			"subprojects": []string{},
		},
	})
	if err != nil {
		t.Fatalf("where tool: %v", err)
	}
	if whereRes.IsError {
		t.Fatalf("where tool returned error: %s", textFromResult(whereRes))
	}
	whereText := textFromResult(whereRes)
	want := "org.jetbrains.kotlinx:kotlinx-datetime:0.7.1!/" + inner + "|" + jarPath + "\n"
	if whereText != want {
		t.Fatalf("unexpected where output:\nwant: %q\n got: %q", want, whereText)
	}

	fileID, _, ok := strings.Cut(strings.TrimSpace(whereText), "|")
	if !ok {
		t.Fatalf("missing file-id separator in output: %q", whereText)
	}

	catRes, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "cat",
		Arguments: map[string]any{
			"fileId": fileID,
		},
	})
	if err != nil {
		t.Fatalf("cat tool: %v", err)
	}
	if !catRes.IsError {
		t.Fatal("expected cat tool to fail when entry body cannot be read")
	}
	if text := textFromResult(catRes); !strings.Contains(text, "unsupported compression algorithm") {
		t.Fatalf("expected body read failure, got: %q", text)
	}
}

func TestMCPServerDepsResolveFetchIntegration(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	root := filepath.Clean(filepath.Join(wd, "..", ".."))
	projectDir := filepath.Join(root, "testdata", "fixture")
	jarPath := filepath.Join(t.TempDir(), "kotlinx-datetime-sources.jar")
	inner := "kotlinx/datetime/LocalDate.kt"
	coord := "org.jetbrains.kotlinx:kotlinx-datetime:0.7.1"

	if err := writeZipFile(jarPath, inner, "before\npublic class LocalDate\nafter\n"); err != nil {
		t.Fatalf("write jar: %v", err)
	}

	ctx, session, cleanup := startTestSession(t, root, projectDir, jarPath)
	defer cleanup()

	depsRes, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "deps",
		Arguments: map[string]any{
			"project": projectDir,
		},
	})
	if err != nil {
		t.Fatalf("deps tool: %v", err)
	}
	depsText := textFromResult(depsRes)
	if !strings.Contains(depsText, coord+"  [sources: yes]  [path: "+jarPath+"]") {
		t.Fatalf("unexpected deps output: %q", depsText)
	}

	resolveRes, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "resolve",
		Arguments: map[string]any{
			"project":  projectDir,
			"group":    "org.jetbrains.kotlinx",
			"artifact": "kotlinx-datetime",
		},
	})
	if err != nil {
		t.Fatalf("resolve tool: %v", err)
	}
	resolveText := textFromResult(resolveRes)
	if !strings.Contains(resolveText, coord+"|"+jarPath) {
		t.Fatalf("unexpected resolve output: %q", resolveText)
	}

	fetchRes, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "fetch",
		Arguments: map[string]any{
			"project":  projectDir,
			"group":    "org.jetbrains.kotlinx",
			"artifact": "kotlinx-datetime",
			"version":  "0.7.1",
		},
	})
	if err != nil {
		t.Fatalf("fetch tool: %v", err)
	}
	fetchText := textFromResult(fetchRes)
	if fetchText != coord+"|"+jarPath+"\n" {
		t.Fatalf("unexpected fetch output: %q", fetchText)
	}
}

func startTestSession(t *testing.T, root string, projectDir string, jarPath string) (context.Context, toolSession, func()) {
	return startTestSessionAt(t, root, projectDir, jarPath)
}

func startTestSessionAt(t *testing.T, root string, workDir string, jarPath string) (context.Context, toolSession, func()) {
	t.Helper()

	binPath := filepath.Join(t.TempDir(), "ksrc")
	buildCmd := exec.Command("go", "build", "-o", binPath, "./cmd/ksrc")
	buildCmd.Dir = root
	buildOut, err := buildCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("build ksrc: %v\n%s", err, string(buildOut))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	cmd := exec.CommandContext(ctx, binPath, "mcp", "--tools=all")
	cmd.Dir = workDir
	cacheDir := filepath.Join(t.TempDir(), "fileid-cache")
	cmd.Env = append(os.Environ(),
		"KSRC_TEST_JAR="+jarPath,
		"KSRC_FILEID_CACHE_DIR="+cacheDir,
	)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		cancel()
		t.Fatalf("stdin pipe: %v", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		t.Fatalf("stdout pipe: %v", err)
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		cancel()
		t.Fatalf("start mcp: %v", err)
	}

	client := mcp.NewClient(&mcp.Implementation{Name: "client", Version: "v0.0.1"}, nil)
	session, err := client.Connect(ctx, &mcp.IOTransport{Reader: stdout, Writer: stdin}, nil)
	if err != nil {
		_ = cmd.Process.Kill()
		cancel()
		t.Fatalf("connect mcp: %v", err)
	}

	cleanup := func() {
		_ = session.Close()
		_ = stdin.Close()
		_ = stdout.Close()
		_ = cmd.Wait()
		cancel()
	}
	return ctx, session, cleanup
}

func textFromResult(res *mcp.CallToolResult) string {
	if res == nil {
		return ""
	}
	var sb strings.Builder
	for _, c := range res.Content {
		if text, ok := c.(*mcp.TextContent); ok {
			sb.WriteString(text.Text)
		}
	}
	return sb.String()
}

func writeZipFile(path, inner, content string) error {
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

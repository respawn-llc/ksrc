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
)

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

	binPath := filepath.Join(t.TempDir(), "ksrc")
	buildCmd := exec.Command("go", "build", "-o", binPath, "./cmd/ksrc")
	buildCmd.Dir = root
	buildOut, err := buildCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("build ksrc: %v\n%s", err, string(buildOut))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, binPath, "mcp")
	cmd.Dir = projectDir
	cmd.Env = append(os.Environ(), "KSRC_TEST_JAR="+jarPath)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatalf("stdin pipe: %v", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("stdout pipe: %v", err)
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		t.Fatalf("start mcp: %v", err)
	}

	client := mcp.NewClient(&mcp.Implementation{Name: "client", Version: "v0.0.1"}, nil)
	session, err := client.Connect(ctx, &mcp.IOTransport{Reader: stdout, Writer: stdin}, nil)
	if err != nil {
		_ = cmd.Process.Kill()
		t.Fatalf("connect mcp: %v", err)
	}

	searchRes, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "search",
		Arguments: map[string]any{
			"query":      "public class LocalDate",
			"group":      "org.jetbrains.kotlinx",
			"artifact":   "kotlinx-datetime",
			"project":    projectDir,
			"subprojects": []string{},
		},
	})
	if err != nil {
		_ = session.Close()
		_ = cmd.Process.Kill()
		t.Fatalf("search tool: %v", err)
	}
	searchText := textFromResult(searchRes)
	expectedFileID := "org.jetbrains.kotlinx:kotlinx-datetime:0.6.1!/" + inner
	if !strings.Contains(searchText, expectedFileID) {
		t.Fatalf("unexpected search output: %s", searchText)
	}
	fileID := expectedFileID

	catRes, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "cat",
		Arguments: map[string]any{
			"fileId": fileID,
			"lines":  "2,2",
		},
	})
	if err != nil {
		_ = session.Close()
		_ = cmd.Process.Kill()
		t.Fatalf("cat tool: %v", err)
	}
	catText := strings.TrimSpace(textFromResult(catRes))
	if catText != "public class LocalDate" {
		t.Fatalf("unexpected cat output: %q", catText)
	}

	_ = session.Close()
	_ = stdin.Close()
	_ = stdout.Close()
	_ = cmd.Wait()
	_ = stderr
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

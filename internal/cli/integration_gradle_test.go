package cli

import (
	"os"
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

package cli

import (
	"strings"
	"testing"
)

func TestVersionFlag(t *testing.T) {
	orig := Version
	Version = "9.9.9"
	t.Cleanup(func() {
		Version = orig
	})

	out, err := runCommand(NewApp(), []string{"--version"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.TrimSpace(out) != "9.9.9" {
		t.Fatalf("unexpected version output: %q", out)
	}
}

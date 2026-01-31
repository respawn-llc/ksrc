package mcpserver

import "testing"

func TestParseToolsDefaults(t *testing.T) {
	set, err := ParseTools("")
	if err != nil {
		t.Fatalf("ParseTools error: %v", err)
	}
	for _, name := range DefaultToolNames() {
		if !set.Enabled(name) {
			t.Fatalf("expected default tool %s", name)
		}
	}
}

func TestParseToolsAll(t *testing.T) {
	set, err := ParseTools("all")
	if err != nil {
		t.Fatalf("ParseTools error: %v", err)
	}
	for _, name := range KnownTools() {
		if !set.Enabled(name) {
			t.Fatalf("expected tool %s", name)
		}
	}
}

func TestParseToolsList(t *testing.T) {
	set, err := ParseTools("search,cat")
	if err != nil {
		t.Fatalf("ParseTools error: %v", err)
	}
	if !set.Enabled(ToolSearch) || !set.Enabled(ToolCat) {
		t.Fatalf("expected search and cat enabled")
	}
	if set.Enabled(ToolDeps) {
		t.Fatalf("expected deps disabled")
	}
}

func TestParseToolsUnknown(t *testing.T) {
	_, err := ParseTools("search,wat")
	if err == nil {
		t.Fatalf("expected error for unknown tool")
	}
}

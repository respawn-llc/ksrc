package adapter

import (
	"strings"
	"testing"

	"github.com/respawn-app/ksrc/internal/resolve"
	"github.com/respawn-app/ksrc/internal/search"
)

func TestWriteDepsOutputDeduplicatesAndFallsBackToSources(t *testing.T) {
	var sb strings.Builder
	sources := []resolve.SourceJar{{Coord: resolve.Coord{Group: "g", Artifact: "a", Version: "1"}, Path: "/tmp/a-1-sources.jar"}}
	deps := []resolve.Coord{{Group: "g", Artifact: "a", Version: "1"}, {Group: "g", Artifact: "a", Version: "1"}}

	if err := WriteDepsOutput(&sb, sources, deps); err != nil {
		t.Fatalf("WriteDepsOutput error: %v", err)
	}
	if got := sb.String(); got != "g:a:1  [sources: yes]  [path: /tmp/a-1-sources.jar]\n" {
		t.Fatalf("unexpected output: %q", got)
	}

	sb.Reset()
	if err := WriteDepsOutput(&sb, sources, nil); err != nil {
		t.Fatalf("WriteDepsOutput fallback error: %v", err)
	}
	if got := sb.String(); got != "g:a:1  [sources: yes]  [path: /tmp/a-1-sources.jar]\n" {
		t.Fatalf("unexpected fallback output: %q", got)
	}
}

func TestWriteRGCommandReport(t *testing.T) {
	var sb strings.Builder
	plan := search.ExecPlan{
		Cmd:      "rg",
		Args:     []string{"--json", "Demo", "/tmp/jar-1", "/tmp/jar-2"},
		JarCount: 2,
		Mode:     "dirs",
	}

	if err := WriteRGCommandReport(&sb, plan); err != nil {
		t.Fatalf("WriteRGCommandReport error: %v", err)
	}
	want := "VERBOSE: rg: rg --json Demo <2 jars>\nVERBOSE: rg jars: 2 (mode=dirs)\n"
	if got := sb.String(); got != want {
		t.Fatalf("unexpected report: %q", got)
	}
}

func TestWriteSearchMatchesWithExtractedPathUsesQuotedTabSeparatedFields(t *testing.T) {
	var sb strings.Builder
	matches := []search.Match{{
		FileID: "g:a:1!/src/Main.kt",
		File:   "/tmp/with space/src:Main.kt",
		Line:   12,
		Column: 3,
		Text:   "value\twith colon: and quote \"",
	}}

	if err := WriteSearchMatches(&sb, matches, true); err != nil {
		t.Fatalf("WriteSearchMatches error: %v", err)
	}
	want := "g:a:1!/src/Main.kt\t\"/tmp/with space/src:Main.kt\"\t12\t3\t\"value\\twith colon: and quote \\\"\"\n"
	if got := sb.String(); got != want {
		t.Fatalf("unexpected output:\nwant: %q\n got: %q", want, got)
	}
}

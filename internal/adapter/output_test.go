package adapter

import (
	"strings"
	"testing"

	"github.com/respawn-app/ksrc/internal/resolve"
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

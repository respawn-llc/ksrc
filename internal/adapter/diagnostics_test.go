package adapter

import (
	"bytes"
	"testing"

	"github.com/respawn-app/ksrc/internal/resolution"
	"github.com/respawn-app/ksrc/internal/resolve"
)

func TestWriteDiagnosticsHandlesNilAndFormatsWarningsAndVerbose(t *testing.T) {
	t.Parallel()

	WriteDiagnostics(nil, resolution.ResolveMeta{
		Warnings: []string{"warning"},
		Verbose:  []string{"verbose"},
	}, true)

	var out bytes.Buffer
	WriteDiagnostics(&out, resolution.ResolveMeta{
		Warnings: []string{"warning one", "warning two"},
		Verbose:  []string{"  first verbose  ", "   ", "second verbose"},
	}, true)

	want := "WARN: warning one\nWARN: warning two\nVERBOSE: first verbose\nVERBOSE: second verbose\n"
	if out.String() != want {
		t.Fatalf("unexpected diagnostics output:\nwant: %q\n got: %q", want, out.String())
	}
}

func TestNoSourcesErrorFormatsHint(t *testing.T) {
	t.Parallel()

	if got := NoSourcesError("   ").Error(); got != "E_NO_SOURCES: no sources resolved" {
		t.Fatalf("unexpected empty-hint error: %q", got)
	}
	if got := NoSourcesError("Try fetch").Error(); got != "E_NO_SOURCES: no sources resolved. Try fetch" {
		t.Fatalf("unexpected hinted error: %q", got)
	}
}

func TestNoSourcesHintVariants(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		group    string
		artifact string
		version  string
		want     string
	}{
		{
			name:     "fully qualified",
			group:    "org.jetbrains.kotlinx",
			artifact: "kotlinx-datetime",
			version:  "0.7.1",
			want:     "Try: ksrc fetch org.jetbrains.kotlinx:kotlinx-datetime:0.7.1 to download sources.",
		},
		{
			name:     "missing version",
			group:    "org.jetbrains.kotlinx",
			artifact: "kotlinx-datetime",
			want:     "Try: add a version (group:artifact:version) or run ksrc deps to see resolved coords.",
		},
		{
			name: "fallback",
			want: "Try: ksrc deps (list resolved coords), then ksrc fetch <coord> to download sources.",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			if got := NoSourcesHint(testCase.group, testCase.artifact, testCase.version); got != testCase.want {
				t.Fatalf("unexpected hint:\nwant: %q\n got: %q", testCase.want, got)
			}
		})
	}
}

func TestNoSourcesHintForCoordDelegates(t *testing.T) {
	t.Parallel()

	coord := resolve.Coord{Group: "org.jetbrains.kotlinx", Artifact: "kotlinx-datetime", Version: "0.7.1"}
	if got, want := NoSourcesHintForCoord(coord), NoSourcesHint(coord.Group, coord.Artifact, coord.Version); got != want {
		t.Fatalf("unexpected coord hint:\nwant: %q\n got: %q", want, got)
	}
}

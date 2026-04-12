package mcpserver

import (
	"fmt"
	"os"
	"strings"

	"github.com/respawn-app/ksrc/internal/resolution"
	"github.com/respawn-app/ksrc/internal/resolve"
)

func joinCSV(values []string) string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		out = append(out, value)
	}
	if len(out) == 0 {
		return ""
	}
	return strings.Join(out, ",")
}

func cleanList(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		out = append(out, value)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func boolOrDefault(value *bool, def bool) bool {
	if value == nil {
		return def
	}
	return *value
}

func withDefaultString(value string, def string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return def
	}
	return value
}

func noSourcesError(group, artifact, version string) error {
	msg := "E_NO_SOURCES: no sources resolved."
	return fmt.Errorf("%s %s", msg, noSourcesHint(group, artifact, version))
}

func noSourcesHint(group, artifact, version string) string {
	if group != "" && artifact != "" && version != "" {
		return fmt.Sprintf("Try: ksrc fetch %s:%s:%s to download sources.", group, artifact, version)
	}
	if group != "" && artifact != "" {
		return "Try: add a version (group:artifact:version) or run ksrc deps to see resolved coords."
	}
	return "Try: ksrc deps (list resolved coords), then ksrc fetch <coord> to download sources."
}

func filterCoords(coords []resolve.Coord, group, artifact, version string) []resolve.Coord {
	return resolve.FilterCoords(coords, "", group, artifact, version)
}

func emitDiagnostics(meta resolution.ResolveMeta, verbose bool) {
	for _, warning := range meta.Warnings {
		fmt.Fprintf(os.Stderr, "WARN: %s\n", warning)
	}
	if !verbose {
		return
	}
	for _, line := range meta.Verbose {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fmt.Fprintf(os.Stderr, "VERBOSE: %s\n", line)
	}
}

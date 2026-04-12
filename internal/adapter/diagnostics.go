package adapter

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/respawn-app/ksrc/internal/resolution"
	"github.com/respawn-app/ksrc/internal/resolve"
)

func WriteDiagnostics(w io.Writer, meta resolution.ResolveMeta, verbose bool) {
	if w == nil {
		return
	}
	for _, warning := range meta.Warnings {
		fmt.Fprintf(w, "WARN: %s\n", warning)
	}
	if !verbose {
		return
	}
	for _, line := range meta.Verbose {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fmt.Fprintf(w, "VERBOSE: %s\n", line)
	}
}

func NoSourcesError(hint string) error {
	msg := "E_NO_SOURCES: no sources resolved."
	hint = strings.TrimSpace(hint)
	if hint == "" {
		return errors.New(strings.TrimSuffix(msg, "."))
	}
	return errors.New(msg + " " + hint)
}

func NoSourcesHint(group, artifact, version string) string {
	group = strings.TrimSpace(group)
	artifact = strings.TrimSpace(artifact)
	version = strings.TrimSpace(version)
	if group != "" && artifact != "" && version != "" {
		return fmt.Sprintf("Try: ksrc fetch %s:%s:%s to download sources.", group, artifact, version)
	}
	if group != "" && artifact != "" {
		return "Try: add a version (group:artifact:version) or run ksrc deps to see resolved coords."
	}
	return "Try: ksrc deps (list resolved coords), then ksrc fetch <coord> to download sources."
}

func NoSourcesHintForCoord(coord resolve.Coord) string {
	return NoSourcesHint(coord.Group, coord.Artifact, coord.Version)
}

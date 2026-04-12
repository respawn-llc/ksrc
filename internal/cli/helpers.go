package cli

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/respawn-app/ksrc/internal/adapter"
	"github.com/respawn-app/ksrc/internal/resolution"
	"github.com/respawn-app/ksrc/internal/resolve"
)

func resolveSources(ctx context.Context, app *App, flags ResolveFlags, dep string, applyFilters bool, allowCacheFallback bool) ([]resolve.SourceJar, []resolve.Coord, resolution.ResolveMeta, error) {
	result, err := adapter.Resolver{Runner: app.Runner, Verbose: app.Verbose}.ResolveSources(ctx, flags.ToSpec(dep, applyFilters, allowCacheFallback))
	return result.Sources, result.Deps, result.Meta, err
}

func noSourcesErr(flags ResolveFlags, hint string) error {
	var parts []string
	if flags.Offline {
		parts = append(parts, "You ran with --offline; rerun without it to allow downloads.")
	}
	if strings.TrimSpace(hint) != "" {
		parts = append(parts, hint)
	}
	return adapter.NoSourcesError(strings.Join(parts, " "))
}

func noSourcesHintForFlags(flags ResolveFlags, meta resolution.ResolveMeta) string {
	if flags.All {
		return joinHints(
			"Try: ksrc deps (list resolved coords), then ksrc fetch <coord> to download sources.",
			projectHint(meta),
		)
	}
	if coord, ok := resolve.SelectorToCoord(flags.Module, flags.Group, flags.Artifact, flags.Version); ok {
		if coord.Version != "" {
			return joinHints(
				fmt.Sprintf("Try: ksrc fetch %s to download sources.", coord.String()),
				projectHint(meta),
			)
		}
		if coord.Group != "" || coord.Artifact != "" {
			return joinHints(
				"Try: add a version (group:artifact:version) or run ksrc deps to see resolved coords.",
				projectHint(meta),
			)
		}
	}
	return joinHints(
		"Try: ksrc deps (list resolved coords), then ksrc fetch <coord> to download sources.",
		projectHint(meta),
	)
}

func noSourcesHintForCoord(coord resolve.Coord) string {
	return adapter.NoSourcesHintForCoord(coord)
}

func joinHints(parts ...string) string {
	var out []string
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	if len(out) == 0 {
		return ""
	}
	return strings.Join(out, " ")
}

func projectHint(meta resolution.ResolveMeta) string {
	if len(meta.IncludedBuilds) == 0 {
		return ""
	}
	return fmt.Sprintf("Composite build detected; try: --project %s", filepath.Clean(meta.IncludedBuilds[0]))
}

func emitDiagnostics(cmd stderrWriter, meta resolution.ResolveMeta, verbose bool) {
	adapter.WriteDiagnostics(cmd.ErrOrStderr(), meta, verbose)
}

type stderrWriter interface {
	ErrOrStderr() io.Writer
}

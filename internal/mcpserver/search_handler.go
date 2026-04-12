package mcpserver

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/respawn-app/ksrc/internal/adapter"
	"github.com/respawn-app/ksrc/internal/search"
)

func (s *toolState) handleSearch(ctx context.Context, call *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	input, err := decodeInput[SearchInput](call)
	if err != nil {
		return nil, err
	}
	query := strings.TrimSpace(input.Query)
	if query == "" {
		return toolError(fmt.Errorf("query is required")), nil
	}

	spec := buildSearchSpec(input)
	result, err := s.resolver().ResolveSources(ctx, spec)
	if err != nil {
		return toolError(err), nil
	}
	s.emitDiagnostics(result.Meta)
	if len(result.Sources) == 0 {
		return toolError(adapter.NoSourcesError(adapter.NoSourcesHint(input.Group, input.Artifact, input.Version))), nil
	}
	if _, err := s.runner.LookPath("rg"); err != nil {
		return toolError(fmt.Errorf("rg not found on PATH, ask the user to install ripgrep first. The user can run `ksrc doctor` to get guidance.")), nil
	}

	rgArgs, err := cleanRgArgs(input.RgArgs)
	if err != nil {
		return toolError(err), nil
	}
	if input.Context > 0 {
		rgArgs = append(rgArgs, "-C", fmt.Sprintf("%d", input.Context))
	}

	var report func(search.ExecPlan)
	if s.verbose {
		report = func(plan search.ExecPlan) {
			_ = adapter.WriteRGCommandReport(os.Stderr, plan)
		}
	}

	request := adapter.BuildRequest(spec)
	matches, err := search.Run(ctx, s.runner, search.Options{
		Pattern: query,
		Jars:    result.Sources,
		RGArgs:  rgArgs,
		WorkDir: request.Project,
		Report:  report,
	})
	if err != nil {
		return toolError(err), nil
	}
	adapter.TryTrackSearchMatches(matches)
	if len(matches) == 0 {
		return textResult("no results"), nil
	}
	return builderResult(func(sb *strings.Builder) error {
		return adapter.WriteSearchMatches(sb, matches, false)
	})
}

package mcpserver

import (
	"context"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/respawn-app/ksrc/internal/adapter"
	"github.com/respawn-app/ksrc/internal/resolve"
)

func (s *toolState) handleFetch(ctx context.Context, call *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	input, err := decodeInput[FetchInput](call)
	if err != nil {
		return nil, err
	}
	group := strings.TrimSpace(input.Group)
	artifact := strings.TrimSpace(input.Artifact)
	version := strings.TrimSpace(input.Version)
	if group == "" || artifact == "" || version == "" {
		return toolError(fmt.Errorf("group, artifact, and version are required. Obtain them from the file id returned by `search` or `deps`")), nil
	}
	coord := resolve.Coord{Group: group, Artifact: artifact, Version: version}

	result, err := s.resolver().ResolveSources(ctx, buildFetchSpec(input, coord))
	if err != nil {
		return toolError(err), nil
	}
	s.emitDiagnostics(result.Meta)
	if len(result.Sources) == 0 {
		return toolError(adapter.NoSourcesError(adapter.NoSourcesHint(group, artifact, version))), nil
	}
	return builderResult(func(sb *strings.Builder) error {
		return adapter.WriteCoordMatches(sb, result.Sources, coord)
	})
}

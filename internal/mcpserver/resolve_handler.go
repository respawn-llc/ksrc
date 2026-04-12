package mcpserver

import (
	"context"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/respawn-app/ksrc/internal/adapter"
)

func (s *toolState) handleResolve(ctx context.Context, call *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	input, err := decodeInput[ResolveInput](call)
	if err != nil {
		return nil, err
	}
	result, err := s.resolver().ResolveSources(ctx, buildResolveToolSpec(input))
	if err != nil {
		return toolError(err), nil
	}
	s.emitDiagnostics(result.Meta)
	if len(result.Sources) == 0 {
		return toolError(adapter.NoSourcesError(adapter.NoSourcesHint(input.Group, input.Artifact, input.Version))), nil
	}
	return builderResult(func(sb *strings.Builder) error {
		return adapter.WriteCoordPaths(sb, result.Sources)
	})
}

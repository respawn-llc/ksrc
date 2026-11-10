package mcpserver

import (
	"context"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/respawn-app/ksrc/internal/adapter"
	"github.com/respawn-app/ksrc/internal/resolve"
)

func (s *toolState) handleDeps(ctx context.Context, call *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	input, err := decodeInput[DepsInput](call)
	if err != nil {
		return nil, err
	}
	spec := buildDepsSpec(input)
	if _, ok, err := resolve.ExactPlatformCoord("", input.Group, input.Artifact, input.Version); err != nil {
		return toolError(err), nil
	} else if ok {
		spec.Group = input.Group
		spec.Artifact = input.Artifact
		spec.Version = input.Version
	}
	result, err := s.resolver().ResolveSources(ctx, spec)
	if err != nil {
		return toolError(err), nil
	}
	s.emitDiagnostics(result.Meta)

	group := strings.TrimSpace(input.Group)
	artifact := strings.TrimSpace(input.Artifact)
	version := strings.TrimSpace(input.Version)
	filteredSources := resolve.FilterSources(result.Sources, "", group, artifact, version)
	filteredDeps := resolve.FilterCoords(result.Deps, "", group, artifact, version)

	return builderResult(func(sb *strings.Builder) error {
		return adapter.WriteDepsOutput(sb, filteredSources, filteredDeps)
	})
}

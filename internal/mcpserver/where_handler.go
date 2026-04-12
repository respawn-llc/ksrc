package mcpserver

import (
	"context"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/respawn-app/ksrc/internal/adapter"
	"github.com/respawn-app/ksrc/internal/resolve"
)

func (s *toolState) handleWhere(ctx context.Context, call *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	input, err := decodeInput[WhereInput](call)
	if err != nil {
		return nil, err
	}
	arg := strings.TrimSpace(input.PathOrCoord)
	if arg == "" {
		return toolError(fmt.Errorf("pathOrCoord is required")), nil
	}

	if strings.Contains(arg, "!/") {
		coord, inner, err := resolve.ParseFileID(arg)
		if err != nil {
			return toolError(err), nil
		}
		result, err := s.resolver().ResolveSources(ctx, buildWhereCoordSpec(input, coord, coord.String()))
		if err != nil {
			return toolError(err), nil
		}
		s.emitDiagnostics(result.Meta)
		if len(result.Sources) == 0 {
			return toolError(adapter.NoSourcesError(adapter.NoSourcesHintForCoord(coord))), nil
		}
		jarPath, err := adapter.FindJarByCoord(result.Sources, coord, toolFetchHint())
		if err != nil {
			return toolError(err), nil
		}
		return builderResult(func(sb *strings.Builder) error {
			return adapter.WriteFileIDPath(sb, resolve.FormatFileID(coord, inner), jarPath)
		})
	}

	if coord, err := resolve.ParseCoord(arg); err == nil {
		dep := ""
		if coord.Version != "" {
			dep = coord.String()
		}
		result, err := s.resolver().ResolveSources(ctx, buildWhereCoordSpec(input, coord, dep))
		if err != nil {
			return toolError(err), nil
		}
		s.emitDiagnostics(result.Meta)
		if len(result.Sources) == 0 {
			return toolError(adapter.NoSourcesError(adapter.NoSourcesHintForCoord(coord))), nil
		}
		jarPath, err := adapter.FindJarByCoord(result.Sources, coord, toolFetchHint())
		if err != nil {
			return toolError(err), nil
		}
		return builderResult(func(sb *strings.Builder) error {
			return adapter.WriteCoordPath(sb, coord, jarPath)
		})
	}

	group := strings.TrimSpace(input.Group)
	artifact := strings.TrimSpace(input.Artifact)
	version := strings.TrimSpace(input.Version)
	if group == "" || artifact == "" {
		return toolError(fmt.Errorf("path requires group and artifact filters or a file-id")), nil
	}
	path := strings.TrimPrefix(arg, "/")
	result, err := s.resolver().ResolveSources(ctx, buildWhereSpec(input, group, artifact, version, ""))
	if err != nil {
		return toolError(err), nil
	}
	s.emitDiagnostics(result.Meta)
	if len(result.Sources) == 0 {
		return toolError(adapter.NoSourcesError(adapter.NoSourcesHint(group, artifact, version))), nil
	}
	location, ok := adapter.FindFile(result.Sources, path)
	if !ok {
		return toolError(fmt.Errorf("file not found in resolved sources: %s. Try specifying: `project` (for monorepos), `scope` for build time deps etc., or `configs` for non-standard compilations.", strings.TrimPrefix(path, "/"))), nil
	}
	return builderResult(func(sb *strings.Builder) error {
		return adapter.WriteFileLocation(sb, location)
	})
}

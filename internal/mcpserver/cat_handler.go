package mcpserver

import (
	"context"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/respawn-app/ksrc/internal/adapter"
	"github.com/respawn-app/ksrc/internal/cat"
	"github.com/respawn-app/ksrc/internal/resolve"
)

func (s *toolState) handleCat(ctx context.Context, call *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	input, err := decodeInput[CatInput](call)
	if err != nil {
		return nil, err
	}
	fileID := strings.TrimSpace(input.FileID)
	if fileID == "" {
		return toolError(fmt.Errorf("fileId is required. Obtain it from `search` tool output, the file id is the string before the line:column")), nil
	}
	coord, inner, err := resolve.ParseFileID(fileID)
	if err != nil {
		return toolError(err), nil
	}
	lr, err := cat.ParseLineRange(input.Lines)
	if err != nil {
		return toolError(err), nil
	}

	result, err := s.resolver().ResolveSources(ctx, buildFileIDSpec(coord))
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
	data, err := cat.ReadFileFromZip(jarPath, inner, lr)
	if err != nil {
		return toolError(err), nil
	}
	return textResult(string(data)), nil
}

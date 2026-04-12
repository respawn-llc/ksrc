package mcpserver

import (
	"encoding/json"
	"os"
	"strings"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/respawn-app/ksrc/internal/adapter"
	"github.com/respawn-app/ksrc/internal/executil"
	"github.com/respawn-app/ksrc/internal/resolution"
)

type toolState struct {
	runner  executil.Runner
	verbose bool
}

func (s *toolState) resolver() adapter.Resolver {
	return adapter.Resolver{Runner: s.runner, Verbose: s.verbose}
}

func (s *toolState) emitDiagnostics(meta resolution.ResolveMeta) {
	adapter.WriteDiagnostics(os.Stderr, meta, s.verbose)
}

func textResult(text string) *mcp.CallToolResult {
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: text}}}
}

func builderResult(write func(*strings.Builder) error) (*mcp.CallToolResult, error) {
	var sb strings.Builder
	if err := write(&sb); err != nil {
		return nil, err
	}
	return textResult(sb.String()), nil
}

func toolError(err error) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}},
	}
}

func decodeInput[T any](req *mcp.CallToolRequest) (T, error) {
	var input T
	if req == nil || req.Params == nil || req.Params.Arguments == nil {
		return input, nil
	}
	if err := json.Unmarshal(req.Params.Arguments, &input); err != nil {
		return input, err
	}
	return input, nil
}

func mustInputSchema[T any]() *jsonschema.Schema {
	schema, err := jsonschema.For[T](nil)
	if err != nil {
		panic(err)
	}
	return schema
}

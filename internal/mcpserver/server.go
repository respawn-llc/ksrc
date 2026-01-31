package mcpserver

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/respawn-app/ksrc/internal/executil"
)

type Options struct {
	Runner  executil.Runner
	Verbose bool
	Tools   ToolSet
	Version string
}

func Run(ctx context.Context, opts Options) error {
	if opts.Runner == nil {
		return fmt.Errorf("runner is required")
	}
	if opts.Version == "" {
		opts.Version = "dev"
	}
	if opts.Tools == nil {
		opts.Tools = DefaultTools()
	}
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "ksrc",
		Version: opts.Version,
	}, nil)

	state := &toolState{runner: opts.Runner, verbose: opts.Verbose}
	registerTools(server, state, opts.Tools)

	transport := &mcp.StdioTransport{}
	return server.Run(ctx, transport)
}

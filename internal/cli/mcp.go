package cli

import (
	"context"

	"github.com/respawn-app/ksrc/internal/mcpserver"
	"github.com/spf13/cobra"
)

func newMcpCmd(app *App) *cobra.Command {
	var tools string

	cmd := &cobra.Command{
		Use:   "mcp",
		Short: "Run MCP server over stdio",
		RunE: func(cmd *cobra.Command, args []string) error {
			set, err := mcpserver.ParseTools(tools)
			if err != nil {
				return err
			}
			return mcpserver.Run(context.Background(), mcpserver.Options{
				Runner:  app.Runner,
				Verbose: app.Verbose,
				Tools:   set,
				Version: versionString(),
			})
		},
	}

	cmd.Flags().StringVar(&tools, "tools", "", "comma-separated tool list (default: search,cat,deps; use 'all' for all tools)")
	return cmd
}

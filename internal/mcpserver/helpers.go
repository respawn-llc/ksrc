package mcpserver

import "github.com/respawn-app/ksrc/internal/adapter"

func cleanList(values []string) []string { return adapter.CleanList(values) }

func boolOrDefault(value *bool, def bool) bool { return adapter.BoolOrDefault(value, def) }

func toolFetchHint() string {
	return "Try: calling `fetch` tool, or if you don't see it, ask the user to enable with `ksrc mcp --tools=all`."
}

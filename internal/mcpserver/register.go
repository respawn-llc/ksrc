package mcpserver

import (
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func registerTools(server *mcp.Server, state *toolState, tools ToolSet) {
	if tools.Enabled(ToolSearch) {
		server.AddTool(&mcp.Tool{
			Name:        toolName(ToolSearch),
			Description: "Avoid directly accessing `.gradle`; Instead, proactively use this tool to find third-party Gradle dependency sources & learn unfamiliar APIs. Start by calling `search` (this tool) and pass `query` (rg-style globs) to find matches. This returns file-id and the match: `group:artifact:version!/path/inside/jar.ext line:col: <context>. Then pass returned file-id to the `cat` tool to read the file content",
			InputSchema: mustInputSchema[SearchInput](),
		}, state.handleSearch)
	}
	if tools.Enabled(ToolCat) {
		server.AddTool(&mcp.Tool{
			Name:        toolName(ToolCat),
			Description: "Read a file by file-id returned from ksrc `search` or `where`. Follow-up reads reuse the tracked backing jar path when available; for standalone/cache-miss reads, pass same `project`/scope/config filters used during search. Recommended: pass `lines` range as \"A,B\" to avoid reading large files.",
			InputSchema: mustInputSchema[CatInput](),
		}, state.handleCat)
	}
	if tools.Enabled(ToolDeps) {
		server.AddTool(&mcp.Tool{
			Name:        toolName(ToolDeps),
			Description: "List resolved dependencies and whether their sources are available. Use when no matches or sources are found unexpectedly. By default, search already download deps, but you may need to use/ ask the user to enable `fetch` tool if you need to fetch a dependency that your project does not depend on.",
			InputSchema: mustInputSchema[DepsInput](),
		}, state.handleDeps)
	}
	if tools.Enabled(ToolFetch) {
		server.AddTool(&mcp.Tool{
			Name:        toolName(ToolFetch),
			Description: "Ensure sources for a coordinate exist in Gradle caches. You may need to call this if current `project` (by default the cwd) doesn't directly use the target dependency, e.g. composite builds or multiple subprojects",
			InputSchema: mustInputSchema[FetchInput](),
		}, state.handleFetch)
	}
	if tools.Enabled(ToolResolve) {
		server.AddTool(&mcp.Tool{
			Name:        toolName(ToolResolve),
			Description: "Resolve dependency source jars. Use this to list all source jars that may contain needed dependency, or when diagnosing missing sources.",
			InputSchema: mustInputSchema[ResolveInput](),
		}, state.handleResolve)
	}
	if tools.Enabled(ToolWhere) {
		server.AddTool(&mcp.Tool{
			Name:        toolName(ToolWhere),
			Description: "Locate cached source artifact or file and return `<coord>|<jar-path>` or `<file-id>|<jar-path>`. For path lookups, the emitted file-id is reusable with `cat`.",
			InputSchema: mustInputSchema[WhereInput](),
		}, state.handleWhere)
	}
}

func toolName(name string) string {
	return name
}

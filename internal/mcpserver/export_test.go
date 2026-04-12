package mcpserver

import (
	"github.com/respawn-app/ksrc/internal/adapter"
	"github.com/respawn-app/ksrc/internal/resolution"
	"github.com/respawn-app/ksrc/internal/resolve"
)

func BuildSearchRequestForTest(input SearchInput) resolution.Request {
	return adapter.BuildRequest(buildSearchSpec(input))
}

func BuildResolveRequestForTest(input ResolveInput) resolution.Request {
	return adapter.BuildRequest(buildResolveToolSpec(input))
}

func BuildFetchRequestForTest(input FetchInput, coord resolve.Coord) resolution.Request {
	return adapter.BuildRequest(buildFetchSpec(input, coord))
}

func BuildWhereCoordRequestForTest(input WhereInput, coord resolve.Coord, dep string) resolution.Request {
	return adapter.BuildRequest(buildWhereCoordSpec(input, coord, dep))
}

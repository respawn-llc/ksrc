package adapter

import (
	"context"

	"github.com/respawn-app/ksrc/internal/executil"
	"github.com/respawn-app/ksrc/internal/resolution"
)

type Resolver struct {
	Runner  executil.Runner
	Verbose bool
}

func (r Resolver) ResolveSources(ctx context.Context, spec ResolveSpec) (resolution.Result, error) {
	service := resolution.Service{Runner: r.Runner, Verbose: r.Verbose}
	return service.ResolveSources(ctx, BuildRequest(spec))
}

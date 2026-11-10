package adapter

import (
	"context"
	"strings"

	"github.com/respawn-app/ksrc/internal/executil"
	"github.com/respawn-app/ksrc/internal/resolution"
	"github.com/respawn-app/ksrc/internal/resolve"
)

type Resolver struct {
	Runner  executil.Runner
	Verbose bool
}

func (r Resolver) ResolveSources(ctx context.Context, spec ResolveSpec) (resolution.Result, error) {
	if result, ok, err := resolveAndroidPlatform(spec); ok || err != nil {
		return result, err
	}
	service := resolution.Service{Runner: r.Runner, Verbose: r.Verbose}
	return service.ResolveSources(ctx, BuildRequest(spec))
}

func resolveAndroidPlatform(spec ResolveSpec) (resolution.Result, bool, error) {
	coord, ok, err := resolve.ExactPlatformCoord(spec.Module, spec.Group, spec.Artifact, spec.Version)
	if err != nil || !ok {
		return resolution.Result{}, ok, err
	}
	project := strings.TrimSpace(spec.Project)
	if project == "" {
		project = "."
	}
	sdkRoot, err := resolve.FindAndroidSDK(spec.AndroidHome, project)
	if err != nil {
		return resolution.Result{}, true, err
	}
	loaded, err := resolve.LoadPlatform(sdkRoot, coord.Version)
	if err != nil {
		return resolution.Result{}, true, err
	}
	return resolution.Result{
		Sources: loaded.Sources,
		Deps:    []resolve.Coord{loaded.Coord},
		Meta:    resolution.ResolveMeta{Warnings: loaded.Warnings},
	}, true, nil
}

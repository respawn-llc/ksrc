package gradle

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/respawn-app/ksrc/internal/executil"
	"github.com/respawn-app/ksrc/internal/gradlehome"
)

type SingleResolver interface {
	ResolveOnce(ctx context.Context, runner executil.Runner, opts ResolveOptions) (ResolveResult, error)
}

type resolveOnceFunc func(context.Context, executil.Runner, ResolveOptions) (ResolveResult, error)

func (f resolveOnceFunc) ResolveOnce(ctx context.Context, runner executil.Runner, opts ResolveOptions) (ResolveResult, error) {
	return f(ctx, runner, opts)
}

func Resolve(ctx context.Context, runner executil.Runner, opts ResolveOptions) (ResolveResult, error) {
	normalizedOpts, err := normalizeTraversalOptions(opts)
	if err != nil {
		return ResolveResult{}, err
	}
	opts = normalizedOpts
	return resolveWith(ctx, runner, opts, resolveOnceFunc(resolveOnce))
}

func resolveWith(ctx context.Context, runner executil.Runner, opts ResolveOptions, resolver SingleResolver) (ResolveResult, error) {
	normalizedOpts, err := normalizeTraversalOptions(opts)
	if err != nil {
		return ResolveResult{}, err
	}
	opts = normalizedOpts
	rootOpts := opts
	rootOpts.ProjectPath = ""
	rootOpts.RootDir = opts.ProjectDir

	rootRes, err := resolver.ResolveOnce(ctx, runner, rootOpts)
	if err != nil {
		return ResolveResult{}, err
	}

	combined := rootRes
	if opts.Verbose && len(rootRes.IncludedBuilds) > 0 {
		combined.Verbose = append(combined.Verbose, fmt.Sprintf("Included builds detected: %s", strings.Join(rootRes.IncludedBuilds, ", ")))
	}
	if opts.Verbose && !opts.IncludeIncludedBuilds && len(rootRes.IncludedBuilds) > 0 {
		combined.Verbose = append(combined.Verbose, "Included builds scanning disabled (--include-builds=false).")
	}
	if len(combined.Sources) > 0 {
		return combined, nil
	}
	if opts.Offline {
		if opts.Verbose {
			combined.Verbose = append(combined.Verbose, "Offline: skipping buildSrc and included builds.")
		}
		return combined, nil
	}

	if opts.IncludeBuildSrc {
		buildSrcDir := filepath.Join(opts.ProjectDir, "buildSrc")
		if shouldResolveBuildSrc(buildSrcDir, opts.ProjectDir, opts.ProjectPath) {
			if opts.Verbose {
				combined.Verbose = append(combined.Verbose, fmt.Sprintf("buildSrc scan: %s", buildSrcDir))
			}
			buildSrcOpts := rootOpts
			buildSrcOpts.ProjectPath = buildSrcDir
			buildSrcOpts.Subprojects = nil
			buildSrcRes, err := resolver.ResolveOnce(ctx, runner, buildSrcOpts)
			if err != nil {
				combined.Warnings = append(combined.Warnings, fmt.Sprintf("buildSrc resolve failed (%s): %v", buildSrcDir, err))
			} else {
				combined = mergeResults(combined, buildSrcRes)
				if len(buildSrcRes.Sources) > 0 {
					combined.Warnings = append(combined.Warnings, "resolved sources from buildSrc")
					return combined, nil
				}
			}
		} else if opts.Verbose {
			combined.Verbose = append(combined.Verbose, fmt.Sprintf("buildSrc skipped: %s", buildSrcDir))
		}
	} else if opts.Verbose {
		combined.Verbose = append(combined.Verbose, "buildSrc scanning disabled (--buildsrc=false).")
	}

	if opts.IncludeIncludedBuilds && len(rootRes.IncludedBuilds) > 0 {
		if opts.Verbose {
			combined.Verbose = append(combined.Verbose, fmt.Sprintf("Included builds queued: %d", len(rootRes.IncludedBuilds)))
		}
		buildQueue := append([]string{}, rootRes.IncludedBuilds...)
		seenBuilds := make(map[string]struct{})
		for len(buildQueue) > 0 {
			buildDir := strings.TrimSpace(buildQueue[0])
			buildQueue = buildQueue[1:]
			if buildDir == "" {
				continue
			}
			key := cleanPath(buildDir)
			if _, exists := seenBuilds[key]; exists {
				if opts.Verbose {
					combined.Verbose = append(combined.Verbose, fmt.Sprintf("Included build skipped (duplicate): %s", buildDir))
				}
				continue
			}
			seenBuilds[key] = struct{}{}
			if opts.Verbose {
				combined.Verbose = append(combined.Verbose, fmt.Sprintf("Included build scan: %s", buildDir))
			}

			buildOpts := opts
			buildOpts.ProjectDir = buildDir
			buildOpts.RootDir = opts.ProjectDir
			buildOpts.ProjectPath = ""

			res, err := resolver.ResolveOnce(ctx, runner, buildOpts)
			if err != nil {
				combined.Warnings = append(combined.Warnings, fmt.Sprintf("included build resolve failed (%s): %v", buildDir, err))
				continue
			}
			combined = mergeResults(combined, res)
			if len(res.Sources) > 0 {
				return combined, nil
			}
			for _, inc := range res.IncludedBuilds {
				inc = strings.TrimSpace(inc)
				if inc == "" {
					continue
				}
				if _, exists := seenBuilds[cleanPath(inc)]; exists {
					continue
				}
				buildQueue = append(buildQueue, inc)
			}
		}
	}

	return combined, nil
}

func normalizeTraversalOptions(opts ResolveOptions) (ResolveOptions, error) {
	if strings.TrimSpace(opts.GradleUserHome) == "" {
		return opts, nil
	}
	home, err := gradlehome.Resolve(opts.GradleUserHome, opts.ProjectDir)
	if err != nil {
		return ResolveOptions{}, err
	}
	opts.GradleUserHome = home.Path
	return opts, nil
}

func shouldResolveBuildSrc(buildSrcDir string, projectDir string, projectPath string) bool {
	if projectDir != "" && samePath(buildSrcDir, projectDir) {
		return false
	}
	if projectPath != "" && samePath(buildSrcDir, projectPath) {
		return false
	}
	info, err := os.Stat(buildSrcDir)
	if err != nil || !info.IsDir() {
		return false
	}
	return hasGradleBuildFile(buildSrcDir)
}

func hasGradleBuildFile(dir string) bool {
	if _, err := os.Stat(filepath.Join(dir, "build.gradle")); err == nil {
		return true
	}
	if _, err := os.Stat(filepath.Join(dir, "build.gradle.kts")); err == nil {
		return true
	}
	if _, err := os.Stat(filepath.Join(dir, "settings.gradle")); err == nil {
		return true
	}
	if _, err := os.Stat(filepath.Join(dir, "settings.gradle.kts")); err == nil {
		return true
	}
	return false
}

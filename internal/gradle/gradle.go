package gradle

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/respawn-app/ksrc/internal/executil"
	"github.com/respawn-app/ksrc/internal/resolve"
)

type ResolveOptions struct {
	ProjectDir            string
	RootDir               string
	ProjectPath           string
	Module                string
	Group                 string
	Artifact              string
	Version               string
	Scope                 string
	Configs               []string
	Targets               []string
	Subprojects           []string
	Dep                   string
	Offline               bool
	Refresh               bool
	IncludeBuildSrc       bool
	IncludeBuildscript    bool
	IncludeIncludedBuilds bool
}

type ResolveResult struct {
	Sources        []resolve.SourceJar
	Deps           []resolve.Coord
	IncludedBuilds []string
	Warnings       []string
}

func Resolve(ctx context.Context, runner executil.Runner, opts ResolveOptions) (ResolveResult, error) {
	rootOpts := opts
	rootOpts.ProjectPath = ""
	rootOpts.RootDir = opts.ProjectDir

	rootRes, err := resolveOnce(ctx, runner, rootOpts)
	if err != nil {
		return ResolveResult{}, err
	}

	combined := rootRes
	if len(combined.Sources) > 0 {
		return combined, nil
	}
	if opts.Offline {
		return combined, nil
	}

	if opts.IncludeBuildSrc {
		buildSrcDir := filepath.Join(opts.ProjectDir, "buildSrc")
		if shouldResolveBuildSrc(buildSrcDir, opts.ProjectDir, opts.ProjectPath) {
			buildSrcOpts := rootOpts
			buildSrcOpts.ProjectPath = buildSrcDir
			buildSrcOpts.Subprojects = nil
			buildSrcRes, err := resolveOnce(ctx, runner, buildSrcOpts)
			if err != nil {
				combined.Warnings = append(combined.Warnings, fmt.Sprintf("buildSrc resolve failed (%s): %v", buildSrcDir, err))
			} else {
				combined = mergeResults(combined, buildSrcRes)
				if len(buildSrcRes.Sources) > 0 {
					combined.Warnings = append(combined.Warnings, "resolved sources from buildSrc")
					return combined, nil
				}
			}
		}
	}

	if opts.IncludeIncludedBuilds && len(rootRes.IncludedBuilds) > 0 {
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
				continue
			}
			seenBuilds[key] = struct{}{}

			buildOpts := opts
			buildOpts.ProjectDir = buildDir
			buildOpts.RootDir = opts.ProjectDir
			buildOpts.ProjectPath = ""

			res, err := resolveOnce(ctx, runner, buildOpts)
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

func resolveOnce(ctx context.Context, runner executil.Runner, opts ResolveOptions) (ResolveResult, error) {
	scriptPath, cleanup, err := writeInitScript()
	if err != nil {
		return ResolveResult{}, err
	}
	defer cleanup()

	gradleCmd, err := findGradle(runner, opts.ProjectDir, opts.RootDir)
	if err != nil {
		return ResolveResult{}, err
	}

	args := []string{"-I", scriptPath, "-Dorg.gradle.console=plain", "--info", "--no-configuration-cache"}
	if opts.ProjectPath != "" {
		args = append(args, "-p", opts.ProjectPath)
	}
	if opts.Offline {
		args = append(args, "--offline")
	}
	if opts.Refresh {
		args = append(args, "--refresh-dependencies")
	}
	args = append(args, buildProps(opts)...) // -P...
	args = append(args, "ksrcSources")

	stdout, stderr, err := runner.Run(ctx, opts.ProjectDir, gradleCmd, args...)
	if err != nil {
		return ResolveResult{}, &ExecError{Err: err, Stdout: stdout, Stderr: stderr}
	}

	result := ResolveResult{}
	seen := make(map[string]struct{})
	seenIncludes := make(map[string]struct{})
	for _, line := range strings.Split(stdout, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "KSRC|") {
			coord, path, ok := parseLine(line, "KSRC|")
			if !ok {
				continue
			}
			key := coord.String() + "|" + path
			if _, exists := seen[key]; exists {
				continue
			}
			seen[key] = struct{}{}
			result.Sources = append(result.Sources, resolve.SourceJar{Coord: coord, Path: path})
			continue
		}
		if strings.HasPrefix(line, "KSRCDEP|") {
			coord, _, ok := parseLine(line, "KSRCDEP|")
			if !ok {
				continue
			}
			result.Deps = append(result.Deps, coord)
		}
		if strings.HasPrefix(line, "KSRCINCLUDE|") {
			path := strings.TrimSpace(strings.TrimPrefix(line, "KSRCINCLUDE|"))
			if path == "" {
				continue
			}
			if _, exists := seenIncludes[path]; exists {
				continue
			}
			seenIncludes[path] = struct{}{}
			result.IncludedBuilds = append(result.IncludedBuilds, path)
		}
	}
	return result, nil
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
	if hasGradleBuildFile(buildSrcDir) {
		return true
	}
	return false
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

func samePath(a string, b string) bool {
	aAbs, errA := filepath.Abs(a)
	bAbs, errB := filepath.Abs(b)
	if errA == nil && errB == nil {
		return aAbs == bAbs
	}
	return filepath.Clean(a) == filepath.Clean(b)
}

func mergeResults(base ResolveResult, extra ResolveResult) ResolveResult {
	if len(extra.Sources) == 0 && len(extra.Deps) == 0 && len(extra.IncludedBuilds) == 0 && len(extra.Warnings) == 0 {
		return base
	}
	seenSources := make(map[string]struct{}, len(base.Sources))
	for _, s := range base.Sources {
		seenSources[s.Coord.String()+"|"+s.Path] = struct{}{}
	}
	for _, s := range extra.Sources {
		key := s.Coord.String() + "|" + s.Path
		if _, ok := seenSources[key]; ok {
			continue
		}
		seenSources[key] = struct{}{}
		base.Sources = append(base.Sources, s)
	}

	seenDeps := make(map[string]struct{}, len(base.Deps))
	for _, d := range base.Deps {
		seenDeps[d.String()] = struct{}{}
	}
	for _, d := range extra.Deps {
		key := d.String()
		if _, ok := seenDeps[key]; ok {
			continue
		}
		seenDeps[key] = struct{}{}
		base.Deps = append(base.Deps, d)
	}

	if len(extra.IncludedBuilds) > 0 {
		seenIncludes := make(map[string]struct{}, len(base.IncludedBuilds))
		for _, inc := range base.IncludedBuilds {
			seenIncludes[inc] = struct{}{}
		}
		for _, inc := range extra.IncludedBuilds {
			if _, ok := seenIncludes[inc]; ok {
				continue
			}
			seenIncludes[inc] = struct{}{}
			base.IncludedBuilds = append(base.IncludedBuilds, inc)
		}
	}
	if len(extra.Warnings) > 0 {
		seenWarnings := make(map[string]struct{}, len(base.Warnings))
		for _, warning := range base.Warnings {
			seenWarnings[warning] = struct{}{}
		}
		for _, warning := range extra.Warnings {
			if _, ok := seenWarnings[warning]; ok {
				continue
			}
			seenWarnings[warning] = struct{}{}
			base.Warnings = append(base.Warnings, warning)
		}
	}
	return base
}

func buildProps(opts ResolveOptions) []string {
	props := []string{}
	add := func(k, v string) {
		if strings.TrimSpace(v) == "" {
			return
		}
		props = append(props, "-P"+k+"="+v)
	}
	add("ksrcModule", opts.Module)
	add("ksrcGroup", opts.Group)
	add("ksrcArtifact", opts.Artifact)
	add("ksrcVersion", opts.Version)
	add("ksrcScope", opts.Scope)
	if len(opts.Configs) > 0 {
		add("ksrcConfig", strings.Join(opts.Configs, ","))
	}
	if len(opts.Targets) > 0 {
		add("ksrcTargets", strings.Join(opts.Targets, ","))
	}
	if len(opts.Subprojects) > 0 {
		add("ksrcSubprojects", strings.Join(opts.Subprojects, ","))
	}
	add("ksrcDep", opts.Dep)
	if opts.IncludeBuildscript {
		add("ksrcBuildscript", "true")
	} else {
		add("ksrcBuildscript", "false")
	}
	if opts.IncludeIncludedBuilds {
		add("ksrcIncludeBuilds", "true")
	} else {
		add("ksrcIncludeBuilds", "false")
	}
	return props
}

func cleanPath(path string) string {
	abs, err := filepath.Abs(path)
	if err == nil {
		return abs
	}
	return filepath.Clean(path)
}

func parseLine(line, prefix string) (resolve.Coord, string, bool) {
	trim := strings.TrimPrefix(line, prefix)
	parts := strings.SplitN(trim, "|", 2)
	coord, err := resolve.ParseCoord(parts[0])
	if err != nil {
		return resolve.Coord{}, "", false
	}
	if len(parts) == 1 {
		return coord, "", true
	}
	return coord, strings.TrimSpace(parts[1]), true
}

func findGradle(runner executil.Runner, projectDir string, rootDir string) (string, error) {
	if wrapper := localWrapperPath(projectDir); wrapper != "" {
		return "./gradlew", nil
	}
	if rootDir != "" && !samePath(projectDir, rootDir) {
		if wrapper := localWrapperPath(rootDir); wrapper != "" {
			return wrapper, nil
		}
	}
	path, err := runner.LookPath("gradle")
	if err == nil && path != "" {
		return "gradle", nil
	}
	return "", fmt.Errorf("gradle not found (no ./gradlew and gradle not on PATH)")
}

func localWrapperPath(projectDir string) string {
	wrapper := filepath.Join(projectDir, "gradlew")
	if info, err := os.Stat(wrapper); err == nil && !info.IsDir() {
		abs, err := filepath.Abs(wrapper)
		if err == nil {
			return abs
		}
		return wrapper
	}
	return ""
}

func writeInitScript() (string, func(), error) {
	file, err := os.CreateTemp("", "ksrc-init-*.gradle")
	if err != nil {
		return "", nil, err
	}
	if _, err := file.WriteString(InitScript()); err != nil {
		_ = file.Close()
		_ = os.Remove(file.Name())
		return "", nil, err
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(file.Name())
		return "", nil, err
	}
	cleanup := func() {
		_ = os.Remove(file.Name())
	}
	return file.Name(), cleanup, nil
}

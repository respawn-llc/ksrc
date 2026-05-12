package gradle

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
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
	GradleUserHome        string
	Offline               bool
	Refresh               bool
	IncludeBuildSrc       bool
	IncludeBuildscript    bool
	IncludeIncludedBuilds bool
	Verbose               bool
}

type ResolveResult struct {
	Sources        []resolve.SourceJar
	Deps           []resolve.Coord
	IncludedBuilds []string
	Warnings       []string
	Verbose        []string
}

const recordPrefix = "KSRCJSON\t"
const ksrcGradleTaskName = "__ksrcSources"

func KsrcGradleTaskName() string {
	return ksrcGradleTaskName
}

type recordType string

const (
	recordTypeSource  recordType = "source"
	recordTypeDep     recordType = "dep"
	recordTypeInclude recordType = "include"
)

type outputRecord struct {
	Type       recordType    `json:"type"`
	Group      string        `json:"group,omitempty"`
	Artifact   string        `json:"artifact,omitempty"`
	Version    string        `json:"version,omitempty"`
	Path       string        `json:"path,omitempty"`
	SelectedBy []outputCoord `json:"selectedBy,omitempty"`
}

type outputCoord struct {
	Group    string `json:"group,omitempty"`
	Artifact string `json:"artifact,omitempty"`
	Version  string `json:"version,omitempty"`
}

func (r outputRecord) coord() (resolve.Coord, bool) {
	if strings.TrimSpace(r.Group) == "" || strings.TrimSpace(r.Artifact) == "" || strings.TrimSpace(r.Version) == "" {
		return resolve.Coord{}, false
	}
	return resolve.Coord{Group: r.Group, Artifact: r.Artifact, Version: r.Version}, true
}

func (r outputRecord) selectedByCoords() []resolve.Coord {
	if len(r.SelectedBy) == 0 {
		return nil
	}
	coords := make([]resolve.Coord, 0, len(r.SelectedBy))
	for _, selectedBy := range r.SelectedBy {
		if strings.TrimSpace(selectedBy.Group) == "" || strings.TrimSpace(selectedBy.Artifact) == "" || strings.TrimSpace(selectedBy.Version) == "" {
			continue
		}
		coords = append(coords, resolve.Coord{Group: selectedBy.Group, Artifact: selectedBy.Artifact, Version: selectedBy.Version})
	}
	return coords
}

func parseOutputRecord(line string) (outputRecord, bool) {
	if !strings.HasPrefix(line, recordPrefix) {
		return outputRecord{}, false
	}
	var record outputRecord
	if err := json.Unmarshal([]byte(strings.TrimPrefix(line, recordPrefix)), &record); err != nil {
		return outputRecord{}, false
	}
	return record, true
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

	args := []string{"-I", scriptPath, "-Dorg.gradle.console=plain", "--info"}
	if strings.TrimSpace(opts.GradleUserHome) != "" {
		args = append(args, "--gradle-user-home", opts.GradleUserHome)
	}
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
	args = append(args, ksrcGradleTaskName)

	result := ResolveResult{}
	invocation := formatGradleInvocation(gradleCmd, args, opts.ProjectDir, opts.ProjectPath)
	if opts.Verbose {
		result.Verbose = append(result.Verbose, invocation...)
	}

	stdout, stderr, err := runner.Run(ctx, opts.ProjectDir, gradleCmd, args...)
	if err != nil {
		return ResolveResult{}, &ExecError{
			Err:         err,
			Stdout:      stdout,
			Stderr:      stderr,
			Cmd:         gradleCmd,
			Args:        args,
			ProjectDir:  opts.ProjectDir,
			ProjectPath: opts.ProjectPath,
			Invocation:  invocation,
		}
	}

	seen := make(map[string]struct{})
	seenIncludes := make(map[string]struct{})
	for _, line := range strings.Split(stdout, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		record, ok := parseOutputRecord(line)
		if !ok {
			continue
		}
		switch record.Type {
		case recordTypeSource:
			coord, ok := record.coord()
			if !ok {
				continue
			}
			path := strings.TrimSpace(record.Path)
			if path == "" {
				continue
			}
			key := coord.String() + "|" + path
			if _, exists := seen[key]; exists {
				continue
			}
			seen[key] = struct{}{}
			result.Sources = append(result.Sources, resolve.SourceJar{Coord: coord, Path: path, SelectedBy: record.selectedByCoords()})
		case recordTypeDep:
			coord, ok := record.coord()
			if !ok {
				continue
			}
			result.Deps = append(result.Deps, coord)
		case recordTypeInclude:
			path := strings.TrimSpace(record.Path)
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

func samePath(a string, b string) bool {
	aAbs, errA := filepath.Abs(a)
	bAbs, errB := filepath.Abs(b)
	if errA == nil && errB == nil {
		return aAbs == bAbs
	}
	return filepath.Clean(a) == filepath.Clean(b)
}

func mergeResults(base ResolveResult, extra ResolveResult) ResolveResult {
	if len(extra.Sources) == 0 && len(extra.Deps) == 0 && len(extra.IncludedBuilds) == 0 && len(extra.Warnings) == 0 && len(extra.Verbose) == 0 {
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
	if len(extra.Verbose) > 0 {
		seenVerbose := make(map[string]struct{}, len(base.Verbose))
		for _, line := range base.Verbose {
			seenVerbose[line] = struct{}{}
		}
		for _, line := range extra.Verbose {
			if _, ok := seenVerbose[line]; ok {
				continue
			}
			seenVerbose[line] = struct{}{}
			base.Verbose = append(base.Verbose, line)
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

func formatGradleInvocation(cmd string, args []string, projectDir string, projectPath string) []string {
	lines := make([]string, 0, 4)
	lines = append(lines, fmt.Sprintf("Gradle exec: %s", cmd))
	lines = append(lines, fmt.Sprintf("Gradle dir: %s", projectDir))
	if projectPath == "" {
		lines = append(lines, "Gradle projectPath: (root)")
	} else {
		lines = append(lines, fmt.Sprintf("Gradle projectPath: %s", projectPath))
	}
	if len(args) == 0 {
		lines = append(lines, "Gradle args: (none)")
	} else {
		lines = append(lines, fmt.Sprintf("Gradle args: %s", strings.Join(args, " ")))
	}
	return lines
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
	return writeInitScriptContent(initScriptTemplateVersion, InitScript())
}

func writeInitScriptContent(version string, script string) (string, func(), error) {
	root, err := os.UserCacheDir()
	if err != nil {
		return "", nil, err
	}
	hash := sha256.Sum256([]byte(script))
	dir := filepath.Join(root, "ksrc", "gradle-init")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", nil, err
	}
	path := filepath.Join(dir, "ksrc-init-"+version+"-"+hex.EncodeToString(hash[:8])+".gradle")
	if existing, err := os.ReadFile(path); err == nil && string(existing) == script {
		return path, func() {}, nil
	}
	tmp, err := os.CreateTemp(dir, filepath.Base(path)+".tmp-*")
	if err != nil {
		return "", nil, err
	}
	tmpPath := tmp.Name()
	cleanupTmp := func() {
		_ = os.Remove(tmpPath)
	}
	if _, err := tmp.WriteString(script); err != nil {
		_ = tmp.Close()
		cleanupTmp()
		return "", nil, err
	}
	if err := tmp.Close(); err != nil {
		cleanupTmp()
		return "", nil, err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		if existing, readErr := os.ReadFile(path); readErr == nil && string(existing) == script {
			cleanupTmp()
			return path, func() {}, nil
		}
		cleanupTmp()
		return "", nil, err
	}
	return path, func() {}, nil
}

package mcpserver

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/respawn-app/ksrc/internal/cat"
	"github.com/respawn-app/ksrc/internal/executil"
	"github.com/respawn-app/ksrc/internal/resolve"
	"github.com/respawn-app/ksrc/internal/resolution"
	"github.com/respawn-app/ksrc/internal/search"
)

type toolState struct {
	runner  executil.Runner
	verbose bool
}

func registerTools(server *mcp.Server, state *toolState, tools ToolSet) {
	if tools.Enabled(ToolSearch) {
		mcp.AddTool(server, &mcp.Tool{
			Name:        toolName(ToolSearch),
			Description: "Search dependency sources for a pattern",
		}, state.handleSearch)
	}
	if tools.Enabled(ToolCat) {
		mcp.AddTool(server, &mcp.Tool{
			Name:        toolName(ToolCat),
			Description: "Read a file by file-id",
		}, state.handleCat)
	}
	if tools.Enabled(ToolDeps) {
		mcp.AddTool(server, &mcp.Tool{
			Name:        toolName(ToolDeps),
			Description: "List resolved dependencies and source availability",
		}, state.handleDeps)
	}
	if tools.Enabled(ToolFetch) {
		mcp.AddTool(server, &mcp.Tool{
			Name:        toolName(ToolFetch),
			Description: "Ensure sources for a coordinate exist in Gradle caches",
		}, state.handleFetch)
	}
	if tools.Enabled(ToolResolve) {
		mcp.AddTool(server, &mcp.Tool{
			Name:        toolName(ToolResolve),
			Description: "Resolve dependency sources",
		}, state.handleResolve)
	}
	if tools.Enabled(ToolWhere) {
		mcp.AddTool(server, &mcp.Tool{
			Name:        toolName(ToolWhere),
			Description: "Locate cached source artifact or file",
		}, state.handleWhere)
	}
}

func toolName(name string) string {
	return name
}

type SearchInput struct {
	Query       string   `json:"query"`
	Context     int      `json:"context,omitempty"`
	Group       string   `json:"group,omitempty"`
	Artifact    string   `json:"artifact,omitempty"`
	Version     string   `json:"version,omitempty"`
	Config      []string `json:"config,omitempty"`
	Project     string   `json:"project,omitempty"`
	Subprojects []string `json:"subprojects,omitempty"`
	RgArgs      []string `json:"rgArgs,omitempty"`
	Scope       string   `json:"scope,omitempty"`
	Targets     []string `json:"targets,omitempty"`
}

type CatInput struct {
	FileID string `json:"fileId"`
	Lines  string `json:"lines,omitempty"`
}

type DepsInput struct {
	Project       string   `json:"project,omitempty"`
	Scope         string   `json:"scope,omitempty"`
	Config        []string `json:"config,omitempty"`
	Targets       []string `json:"targets,omitempty"`
	Subprojects   []string `json:"subprojects,omitempty"`
	Buildsrc      *bool    `json:"buildsrc,omitempty"`
	Buildscript   *bool    `json:"buildscript,omitempty"`
	IncludeBuilds *bool    `json:"includeBuilds,omitempty"`
	Group         string   `json:"group,omitempty"`
	Artifact      string   `json:"artifact,omitempty"`
	Version       string   `json:"version,omitempty"`
}

type FetchInput struct {
	Group         string `json:"group"`
	Artifact      string `json:"artifact"`
	Version       string `json:"version"`
	Project       string `json:"project,omitempty"`
	Buildsrc      *bool  `json:"buildsrc,omitempty"`
	Buildscript   *bool  `json:"buildscript,omitempty"`
	IncludeBuilds *bool  `json:"includeBuilds,omitempty"`
}

type ResolveInput struct {
	Project       string   `json:"project,omitempty"`
	Group         string   `json:"group,omitempty"`
	Artifact      string   `json:"artifact,omitempty"`
	Version       string   `json:"version,omitempty"`
	Scope         string   `json:"scope,omitempty"`
	Config        []string `json:"config,omitempty"`
	Targets       []string `json:"targets,omitempty"`
	Subprojects   []string `json:"subprojects,omitempty"`
	Buildsrc      *bool    `json:"buildsrc,omitempty"`
	Buildscript   *bool    `json:"buildscript,omitempty"`
	IncludeBuilds *bool    `json:"includeBuilds,omitempty"`
}

type WhereInput struct {
	PathOrCoord   string   `json:"pathOrCoord"`
	Project       string   `json:"project,omitempty"`
	Group         string   `json:"group,omitempty"`
	Artifact      string   `json:"artifact,omitempty"`
	Version       string   `json:"version,omitempty"`
	Scope         string   `json:"scope,omitempty"`
	Config        []string `json:"config,omitempty"`
	Targets       []string `json:"targets,omitempty"`
	Subprojects   []string `json:"subprojects,omitempty"`
	Buildsrc      *bool    `json:"buildsrc,omitempty"`
	Buildscript   *bool    `json:"buildscript,omitempty"`
	IncludeBuilds *bool    `json:"includeBuilds,omitempty"`
}

func (s *toolState) handleSearch(ctx context.Context, _ *mcp.CallToolRequest, input SearchInput) (*mcp.CallToolResult, struct{}, error) {
	query := strings.TrimSpace(input.Query)
	if query == "" {
		return nil, struct{}{}, fmt.Errorf("query is required")
	}
	req := resolution.Request{
		Project:               withDefaultString(input.Project, "."),
		Group:                 strings.TrimSpace(input.Group),
		Artifact:              strings.TrimSpace(input.Artifact),
		Version:               strings.TrimSpace(input.Version),
		Scope:                 withDefaultString(input.Scope, "compile"),
		Config:                joinCSV(input.Config),
		Targets:               joinCSV(input.Targets),
		Subprojects:           cleanList(input.Subprojects),
		IncludeBuildSrc:       true,
		IncludeBuildscript:    true,
		IncludeIncludedBuilds: true,
		ApplyFilters:          true,
		AllowCacheFallback:    true,
	}
	service := resolution.Service{Runner: s.runner, Verbose: s.verbose}
	result, err := service.ResolveSources(ctx, req)
	if err != nil {
		return nil, struct{}{}, err
	}
	emitDiagnostics(result.Meta, s.verbose)
	if len(result.Sources) == 0 {
		return nil, struct{}{}, noSourcesError(req.Group, req.Artifact, req.Version)
	}
	if _, err := s.runner.LookPath("rg"); err != nil {
		return nil, struct{}{}, fmt.Errorf("rg not found on PATH")
	}

	rgArgs := cleanList(input.RgArgs)
	if input.Context > 0 {
		rgArgs = append(rgArgs, "-C", fmt.Sprintf("%d", input.Context))
	}

	var report func(search.ExecPlan)
	if s.verbose {
		report = func(plan search.ExecPlan) {
			rgLine := fmt.Sprintf("rg: %s %s", plan.Cmd, formatRgArgs(plan))
			fmt.Fprintf(os.Stderr, "VERBOSE: %s\n", rgLine)
			fmt.Fprintf(os.Stderr, "VERBOSE: rg jars: %d (mode=%s)\n", plan.JarCount, plan.Mode)
		}
	}

	matches, err := search.Run(ctx, s.runner, search.Options{
		Pattern: query,
		Jars:    result.Sources,
		RGArgs:  rgArgs,
		WorkDir: req.Project,
		Report:  report,
	})
	if err != nil {
		return nil, struct{}{}, err
	}
	if len(matches) == 0 {
		return textResult("no results"), struct{}{}, nil
	}

	var sb strings.Builder
	for _, m := range matches {
		fmt.Fprintf(&sb, "%s %d:%d:%s\n", m.FileID, m.Line, m.Column, m.Text)
	}
	return textResult(sb.String()), struct{}{}, nil
}

func (s *toolState) handleCat(ctx context.Context, _ *mcp.CallToolRequest, input CatInput) (*mcp.CallToolResult, struct{}, error) {
	fileID := strings.TrimSpace(input.FileID)
	if fileID == "" {
		return nil, struct{}{}, fmt.Errorf("fileId is required")
	}
	coord, inner, err := resolve.ParseFileID(fileID)
	if err != nil {
		return nil, struct{}{}, err
	}
	lr, err := cat.ParseLineRange(input.Lines)
	if err != nil {
		return nil, struct{}{}, err
	}

	req := resolution.Request{
		Project:               ".",
		Group:                 coord.Group,
		Artifact:              coord.Artifact,
		Version:               coord.Version,
		Scope:                 "compile",
		IncludeBuildSrc:       true,
		IncludeBuildscript:    true,
		IncludeIncludedBuilds: true,
		ApplyFilters:          true,
		AllowCacheFallback:    true,
	}
	service := resolution.Service{Runner: s.runner, Verbose: s.verbose}
	result, err := service.ResolveSources(ctx, req)
	if err != nil {
		return nil, struct{}{}, err
	}
	if len(result.Sources) == 0 {
		return nil, struct{}{}, noSourcesError(coord.Group, coord.Artifact, coord.Version)
	}
	jarPath, err := findJarByCoord(result.Sources, coord)
	if err != nil {
		return nil, struct{}{}, err
	}
	data, err := cat.ReadFileFromZip(jarPath, inner, lr)
	if err != nil {
		return nil, struct{}{}, err
	}
	return textResult(string(data)), struct{}{}, nil
}

func (s *toolState) handleDeps(ctx context.Context, _ *mcp.CallToolRequest, input DepsInput) (*mcp.CallToolResult, struct{}, error) {
	req := resolution.Request{
		Project:               withDefaultString(input.Project, "."),
		Scope:                 withDefaultString(input.Scope, "compile"),
		Config:                joinCSV(input.Config),
		Targets:               joinCSV(input.Targets),
		Subprojects:           cleanList(input.Subprojects),
		IncludeBuildSrc:       boolOrDefault(input.Buildsrc, true),
		IncludeBuildscript:    boolOrDefault(input.Buildscript, true),
		IncludeIncludedBuilds: boolOrDefault(input.IncludeBuilds, true),
		ApplyFilters:          false,
		AllowCacheFallback:    false,
	}
	service := resolution.Service{Runner: s.runner, Verbose: s.verbose}
	result, err := service.ResolveSources(ctx, req)
	if err != nil {
		return nil, struct{}{}, err
	}
	emitDiagnostics(result.Meta, s.verbose)

	filteredSources := resolve.FilterSources(result.Sources, "", input.Group, input.Artifact, input.Version)
	filteredDeps := filterCoords(result.Deps, input.Group, input.Artifact, input.Version)

	var sb strings.Builder
	writeDepsOutput(&sb, filteredSources, filteredDeps)
	return textResult(sb.String()), struct{}{}, nil
}

func (s *toolState) handleFetch(ctx context.Context, _ *mcp.CallToolRequest, input FetchInput) (*mcp.CallToolResult, struct{}, error) {
	group := strings.TrimSpace(input.Group)
	artifact := strings.TrimSpace(input.Artifact)
	version := strings.TrimSpace(input.Version)
	if group == "" || artifact == "" || version == "" {
		return nil, struct{}{}, fmt.Errorf("group, artifact, and version are required")
	}
	coord := resolve.Coord{Group: group, Artifact: artifact, Version: version}
	req := resolution.Request{
		Project:               withDefaultString(input.Project, "."),
		Group:                 group,
		Artifact:              artifact,
		Version:               version,
		Scope:                 "compile",
		IncludeBuildSrc:       boolOrDefault(input.Buildsrc, true),
		IncludeBuildscript:    boolOrDefault(input.Buildscript, true),
		IncludeIncludedBuilds: boolOrDefault(input.IncludeBuilds, true),
		Dep:                   coord.String(),
		ApplyFilters:          false,
		AllowCacheFallback:    false,
	}
	service := resolution.Service{Runner: s.runner, Verbose: s.verbose}
	result, err := service.ResolveSources(ctx, req)
	if err != nil {
		return nil, struct{}{}, err
	}
	emitDiagnostics(result.Meta, s.verbose)
	if len(result.Sources) == 0 {
		return nil, struct{}{}, noSourcesError(group, artifact, version)
	}
	var sb strings.Builder
	for _, src := range result.Sources {
		if src.Coord.Group == group && src.Coord.Artifact == artifact && src.Coord.Version == version {
			fmt.Fprintf(&sb, "%s|%s\n", src.Coord.String(), src.Path)
		}
	}
	return textResult(sb.String()), struct{}{}, nil
}

func (s *toolState) handleResolve(ctx context.Context, _ *mcp.CallToolRequest, input ResolveInput) (*mcp.CallToolResult, struct{}, error) {
	req := resolution.Request{
		Project:               withDefaultString(input.Project, "."),
		Group:                 strings.TrimSpace(input.Group),
		Artifact:              strings.TrimSpace(input.Artifact),
		Version:               strings.TrimSpace(input.Version),
		Scope:                 withDefaultString(input.Scope, "compile"),
		Config:                joinCSV(input.Config),
		Targets:               joinCSV(input.Targets),
		Subprojects:           cleanList(input.Subprojects),
		IncludeBuildSrc:       boolOrDefault(input.Buildsrc, true),
		IncludeBuildscript:    boolOrDefault(input.Buildscript, true),
		IncludeIncludedBuilds: boolOrDefault(input.IncludeBuilds, true),
		ApplyFilters:          true,
		AllowCacheFallback:    true,
	}
	if req.Group == "" && req.Artifact == "" && req.Version == "" {
		req.All = true
	}
	service := resolution.Service{Runner: s.runner, Verbose: s.verbose}
	result, err := service.ResolveSources(ctx, req)
	if err != nil {
		return nil, struct{}{}, err
	}
	emitDiagnostics(result.Meta, s.verbose)
	if len(result.Sources) == 0 {
		return nil, struct{}{}, noSourcesError(req.Group, req.Artifact, req.Version)
	}
	var sb strings.Builder
	for _, src := range result.Sources {
		fmt.Fprintf(&sb, "%s|%s\n", src.Coord.String(), src.Path)
	}
	return textResult(sb.String()), struct{}{}, nil
}

func (s *toolState) handleWhere(ctx context.Context, _ *mcp.CallToolRequest, input WhereInput) (*mcp.CallToolResult, struct{}, error) {
	arg := strings.TrimSpace(input.PathOrCoord)
	if arg == "" {
		return nil, struct{}{}, fmt.Errorf("pathOrCoord is required")
	}
	if strings.Contains(arg, "!/") {
		coord, inner, err := resolve.ParseFileID(arg)
		if err != nil {
			return nil, struct{}{}, err
		}
		req := resolution.Request{
			Project:               withDefaultString(input.Project, "."),
			Group:                 coord.Group,
			Artifact:              coord.Artifact,
			Version:               coord.Version,
			Scope:                 withDefaultString(input.Scope, "compile"),
			Config:                joinCSV(input.Config),
			Targets:               joinCSV(input.Targets),
			Subprojects:           cleanList(input.Subprojects),
			IncludeBuildSrc:       boolOrDefault(input.Buildsrc, true),
			IncludeBuildscript:    boolOrDefault(input.Buildscript, true),
			IncludeIncludedBuilds: boolOrDefault(input.IncludeBuilds, true),
			Dep:                   coord.String(),
			ApplyFilters:          true,
			AllowCacheFallback:    true,
		}
		service := resolution.Service{Runner: s.runner, Verbose: s.verbose}
		result, err := service.ResolveSources(ctx, req)
		if err != nil {
			return nil, struct{}{}, err
		}
		if len(result.Sources) == 0 {
			return nil, struct{}{}, noSourcesError(coord.Group, coord.Artifact, coord.Version)
		}
		jarPath, err := findJarByCoord(result.Sources, coord)
		if err != nil {
			return nil, struct{}{}, err
		}
		return textResult(fmt.Sprintf("%s|%s\n", coord.String()+"!/"+inner, jarPath)), struct{}{}, nil
	}
	if coord, err := resolve.ParseCoord(arg); err == nil {
		dep := ""
		if coord.Version != "" {
			dep = coord.String()
		}
		req := resolution.Request{
			Project:               withDefaultString(input.Project, "."),
			Group:                 coord.Group,
			Artifact:              coord.Artifact,
			Version:               coord.Version,
			Scope:                 withDefaultString(input.Scope, "compile"),
			Config:                joinCSV(input.Config),
			Targets:               joinCSV(input.Targets),
			Subprojects:           cleanList(input.Subprojects),
			IncludeBuildSrc:       boolOrDefault(input.Buildsrc, true),
			IncludeBuildscript:    boolOrDefault(input.Buildscript, true),
			IncludeIncludedBuilds: boolOrDefault(input.IncludeBuilds, true),
			Dep:                   dep,
			ApplyFilters:          true,
			AllowCacheFallback:    true,
		}
		service := resolution.Service{Runner: s.runner, Verbose: s.verbose}
		result, err := service.ResolveSources(ctx, req)
		if err != nil {
			return nil, struct{}{}, err
		}
		emitDiagnostics(result.Meta, s.verbose)
		if len(result.Sources) == 0 {
			return nil, struct{}{}, noSourcesError(coord.Group, coord.Artifact, coord.Version)
		}
		jarPath, err := findJarByCoord(result.Sources, coord)
		if err != nil {
			return nil, struct{}{}, err
		}
		return textResult(fmt.Sprintf("%s|%s\n", coord.String(), jarPath)), struct{}{}, nil
	}

	group := strings.TrimSpace(input.Group)
	artifact := strings.TrimSpace(input.Artifact)
	version := strings.TrimSpace(input.Version)
	if group == "" || artifact == "" {
		return nil, struct{}{}, fmt.Errorf("path requires group and artifact filters or a file-id")
	}
	path := strings.TrimPrefix(arg, "/")
	req := resolution.Request{
		Project:               withDefaultString(input.Project, "."),
		Group:                 group,
		Artifact:              artifact,
		Version:               version,
		Scope:                 withDefaultString(input.Scope, "compile"),
		Config:                joinCSV(input.Config),
		Targets:               joinCSV(input.Targets),
		Subprojects:           cleanList(input.Subprojects),
		IncludeBuildSrc:       boolOrDefault(input.Buildsrc, true),
		IncludeBuildscript:    boolOrDefault(input.Buildscript, true),
		IncludeIncludedBuilds: boolOrDefault(input.IncludeBuilds, true),
		ApplyFilters:          true,
		AllowCacheFallback:    true,
	}
	service := resolution.Service{Runner: s.runner, Verbose: s.verbose}
	result, err := service.ResolveSources(ctx, req)
	if err != nil {
		return nil, struct{}{}, err
	}
	emitDiagnostics(result.Meta, s.verbose)
	if len(result.Sources) == 0 {
		return nil, struct{}{}, noSourcesError(group, artifact, version)
	}
	jarPath, coord, inner, err := findFileInJars(result.Sources, path)
	if err != nil {
		return nil, struct{}{}, err
	}
	return textResult(fmt.Sprintf("%s|%s\n", coord.String()+"!/"+inner, jarPath)), struct{}{}, nil
}

func textResult(text string) *mcp.CallToolResult {
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: text}}}
}

func findJarByCoord(sources []resolve.SourceJar, coord resolve.Coord) (string, error) {
	for _, src := range sources {
		if src.Coord.Group == coord.Group && src.Coord.Artifact == coord.Artifact && src.Coord.Version == coord.Version {
			return src.Path, nil
		}
	}
	return "", fmt.Errorf("source jar not found for %s. Try: ksrc fetch %s", coord.String(), coord.String())
}

func findFileInJars(sources []resolve.SourceJar, inner string) (string, resolve.Coord, string, error) {
	inner = strings.TrimPrefix(inner, "/")
	for _, src := range sources {
		if _, err := cat.ReadFileFromZip(src.Path, inner, nil); err == nil {
			return src.Path, src.Coord, inner, nil
		}
	}
	return "", resolve.Coord{}, "", fmt.Errorf("file not found in resolved sources: %s. Try: ksrc search \"<pattern>\" --artifact <artifact>", inner)
}

func formatRgArgs(plan search.ExecPlan) string {
	args := plan.Args
	if plan.JarCount > 0 && len(args) >= plan.JarCount {
		trimmed := append([]string{}, args[:len(args)-plan.JarCount]...)
		trimmed = append(trimmed, fmt.Sprintf("<%d jars>", plan.JarCount))
		return strings.Join(trimmed, " ")
	}
	return strings.Join(args, " ")
}

func writeDepsOutput(sb *strings.Builder, sources []resolve.SourceJar, deps []resolve.Coord) {
	sourceByCoord := make(map[string]string)
	for _, src := range sources {
		sourceByCoord[src.Coord.String()] = src.Path
	}

	seen := make(map[string]struct{})
	for _, dep := range deps {
		key := dep.String()
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		path := sourceByCoord[key]
		sourcesYes := "no"
		if path != "" {
			sourcesYes = "yes"
		}
		fmt.Fprintf(sb, "%s  [sources: %s]  [path: %s]\n", key, sourcesYes, path)
	}

	if len(deps) == 0 {
		for _, src := range sources {
			key := src.Coord.String()
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			fmt.Fprintf(sb, "%s  [sources: yes]  [path: %s]\n", key, src.Path)
		}
	}
}

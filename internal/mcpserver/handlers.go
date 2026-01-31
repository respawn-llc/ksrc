package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/google/jsonschema-go/jsonschema"
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
		server.AddTool(&mcp.Tool{
			Name:        toolName(ToolSearch),
			Description: "Locate Gradle dependency sources. Use this first; avoid browsing .gradle directly. Use the file-id returned here with the cat tool to read the file contents.",
			InputSchema: mustInputSchema[SearchInput](),
		}, state.handleSearch)
	}
	if tools.Enabled(ToolCat) {
		server.AddTool(&mcp.Tool{
			Name:        toolName(ToolCat),
			Description: "Read a file by file-id returned from search. Provide lines as \"A,B\" to limit output; avoid reading large files in full.",
			InputSchema: mustInputSchema[CatInput](),
		}, state.handleCat)
	}
	if tools.Enabled(ToolDeps) {
		server.AddTool(&mcp.Tool{
			Name:        toolName(ToolDeps),
			Description: "List resolved dependencies and whether sources are available for each one. Use to inspect or filter dependency availability.",
			InputSchema: mustInputSchema[DepsInput](),
		}, state.handleDeps)
	}
	if tools.Enabled(ToolFetch) {
		server.AddTool(&mcp.Tool{
			Name:        toolName(ToolFetch),
			Description: "Ensure sources for a coordinate exist in Gradle caches.",
			InputSchema: mustInputSchema[FetchInput](),
		}, state.handleFetch)
	}
	if tools.Enabled(ToolResolve) {
		server.AddTool(&mcp.Tool{
			Name:        toolName(ToolResolve),
			Description: "Resolve dependency sources.",
			InputSchema: mustInputSchema[ResolveInput](),
		}, state.handleResolve)
	}
	if tools.Enabled(ToolWhere) {
		server.AddTool(&mcp.Tool{
			Name:        toolName(ToolWhere),
			Description: "Locate cached source artifact or file.",
			InputSchema: mustInputSchema[WhereInput](),
		}, state.handleWhere)
	}
}

func toolName(name string) string {
	return name
}

type SearchInput struct {
	Query       string   `json:"query" jsonschema:"search pattern"`
	Context     int      `json:"context,omitempty" jsonschema:"rg context lines"`
	Group       string   `json:"group,omitempty" jsonschema:"group filter"`
	Artifact    string   `json:"artifact,omitempty" jsonschema:"artifact filter"`
	Version     string   `json:"version,omitempty" jsonschema:"version filter"`
	Config      []string `json:"config,omitempty" jsonschema:"Gradle config filters"`
	Project     string   `json:"project,omitempty" jsonschema:"project path"`
	Subprojects []string `json:"subprojects,omitempty" jsonschema:"subproject filters"`
	RgArgs      []string `json:"rgArgs,omitempty" jsonschema:"extra rg args"`
	Scope       string   `json:"scope,omitempty" jsonschema:"dependency scope"`
	Targets     []string `json:"targets,omitempty" jsonschema:"target filters"`
}

type CatInput struct {
	FileID string `json:"fileId" jsonschema:"file-id from search"`
	Lines  string `json:"lines,omitempty" jsonschema:"line range A,B or A:B or A-B or A B or A..B or A;B"`
}

type DepsInput struct {
	Project       string   `json:"project,omitempty" jsonschema:"project path"`
	Scope         string   `json:"scope,omitempty" jsonschema:"dependency scope"`
	Config        []string `json:"config,omitempty" jsonschema:"Gradle config filters"`
	Targets       []string `json:"targets,omitempty" jsonschema:"target filters"`
	Subprojects   []string `json:"subprojects,omitempty" jsonschema:"subproject filters"`
	Buildsrc      *bool    `json:"buildsrc,omitempty" jsonschema:"include buildSrc"`
	Buildscript   *bool    `json:"buildscript,omitempty" jsonschema:"include buildscript"`
	IncludeBuilds *bool    `json:"includeBuilds,omitempty" jsonschema:"include builds"`
	Group         string   `json:"group,omitempty" jsonschema:"group filter"`
	Artifact      string   `json:"artifact,omitempty" jsonschema:"artifact filter"`
	Version       string   `json:"version,omitempty" jsonschema:"version filter"`
}

type FetchInput struct {
	Group         string `json:"group" jsonschema:"group"`
	Artifact      string `json:"artifact" jsonschema:"artifact"`
	Version       string `json:"version" jsonschema:"version"`
	Project       string `json:"project,omitempty" jsonschema:"project path"`
	Buildsrc      *bool  `json:"buildsrc,omitempty" jsonschema:"include buildSrc"`
	Buildscript   *bool  `json:"buildscript,omitempty" jsonschema:"include buildscript"`
	IncludeBuilds *bool  `json:"includeBuilds,omitempty" jsonschema:"include builds"`
}

type ResolveInput struct {
	Project       string   `json:"project,omitempty" jsonschema:"project path"`
	Group         string   `json:"group,omitempty" jsonschema:"group filter"`
	Artifact      string   `json:"artifact,omitempty" jsonschema:"artifact filter"`
	Version       string   `json:"version,omitempty" jsonschema:"version filter"`
	Scope         string   `json:"scope,omitempty" jsonschema:"dependency scope"`
	Config        []string `json:"config,omitempty" jsonschema:"Gradle config filters"`
	Targets       []string `json:"targets,omitempty" jsonschema:"target filters"`
	Subprojects   []string `json:"subprojects,omitempty" jsonschema:"subproject filters"`
	Buildsrc      *bool    `json:"buildsrc,omitempty" jsonschema:"include buildSrc"`
	Buildscript   *bool    `json:"buildscript,omitempty" jsonschema:"include buildscript"`
	IncludeBuilds *bool    `json:"includeBuilds,omitempty" jsonschema:"include builds"`
}

type WhereInput struct {
	PathOrCoord   string   `json:"pathOrCoord" jsonschema:"file-id or path/coord"`
	Project       string   `json:"project,omitempty" jsonschema:"project path"`
	Group         string   `json:"group,omitempty" jsonschema:"group filter"`
	Artifact      string   `json:"artifact,omitempty" jsonschema:"artifact filter"`
	Version       string   `json:"version,omitempty" jsonschema:"version filter"`
	Scope         string   `json:"scope,omitempty" jsonschema:"dependency scope"`
	Config        []string `json:"config,omitempty" jsonschema:"Gradle config filters"`
	Targets       []string `json:"targets,omitempty" jsonschema:"target filters"`
	Subprojects   []string `json:"subprojects,omitempty" jsonschema:"subproject filters"`
	Buildsrc      *bool    `json:"buildsrc,omitempty" jsonschema:"include buildSrc"`
	Buildscript   *bool    `json:"buildscript,omitempty" jsonschema:"include buildscript"`
	IncludeBuilds *bool    `json:"includeBuilds,omitempty" jsonschema:"include builds"`
}

func (s *toolState) handleSearch(ctx context.Context, call *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	input, err := decodeInput[SearchInput](call)
	if err != nil {
		return nil, err
	}
	query := strings.TrimSpace(input.Query)
	if query == "" {
		return toolError(fmt.Errorf("query is required")), nil
	}
	resReq := resolution.Request{
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
	result, err := service.ResolveSources(ctx, resReq)
	if err != nil {
		return toolError(err), nil
	}
	emitDiagnostics(result.Meta, s.verbose)
	if len(result.Sources) == 0 {
		return toolError(noSourcesError(resReq.Group, resReq.Artifact, resReq.Version)), nil
	}
	if _, err := s.runner.LookPath("rg"); err != nil {
		return toolError(fmt.Errorf("rg not found on PATH")), nil
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
		WorkDir: resReq.Project,
		Report:  report,
	})
	if err != nil {
		return toolError(err), nil
	}
	if len(matches) == 0 {
		return textResult("no results"), nil
	}

	var sb strings.Builder
	for _, m := range matches {
		fmt.Fprintf(&sb, "%s %d:%d:%s\n", m.FileID, m.Line, m.Column, m.Text)
	}
	return textResult(sb.String()), nil
}

func (s *toolState) handleCat(ctx context.Context, call *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	input, err := decodeInput[CatInput](call)
	if err != nil {
		return nil, err
	}
	fileID := strings.TrimSpace(input.FileID)
	if fileID == "" {
		return toolError(fmt.Errorf("fileId is required")), nil
	}
	coord, inner, err := resolve.ParseFileID(fileID)
	if err != nil {
		return toolError(err), nil
	}
	lr, err := cat.ParseLineRange(input.Lines)
	if err != nil {
		return toolError(err), nil
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
		return toolError(err), nil
	}
	if len(result.Sources) == 0 {
		return toolError(noSourcesError(coord.Group, coord.Artifact, coord.Version)), nil
	}
	jarPath, err := findJarByCoord(result.Sources, coord)
	if err != nil {
		return toolError(err), nil
	}
	data, err := cat.ReadFileFromZip(jarPath, inner, lr)
	if err != nil {
		return toolError(err), nil
	}
	return textResult(string(data)), nil
}

func (s *toolState) handleDeps(ctx context.Context, call *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	input, err := decodeInput[DepsInput](call)
	if err != nil {
		return nil, err
	}
	resReq := resolution.Request{
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
	result, err := service.ResolveSources(ctx, resReq)
	if err != nil {
		return toolError(err), nil
	}
	emitDiagnostics(result.Meta, s.verbose)

	filteredSources := resolve.FilterSources(result.Sources, "", input.Group, input.Artifact, input.Version)
	filteredDeps := filterCoords(result.Deps, input.Group, input.Artifact, input.Version)

	var sb strings.Builder
	writeDepsOutput(&sb, filteredSources, filteredDeps)
	return textResult(sb.String()), nil
}

func (s *toolState) handleFetch(ctx context.Context, call *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	input, err := decodeInput[FetchInput](call)
	if err != nil {
		return nil, err
	}
	group := strings.TrimSpace(input.Group)
	artifact := strings.TrimSpace(input.Artifact)
	version := strings.TrimSpace(input.Version)
	if group == "" || artifact == "" || version == "" {
		return toolError(fmt.Errorf("group, artifact, and version are required")), nil
	}
	coord := resolve.Coord{Group: group, Artifact: artifact, Version: version}
	resReq := resolution.Request{
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
	result, err := service.ResolveSources(ctx, resReq)
	if err != nil {
		return toolError(err), nil
	}
	emitDiagnostics(result.Meta, s.verbose)
	if len(result.Sources) == 0 {
		return toolError(noSourcesError(group, artifact, version)), nil
	}
	var sb strings.Builder
	for _, src := range result.Sources {
		if src.Coord.Group == group && src.Coord.Artifact == artifact && src.Coord.Version == version {
			fmt.Fprintf(&sb, "%s|%s\n", src.Coord.String(), src.Path)
		}
	}
	return textResult(sb.String()), nil
}

func (s *toolState) handleResolve(ctx context.Context, call *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	input, err := decodeInput[ResolveInput](call)
	if err != nil {
		return nil, err
	}
	resReq := resolution.Request{
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
	if resReq.Group == "" && resReq.Artifact == "" && resReq.Version == "" {
		resReq.All = true
	}
	service := resolution.Service{Runner: s.runner, Verbose: s.verbose}
	result, err := service.ResolveSources(ctx, resReq)
	if err != nil {
		return toolError(err), nil
	}
	emitDiagnostics(result.Meta, s.verbose)
	if len(result.Sources) == 0 {
		return toolError(noSourcesError(resReq.Group, resReq.Artifact, resReq.Version)), nil
	}
	var sb strings.Builder
	for _, src := range result.Sources {
		fmt.Fprintf(&sb, "%s|%s\n", src.Coord.String(), src.Path)
	}
	return textResult(sb.String()), nil
}

func (s *toolState) handleWhere(ctx context.Context, call *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	input, err := decodeInput[WhereInput](call)
	if err != nil {
		return nil, err
	}
	arg := strings.TrimSpace(input.PathOrCoord)
	if arg == "" {
		return toolError(fmt.Errorf("pathOrCoord is required")), nil
	}
	if strings.Contains(arg, "!/") {
		coord, inner, err := resolve.ParseFileID(arg)
		if err != nil {
			return toolError(err), nil
		}
		resReq := resolution.Request{
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
		result, err := service.ResolveSources(ctx, resReq)
		if err != nil {
			return toolError(err), nil
		}
		if len(result.Sources) == 0 {
			return toolError(noSourcesError(coord.Group, coord.Artifact, coord.Version)), nil
		}
		jarPath, err := findJarByCoord(result.Sources, coord)
		if err != nil {
			return toolError(err), nil
		}
		return textResult(fmt.Sprintf("%s|%s\n", coord.String()+"!/"+inner, jarPath)), nil
	}
	if coord, err := resolve.ParseCoord(arg); err == nil {
		dep := ""
		if coord.Version != "" {
			dep = coord.String()
		}
		resReq := resolution.Request{
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
		result, err := service.ResolveSources(ctx, resReq)
		if err != nil {
			return toolError(err), nil
		}
		emitDiagnostics(result.Meta, s.verbose)
		if len(result.Sources) == 0 {
			return toolError(noSourcesError(coord.Group, coord.Artifact, coord.Version)), nil
		}
		jarPath, err := findJarByCoord(result.Sources, coord)
		if err != nil {
			return toolError(err), nil
		}
		return textResult(fmt.Sprintf("%s|%s\n", coord.String(), jarPath)), nil
	}

	group := strings.TrimSpace(input.Group)
	artifact := strings.TrimSpace(input.Artifact)
	version := strings.TrimSpace(input.Version)
	if group == "" || artifact == "" {
		return toolError(fmt.Errorf("path requires group and artifact filters or a file-id")), nil
	}
	path := strings.TrimPrefix(arg, "/")
	resReq := resolution.Request{
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
	result, err := service.ResolveSources(ctx, resReq)
	if err != nil {
		return toolError(err), nil
	}
	emitDiagnostics(result.Meta, s.verbose)
	if len(result.Sources) == 0 {
		return toolError(noSourcesError(group, artifact, version)), nil
	}
	jarPath, coord, inner, err := findFileInJars(result.Sources, path)
	if err != nil {
		return toolError(err), nil
	}
	return textResult(fmt.Sprintf("%s|%s\n", coord.String()+"!/"+inner, jarPath)), nil
}

func textResult(text string) *mcp.CallToolResult {
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: text}}}
}

func toolError(err error) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}},
	}
}

func decodeInput[T any](req *mcp.CallToolRequest) (T, error) {
	var input T
	if req == nil || req.Params.Arguments == nil {
		return input, nil
	}
	if err := json.Unmarshal(req.Params.Arguments, &input); err != nil {
		return input, err
	}
	return input, nil
}

func mustInputSchema[T any]() *jsonschema.Schema {
	schema, err := jsonschema.For[T](nil)
	if err != nil {
		panic(err)
	}
	return schema
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

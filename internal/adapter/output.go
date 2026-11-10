package adapter

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/respawn-app/ksrc/internal/resolve"
	"github.com/respawn-app/ksrc/internal/search"
)

func FormatRGArgs(plan search.ExecPlan) string {
	args := plan.Args
	if plan.JarCount > 0 && len(args) >= plan.JarCount {
		trimmed := append([]string{}, args[:len(args)-plan.JarCount]...)
		trimmed = append(trimmed, fmt.Sprintf("<%d jars>", plan.JarCount))
		return strings.Join(trimmed, " ")
	}
	return strings.Join(args, " ")
}

func WriteRGCommandReport(w io.Writer, plan search.ExecPlan) error {
	if w == nil {
		return nil
	}
	if _, err := fmt.Fprintf(w, "VERBOSE: rg: %s %s\n", plan.Cmd, FormatRGArgs(plan)); err != nil {
		return err
	}
	_, err := fmt.Fprintf(w, "VERBOSE: rg jars: %d (mode=%s)\n", plan.JarCount, plan.Mode)
	return err
}

func WriteSearchMatches(w io.Writer, matches []search.Match, showExtractedPath bool) error {
	for _, match := range matches {
		if showExtractedPath {
			if _, err := fmt.Fprintf(w, "%s\t%s\t%d\t%d\t%s\n", match.FileID, strconv.Quote(match.File), match.Line, match.Column, strconv.Quote(match.Text)); err != nil {
				return err
			}
			continue
		}
		if _, err := fmt.Fprintf(w, "%s %d:%d:%s\n", match.FileID, match.Line, match.Column, match.Text); err != nil {
			return err
		}
	}
	return nil
}

func WriteCoordPaths(w io.Writer, sources []resolve.SourceJar) error {
	for _, source := range sources {
		if err := WriteCoordPath(w, source.Coord, source.Path); err != nil {
			return err
		}
	}
	return nil
}

func WriteCoordMatches(w io.Writer, sources []resolve.SourceJar, coord resolve.Coord) error {
	for _, source := range sources {
		if !sameCoord(source.Coord, coord) {
			continue
		}
		if err := WriteCoordPath(w, source.Coord, source.Path); err != nil {
			return err
		}
	}
	return nil
}

func WriteCoordPath(w io.Writer, coord resolve.Coord, path string) error {
	_, err := fmt.Fprintf(w, "%s|%s\n", coord.String(), path)
	return err
}

func WriteFileLocation(w io.Writer, location FileLocation) error {
	return WriteFileIDPath(w, location.FileID, location.Source.Path)
}

func WriteFileIDPath(w io.Writer, fileID string, jarPath string) error {
	_, err := fmt.Fprintf(w, "%s|%s\n", fileID, jarPath)
	return err
}

type depSourceView struct {
	path  string
	full  bool
	stubs bool
}

func (v depSourceView) status() (string, string) {
	switch {
	case v.full:
		return "yes", v.path
	case v.stubs:
		return "stubs", v.path
	default:
		return "no", ""
	}
}

func noteDepSource(views map[string]*depSourceView, order *[]string, key, path string, stub bool) {
	view, ok := views[key]
	if !ok {
		view = &depSourceView{}
		views[key] = view
		*order = append(*order, key)
	}
	if path == "" {
		return
	}
	if stub {
		view.stubs = true
		if view.path == "" {
			view.path = path
		}
		return
	}
	view.full = true
	view.path = path
}

func WriteDepsOutput(w io.Writer, sources []resolve.SourceJar, deps []resolve.Coord) error {
	views := make(map[string]*depSourceView)
	order := make([]string, 0, len(sources))
	for _, source := range sources {
		noteDepSource(views, &order, source.Coord.String(), source.Path, source.Stub)
	}

	seen := make(map[string]struct{})
	write := func(key string) error {
		if _, ok := seen[key]; ok {
			return nil
		}
		seen[key] = struct{}{}
		status, path := "no", ""
		if view, ok := views[key]; ok {
			status, path = view.status()
		}
		_, err := fmt.Fprintf(w, "%s  [sources: %s]  [path: %s]\n", key, status, path)
		return err
	}
	for _, dep := range deps {
		if err := write(dep.String()); err != nil {
			return err
		}
	}
	if len(deps) != 0 {
		return nil
	}
	for _, key := range order {
		if err := write(key); err != nil {
			return err
		}
	}
	return nil
}

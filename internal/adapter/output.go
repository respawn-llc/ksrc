package adapter

import (
	"fmt"
	"io"
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

func WriteSearchMatches(w io.Writer, matches []search.Match, showExtractedPath bool) error {
	for _, match := range matches {
		if showExtractedPath {
			if _, err := fmt.Fprintf(w, "%s %s:%d:%d:%s\n", match.FileID, match.File, match.Line, match.Column, match.Text); err != nil {
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

func WriteDepsOutput(w io.Writer, sources []resolve.SourceJar, deps []resolve.Coord) error {
	sourceByCoord := make(map[string]string)
	for _, source := range sources {
		sourceByCoord[source.Coord.String()] = source.Path
	}

	seen := make(map[string]struct{})
	for _, dep := range deps {
		key := dep.String()
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		path := sourceByCoord[key]
		hasSources := "no"
		if path != "" {
			hasSources = "yes"
		}
		if _, err := fmt.Fprintf(w, "%s  [sources: %s]  [path: %s]\n", key, hasSources, path); err != nil {
			return err
		}
	}

	if len(deps) != 0 {
		return nil
	}
	for _, source := range sources {
		key := source.Coord.String()
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		if _, err := fmt.Fprintf(w, "%s  [sources: yes]  [path: %s]\n", key, source.Path); err != nil {
			return err
		}
	}
	return nil
}

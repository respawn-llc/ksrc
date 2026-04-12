package adapter

import (
	"fmt"
	"strings"

	"github.com/respawn-app/ksrc/internal/resolve"
)

type FileLocation struct {
	Source    resolve.SourceJar
	InnerPath string
	FileID    string
}

func FindJarByCoord(sources []resolve.SourceJar, coord resolve.Coord, missingHint string) (string, error) {
	for _, source := range sources {
		if sameCoord(source.Coord, coord) {
			return source.Path, nil
		}
	}
	missingHint = strings.TrimSpace(missingHint)
	if missingHint == "" {
		missingHint = NoSourcesHintForCoord(coord)
	}
	return "", fmt.Errorf("source jar not found for %s. %s", coord.String(), missingHint)
}

func FindFile(sources []resolve.SourceJar, path string) (FileLocation, bool) {
	source, inner, ok := resolve.FindFileInSources(sources, path)
	if !ok {
		return FileLocation{}, false
	}
	return FileLocation{
		Source:    source,
		InnerPath: inner,
		FileID:    resolve.FormatFileID(source.Coord, inner),
	}, true
}

func sameCoord(left, right resolve.Coord) bool {
	return left.Group == right.Group && left.Artifact == right.Artifact && left.Version == right.Version
}

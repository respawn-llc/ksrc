package adapter

import (
	"errors"
	"fmt"
	"strings"

	"github.com/respawn-app/ksrc/internal/resolve"
)

type FileLocation struct {
	Source    resolve.SourceJar
	InnerPath string
	FileID    string
}

func ResolveCoordSource(sources []resolve.SourceJar, coord resolve.Coord, missingHint string) (resolve.SourceJar, error) {
	for _, source := range sources {
		if sameCoord(source.Coord, coord) {
			return source, nil
		}
	}
	missingHint = strings.TrimSpace(missingHint)
	if missingHint == "" {
		missingHint = NoSourcesHintForCoord(coord)
	}
	return resolve.SourceJar{}, fmt.Errorf("source jar not found for %s. %s", coord.String(), missingHint)
}

func FindJarByCoord(sources []resolve.SourceJar, coord resolve.Coord, missingHint string) (string, error) {
	source, err := ResolveCoordSource(sources, coord, missingHint)
	if err != nil {
		return "", err
	}
	return source.Path, nil
}

func ResolveFileIDLocation(sources []resolve.SourceJar, fileID string, missingHint string) (FileLocation, error) {
	coord, inner, err := resolve.ParseFileID(fileID)
	if err != nil {
		return FileLocation{}, err
	}
	source, err := ResolveCoordSource(sources, coord, missingHint)
	if err != nil {
		return FileLocation{}, err
	}
	return FileLocation{
		Source:    source,
		InnerPath: inner,
		FileID:    resolve.FormatFileID(source.Coord, inner),
	}, nil
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

func ResolvePathLocation(sources []resolve.SourceJar, path string, notFoundHint string) (FileLocation, error) {
	location, ok := FindFile(sources, path)
	if ok {
		return location, nil
	}
	trimmedPath := strings.TrimPrefix(path, "/")
	message := fmt.Sprintf("file not found in resolved sources: %s", trimmedPath)
	notFoundHint = strings.TrimSpace(notFoundHint)
	if notFoundHint != "" {
		message += ". " + notFoundHint
	}
	return FileLocation{}, errors.New(message)
}

func sameCoord(left, right resolve.Coord) bool {
	return left.Group == right.Group && left.Artifact == right.Artifact && left.Version == right.Version
}

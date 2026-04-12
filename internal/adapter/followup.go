package adapter

import (
	"os"

	"github.com/respawn-app/ksrc/internal/fileidcache"
	"github.com/respawn-app/ksrc/internal/resolve"
	"github.com/respawn-app/ksrc/internal/search"
)

func TrackSearchMatches(matches []search.Match) error {
	seen := make(map[string]struct{}, len(matches))
	for _, match := range matches {
		if match.FileID == "" || match.JarPath == "" {
			continue
		}
		key := match.FileID + "|" + match.JarPath
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		if err := fileidcache.Register(match.FileID, match.JarPath); err != nil {
			return err
		}
	}
	return nil
}

func TryTrackSearchMatches(matches []search.Match) {
	_ = TrackSearchMatches(matches)
}

func TrackFileIDPath(fileID string, jarPath string) error {
	return fileidcache.Register(fileID, jarPath)
}

func TrackFileLocation(location FileLocation) error {
	return TrackFileIDPath(location.FileID, location.Source.Path)
}

func TryTrackFileLocation(location FileLocation) {
	_ = TrackFileLocation(location)
}

func FindFollowupFileIDLocation(fileID string) (FileLocation, bool, error) {
	coord, inner, err := resolve.ParseFileID(fileID)
	if err != nil {
		return FileLocation{}, false, err
	}
	jarPath, found, err := fileidcache.Lookup(fileID)
	if found {
		return FileLocation{
			Source:    resolve.SourceJar{Coord: coord, Path: jarPath},
			InnerPath: inner,
			FileID:    resolve.FormatFileID(coord, inner),
		}, true, nil
	}
	if err != nil {
		// File-id cache is best-effort. Read failures should degrade to cache miss so
		// follow-up commands can still resolve from Gradle caches.
		err = nil
	}
	source, found, err := FindCachedCoordSource(coord)
	if err != nil || !found {
		return FileLocation{}, found, err
	}
	return FileLocation{
		Source:    source,
		InnerPath: inner,
		FileID:    resolve.FormatFileID(source.Coord, inner),
	}, true, nil
}

func FindCachedCoordSource(coord resolve.Coord) (resolve.SourceJar, bool, error) {
	sources, err := resolve.FindCachedSources(coord.Group, coord.Artifact, coord.Version)
	if err != nil {
		if resolve.IsCachedSourcesNotFound(err) || os.IsNotExist(err) {
			return resolve.SourceJar{}, false, nil
		}
		return resolve.SourceJar{}, false, err
	}
	source, err := ResolveCoordSource(sources, coord, "")
	if err != nil {
		return resolve.SourceJar{}, false, err
	}
	return source, true, nil
}

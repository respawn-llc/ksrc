package resolve

import "github.com/respawn-app/ksrc/internal/cat"

// FindFileInSources locates an inner path in resolved source jars.
func FindFileInSources(sources []SourceJar, inner string) (SourceJar, string, bool) {
	inner = normalizeFileIDPath(inner)
	if inner == "" {
		return SourceJar{}, "", false
	}
	for _, src := range sources {
		if _, err := cat.ReadFileFromZip(src.Path, inner, nil); err == nil {
			return src, inner, true
		}
	}
	return SourceJar{}, inner, false
}

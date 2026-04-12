package resolve

import "github.com/respawn-app/ksrc/internal/cat"

// FindFileInSources locates an inner path in resolved source jars.
func FindFileInSources(sources []SourceJar, inner string) (SourceJar, string, bool) {
	inner = normalizeFileIDPath(inner)
	if inner == "" {
		return SourceJar{}, "", false
	}
	for _, src := range sources {
		if ok, err := cat.HasFileInZip(src.Path, inner); err == nil && ok {
			return src, inner, true
		}
	}
	return SourceJar{}, inner, false
}

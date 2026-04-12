package resolve

// FilterSources applies module/group/artifact/version filters.
func FilterSources(sources []SourceJar, module, group, artifact, version string) []SourceJar {
	selector := NewSelector(module, group, artifact, version)
	out := make([]SourceJar, 0, len(sources))
	for _, source := range sources {
		if selector.MatchCoord(source.Coord) {
			out = append(out, source)
		}
	}
	return out
}

// FilterCoords applies module/group/artifact/version filters to coordinates.
func FilterCoords(coords []Coord, module, group, artifact, version string) []Coord {
	selector := NewSelector(module, group, artifact, version)
	out := make([]Coord, 0, len(coords))
	for _, coord := range coords {
		if selector.MatchCoord(coord) {
			out = append(out, coord)
		}
	}
	return out
}

package resolve

import (
	"path"
	"strings"
)

type coordField string

const (
	coordFieldGroup    coordField = "group"
	coordFieldArtifact coordField = "artifact"
)

type moduleCandidateDefinition struct {
	fields        []coordField
	normalizeDots bool
}

var moduleLooseCandidateDefinitions = []moduleCandidateDefinition{
	{fields: []coordField{coordFieldGroup}},
	{fields: []coordField{coordFieldArtifact}},
	{fields: []coordField{coordFieldGroup, coordFieldArtifact}},
	{fields: []coordField{coordFieldArtifact}, normalizeDots: true},
	{fields: []coordField{coordFieldGroup, coordFieldArtifact}, normalizeDots: true},
}

type patternMatcher struct {
	raw      string
	patterns []string
}

func newPatternMatcher(raw string) patternMatcher {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return patternMatcher{}
	}
	patterns := make([]string, 0, strings.Count(raw, ",")+1)
	for _, pattern := range strings.Split(raw, ",") {
		pattern = strings.TrimSpace(pattern)
		if pattern == "" {
			continue
		}
		patterns = append(patterns, pattern)
	}
	return patternMatcher{raw: raw, patterns: patterns}
}

func (m patternMatcher) Match(value string) bool {
	if len(m.patterns) == 0 {
		return true
	}
	for _, pattern := range m.patterns {
		ok, err := path.Match(pattern, value)
		if err == nil && ok {
			return true
		}
		if strings.EqualFold(pattern, value) {
			return true
		}
	}
	return false
}

type moduleMatcher struct {
	raw          string
	qualified    bool
	group        patternMatcher
	artifact     patternMatcher
	version      patternMatcher
	matchVersion bool
	coord        Coord
	hasCoord     bool
	loose        patternMatcher
}

func newModuleMatcher(raw string) moduleMatcher {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return moduleMatcher{}
	}
	if strings.Contains(raw, ":") {
		parts := strings.Split(raw, ":")
		if len(parts) >= 2 {
			matcher := moduleMatcher{
				raw:          raw,
				qualified:    true,
				group:        newPatternMatcher(parts[0]),
				artifact:     newPatternMatcher(parts[1]),
				matchVersion: len(parts) >= 3 && parts[2] != "",
			}
			if len(parts) >= 3 {
				matcher.version = newPatternMatcher(parts[2])
				matcher.coord.Version = parts[2]
			}
			matcher.coord.Group = parts[0]
			matcher.coord.Artifact = parts[1]
			matcher.hasCoord = matcher.coord.Group != "" && matcher.coord.Artifact != ""
			return matcher
		}
	}
	return moduleMatcher{raw: raw, loose: newPatternMatcher(raw)}
}

func (m moduleMatcher) Match(coord Coord) bool {
	if m.raw == "" {
		return true
	}
	if m.qualified {
		if !m.group.Match(coord.Group) {
			return false
		}
		if !m.artifact.Match(coord.Artifact) {
			return false
		}
		if m.matchVersion {
			return m.version.Match(coord.Version)
		}
		return true
	}
	for _, candidate := range moduleLooseCandidates(coord) {
		if m.loose.Match(candidate) {
			return true
		}
		if strings.Contains(candidate, m.raw) {
			return true
		}
	}
	return false
}

type Selector struct {
	module   moduleMatcher
	group    patternMatcher
	artifact patternMatcher
	version  patternMatcher
}

func NewSelector(module, group, artifact, version string) Selector {
	return Selector{
		module:   newModuleMatcher(module),
		group:    newPatternMatcher(group),
		artifact: newPatternMatcher(artifact),
		version:  newPatternMatcher(version),
	}
}

func (s Selector) MatchCoord(coord Coord) bool {
	return s.module.Match(coord) &&
		s.group.Match(coord.Group) &&
		s.artifact.Match(coord.Artifact) &&
		s.version.Match(coord.Version)
}

func (s Selector) Coord() (Coord, bool) {
	if len(s.group.patterns) > 0 && len(s.artifact.patterns) > 0 {
		coord := Coord{Group: s.group.raw, Artifact: s.artifact.raw, Version: s.version.raw}
		return coord, true
	}
	if !s.module.hasCoord {
		return Coord{}, false
	}
	coord := s.module.coord
	if s.version.raw != "" {
		coord.Version = s.version.raw
	}
	return coord, true
}

// SelectorToCoord extracts group/artifact[/version] from filters when available.
func SelectorToCoord(module, group, artifact, version string) (Coord, bool) {
	return NewSelector(module, group, artifact, version).Coord()
}

func MatchAny(patterns string, value string) bool {
	return newPatternMatcher(patterns).Match(value)
}

func MatchModule(selector string, coord Coord) bool {
	return newModuleMatcher(selector).Match(coord)
}

func moduleLooseCandidates(coord Coord) []string {
	values := make([]string, 0, len(moduleLooseCandidateDefinitions))
	for _, candidate := range moduleLooseCandidateDefinitions {
		values = append(values, candidate.value(coord))
	}
	return values
}

func (c moduleCandidateDefinition) value(coord Coord) string {
	parts := make([]string, 0, len(c.fields))
	for _, field := range c.fields {
		parts = append(parts, coordFieldValue(field, coord))
	}
	value := strings.Join(parts, ":")
	if c.normalizeDots {
		return normalizeSelectorCandidate(value)
	}
	return value
}

func coordFieldValue(field coordField, coord Coord) string {
	switch field {
	case coordFieldGroup:
		return coord.Group
	case coordFieldArtifact:
		return coord.Artifact
	default:
		return ""
	}
}

func normalizeSelectorCandidate(value string) string {
	return strings.ReplaceAll(value, "-", ".")
}

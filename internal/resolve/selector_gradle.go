package resolve

import (
	"fmt"
	"strings"
)

func GradleSelectorHelpers() string {
	looseCandidates := make([]string, 0, len(moduleLooseCandidateDefinitions))
	for _, candidate := range moduleLooseCandidateDefinitions {
		looseCandidates = append(looseCandidates, "        "+candidate.gradleExpression())
	}
	return fmt.Sprintf(`
def splitCsv = { String value ->
    if (value == null) return [] as Set
    value.split(',').collect { it.trim() }.findAll { it }.toSet()
}

def matchesGlobPattern = { String patterns, String value ->
    if (patterns == null || patterns.trim().isEmpty()) return true
    def candidate = value == null ? '' : value
    patterns.split(',').collect { it.trim() }.findAll { it }.any { pattern ->
        if (pattern.equalsIgnoreCase(candidate)) return true
        try {
            return java.nio.file.FileSystems.default
                .getPathMatcher('glob:' + pattern)
                .matches(java.nio.file.Paths.get(candidate))
        } catch (Throwable ignored) {
            return false
        }
    }
}

def normalizeSelectorCandidate = { String value ->
    def candidate = value == null ? '' : value
    return candidate.replace('-', '.')
}

def moduleSelectorCandidates = { String group, String artifact ->
    return [
%s
    ]
}

def matchesModuleSelector = { String selector, String group, String artifact, String version ->
    selector = selector == null ? '' : selector.trim()
    if (selector.isEmpty()) return true
    if (selector.contains(':')) {
        def parts = selector.split(':', -1)
        if (parts.length >= 2) {
            if (!matchesGlobPattern(parts[0], group)) return false
            if (!matchesGlobPattern(parts[1], artifact)) return false
            if (parts.length >= 3 && parts[2]) return matchesGlobPattern(parts[2], version)
            return true
        }
    }
    return moduleSelectorCandidates(group, artifact).any { candidate ->
        matchesGlobPattern(selector, candidate) || candidate.contains(selector)
    }
}

def matchesCoordinateSelectors = { String moduleSelector, String groupSelector, String artifactSelector, String versionSelector, String group, String artifact, String version ->
    return matchesModuleSelector(moduleSelector, group, artifact, version) &&
        matchesGlobPattern(groupSelector, group) &&
        matchesGlobPattern(artifactSelector, artifact) &&
        matchesGlobPattern(versionSelector, version)
}
`, strings.Join(looseCandidates, ",\n"))
}

func (c moduleCandidateDefinition) gradleExpression() string {
	expr := c.gradleJoinExpression()
	if c.normalizeDots {
		return fmt.Sprintf("normalizeSelectorCandidate(%s)", expr)
	}
	return expr
}

func (c moduleCandidateDefinition) gradleJoinExpression() string {
	parts := make([]string, 0, len(c.fields))
	for _, field := range c.fields {
		parts = append(parts, string(field))
	}
	if len(parts) == 1 {
		return parts[0]
	}
	return strings.Join(parts, " + ':' + ")
}

package mcpserver

import (
	"fmt"
	"strings"
)

const (
	ToolSearch  = "search"
	ToolCat     = "cat"
	ToolDeps    = "deps"
	ToolFetch   = "fetch"
	ToolResolve = "resolve"
	ToolWhere   = "where"
)

var allToolNames = []string{
	ToolSearch,
	ToolCat,
	ToolDeps,
	ToolFetch,
	ToolResolve,
	ToolWhere,
}

var defaultToolNames = []string{
	ToolSearch,
	ToolCat,
	ToolDeps,
}

type ToolSet map[string]bool

func DefaultTools() ToolSet {
	return toolSetFromList(defaultToolNames)
}

func AllTools() ToolSet {
	return toolSetFromList(allToolNames)
}

func ParseTools(value string) (ToolSet, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return DefaultTools(), nil
	}
	if value == "all" {
		return AllTools(), nil
	}
	parts := strings.Split(value, ",")
	items := make([]string, 0, len(parts))
	for _, part := range parts {
		name := strings.TrimSpace(part)
		if name == "" {
			continue
		}
		items = append(items, name)
	}
	if len(items) == 0 {
		return DefaultTools(), nil
	}
	set := make(ToolSet)
	for _, name := range items {
		if !isKnownTool(name) {
			return nil, fmt.Errorf("unknown tool: %s", name)
		}
		set[name] = true
	}
	return set, nil
}

func (t ToolSet) Enabled(name string) bool {
	return t[name]
}

func toolSetFromList(names []string) ToolSet {
	set := make(ToolSet)
	for _, name := range names {
		set[name] = true
	}
	return set
}

func isKnownTool(name string) bool {
	for _, tool := range allToolNames {
		if tool == name {
			return true
		}
	}
	return false
}

func KnownTools() []string {
	return append([]string{}, allToolNames...)
}

func DefaultToolNames() []string {
	return append([]string{}, defaultToolNames...)
}

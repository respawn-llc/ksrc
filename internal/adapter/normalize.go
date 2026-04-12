package adapter

import "strings"

func SplitCSV(value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return CleanList(strings.Split(value, ","))
}

func JoinCSV(values []string) string {
	cleaned := CleanList(values)
	if len(cleaned) == 0 {
		return ""
	}
	return strings.Join(cleaned, ",")
}

func CleanList(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		out = append(out, value)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func DefaultString(value string, def string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return def
	}
	return value
}

func BoolOrDefault(value *bool, def bool) bool {
	if value == nil {
		return def
	}
	return *value
}

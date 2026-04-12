package resolve

import (
	"math/big"
	"strconv"
	"strings"
)

var (
	knownVersionQualifiers = []string{"alpha", "beta", "milestone", "rc", "snapshot", "", "sp"}
	versionQualifierRanks  = func() map[string]int {
		ranks := make(map[string]int, len(knownVersionQualifiers))
		for idx, qualifier := range knownVersionQualifiers {
			ranks[qualifier] = idx
		}
		return ranks
	}()
	releaseVersionQualifierAliases = map[string]struct{}{
		"ga":      {},
		"final":   {},
		"release": {},
	}
	versionQualifierAliases = map[string]string{
		"cr": "rc",
	}
	releaseVersionQualifierRank = strconv.Itoa(versionQualifierRanks[""])
)

type versionItemKind uint8

const (
	versionItemKindNumber versionItemKind = iota
	versionItemKindString
	versionItemKindList
	versionItemKindCombination
)

type versionItem interface {
	kind() versionItemKind
	compare(other versionItem) int
	isNull() bool
}

type versionNumber struct {
	value *big.Int
}

func newVersionNumber(raw string) *versionNumber {
	value, ok := new(big.Int).SetString(stripVersionLeadingZeros(raw), 10)
	if !ok {
		panic("invalid numeric version token")
	}
	return &versionNumber{value: value}
}

func (n *versionNumber) kind() versionItemKind {
	return versionItemKindNumber
}

func (n *versionNumber) compare(other versionItem) int {
	if other == nil {
		if n.isNull() {
			return 0
		}
		return 1
	}

	switch candidate := other.(type) {
	case *versionNumber:
		return n.value.Cmp(candidate.value)
	case *versionString, *versionCombination, *versionList:
		return 1
	default:
		panic("unexpected version item")
	}
}

func (n *versionNumber) isNull() bool {
	return n.value.Sign() == 0
}

type versionString struct {
	value string
}

func newVersionString(raw string, followedByDigit bool) *versionString {
	value := raw
	if followedByDigit && len(value) == 1 {
		switch value[0] {
		case 'a':
			value = "alpha"
		case 'b':
			value = "beta"
		case 'm':
			value = "milestone"
		}
	}
	if alias, ok := versionQualifierAliases[value]; ok {
		value = alias
	}
	return &versionString{value: value}
}

func (s *versionString) kind() versionItemKind {
	return versionItemKindString
}

func (s *versionString) compare(other versionItem) int {
	if other == nil {
		return strings.Compare(comparableVersionQualifier(s.value), releaseVersionQualifierRank)
	}

	switch candidate := other.(type) {
	case *versionNumber:
		return -1
	case *versionString:
		return strings.Compare(comparableVersionQualifier(s.value), comparableVersionQualifier(candidate.value))
	case *versionCombination:
		if result := s.compare(candidate.stringPart); result != 0 {
			return result
		}
		return -1
	case *versionList:
		return -1
	default:
		panic("unexpected version item")
	}
}

func (s *versionString) isNull() bool {
	return s.value == ""
}

type versionCombination struct {
	stringPart *versionString
	digitPart  versionItem
}

func newVersionCombination(raw string) *versionCombination {
	collapsed := strings.ReplaceAll(raw, "-", "")
	index := strings.IndexFunc(collapsed, isASCIIDigitRune)
	if index <= 0 {
		panic("invalid combined version token")
	}
	return &versionCombination{
		stringPart: newVersionString(collapsed[:index], true),
		digitPart:  parseVersionItem(false, true, collapsed[index:]),
	}
}

func (c *versionCombination) kind() versionItemKind {
	return versionItemKindCombination
}

func (c *versionCombination) compare(other versionItem) int {
	if other == nil {
		return c.stringPart.compare(nil)
	}

	switch candidate := other.(type) {
	case *versionNumber:
		return -1
	case *versionString:
		if result := c.stringPart.compare(candidate); result != 0 {
			return result
		}
		return 1
	case *versionCombination:
		if result := c.stringPart.compare(candidate.stringPart); result != 0 {
			return result
		}
		return c.digitPart.compare(candidate.digitPart)
	case *versionList:
		return -1
	default:
		panic("unexpected version item")
	}
}

func (c *versionCombination) isNull() bool {
	return false
}

type versionList struct {
	items []versionItem
}

func (l *versionList) kind() versionItemKind {
	return versionItemKindList
}

func (l *versionList) compare(other versionItem) int {
	if other == nil {
		for _, item := range l.items {
			if result := item.compare(nil); result != 0 {
				return result
			}
		}
		return 0
	}

	switch candidate := other.(type) {
	case *versionNumber:
		return -1
	case *versionString:
		return 1
	case *versionCombination:
		return 1
	case *versionList:
		max := len(l.items)
		if len(candidate.items) > max {
			max = len(candidate.items)
		}
		for idx := 0; idx < max; idx++ {
			var left versionItem
			if idx < len(l.items) {
				left = l.items[idx]
			}
			var right versionItem
			if idx < len(candidate.items) {
				right = candidate.items[idx]
			}

			var result int
			switch {
			case left == nil && right == nil:
				result = 0
			case left == nil:
				result = -right.compare(nil)
			default:
				result = left.compare(right)
			}
			if result != 0 {
				return result
			}
		}
		return 0
	default:
		panic("unexpected version item")
	}
}

func (l *versionList) isNull() bool {
	return len(l.items) == 0
}

func (l *versionList) normalize() {
	for idx := len(l.items) - 1; idx >= 0; idx-- {
		item := l.items[idx]
		if !item.isNull() {
			continue
		}
		if idx == len(l.items)-1 {
			l.items = append(l.items[:idx], l.items[idx+1:]...)
			continue
		}

		next := l.items[idx+1]
		if next.kind() == versionItemKindString {
			l.items = append(l.items[:idx], l.items[idx+1:]...)
			continue
		}

		nextList, ok := next.(*versionList)
		if !ok || len(nextList.items) == 0 {
			continue
		}
		switch nextList.items[0].kind() {
		case versionItemKindString, versionItemKindCombination:
			l.items = append(l.items[:idx], l.items[idx+1:]...)
		}
	}
}

// CompareVersion compares versions using Maven-style qualifier semantics.
// `.` and `_` normalize to segment separators for cache-version parity, while `+` stays a literal qualifier boundary.
// Returns -1 if a < b, 0 if equal, 1 if a > b.
func CompareVersion(a, b string) int {
	return parseVersion(a).compare(parseVersion(b))
}

func parseVersion(raw string) *versionList {
	version := strings.ToLower(strings.TrimSpace(raw))
	root := &versionList{}
	current := root
	stack := []*versionList{root}
	start := 0
	isDigit := false
	isCombination := false

	for idx := 0; idx < len(version); idx++ {
		ch := version[idx]
		switch {
		case ch == '.' || ch == '_':
			if idx == start {
				current.items = append(current.items, newVersionNumber("0"))
			} else {
				current.items = append(current.items, parseVersionItem(isCombination, isDigit, version[start:idx]))
			}
			start = idx + 1
			isCombination = false
		case ch == '-':
			if idx == start {
				current.items = append(current.items, newVersionNumber("0"))
			} else {
				if !isDigit && idx < len(version)-1 && isASCIIDigit(version[idx+1]) {
					isCombination = true
					continue
				}
				current.items = append(current.items, parseVersionItem(isCombination, isDigit, version[start:idx]))
			}
			start = idx + 1
			if len(current.items) > 0 {
				next := &versionList{}
				current.items = append(current.items, next)
				stack = append(stack, next)
				current = next
			}
			isCombination = false
		case isASCIIDigit(ch):
			if !isDigit && idx > start {
				isCombination = true
				if len(current.items) > 0 {
					next := &versionList{}
					current.items = append(current.items, next)
					stack = append(stack, next)
					current = next
				}
			}
			isDigit = true
		default:
			if isDigit && idx > start {
				current.items = append(current.items, parseVersionItem(isCombination, true, version[start:idx]))
				start = idx
				next := &versionList{}
				current.items = append(current.items, next)
				stack = append(stack, next)
				current = next
				isCombination = false
			}
			isDigit = false
		}
	}

	if len(version) > start {
		if !isDigit && len(current.items) > 0 {
			next := &versionList{}
			current.items = append(current.items, next)
			stack = append(stack, next)
			current = next
		}
		current.items = append(current.items, parseVersionItem(isCombination, isDigit, version[start:]))
	}

	for idx := len(stack) - 1; idx >= 0; idx-- {
		stack[idx].normalize()
	}

	return root
}

func parseVersionItem(isCombination bool, isDigit bool, raw string) versionItem {
	if isCombination {
		return newVersionCombination(raw)
	}
	if isDigit {
		return newVersionNumber(raw)
	}
	return newVersionString(raw, false)
}

func comparableVersionQualifier(qualifier string) string {
	if _, ok := releaseVersionQualifierAliases[qualifier]; ok {
		return releaseVersionQualifierRank
	}
	if rank, ok := versionQualifierRanks[qualifier]; ok {
		return strconv.Itoa(rank)
	}
	return strconv.Itoa(len(knownVersionQualifiers)) + "-" + qualifier
}

func stripVersionLeadingZeros(raw string) string {
	if raw == "" {
		return "0"
	}
	for idx := 0; idx < len(raw); idx++ {
		if raw[idx] != '0' {
			return raw[idx:]
		}
	}
	return "0"
}

func isASCIIDigit(value byte) bool {
	return value >= '0' && value <= '9'
}

func isASCIIDigitRune(value rune) bool {
	return value >= '0' && value <= '9'
}

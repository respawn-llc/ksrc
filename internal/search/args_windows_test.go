package search

import (
	"reflect"
	"strings"
	"testing"
)

func TestWindowsArgLen(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want int
	}{
		{"", 2},
		{"abc", 3},
		{"a b", 5},
		{`C:\Program Files\x`, 20},
		{`a"b`, 4},
		{`a\"b`, 6},
		{`C:\x\`, 5},
		{"😀", 2},
		{"😀 😀", 7},
	} {
		if got := windowsArgLen(tc.in); got != tc.want {
			t.Errorf("windowsArgLen(%q) = %d, want %d", tc.in, got, tc.want)
		}
	}
}

func TestChunkByBudget(t *testing.T) {
	for _, dirs := range [][]string{
		{"dir-a", "dir-b", "dir-c"},
		{strings.Repeat("x", 50), strings.Repeat("x", 50)},
	} {
		base := []string{"--json", "--", "pattern"}
		chunks := chunkByBudget(windowsArgsLen(base), dirs, 100)
		var flattened []string
		for _, chunk := range chunks {
			if len(chunk) == 0 || windowsArgsLen(append(append([]string{}, base...), chunk...)) > 100 {
				t.Fatalf("invalid chunk %v", chunk)
			}
			flattened = append(flattened, chunk...)
		}
		if !reflect.DeepEqual(flattened, dirs) {
			t.Fatalf("directories changed: %v", flattened)
		}
	}
}

func TestChunkSearchDirsKeepsEscapedCommandLineWithinWindowsLimit(t *testing.T) {
	base := []string{"--json", "--color=never", "--", `fun (bar|"baz qux")\(`}
	dirs := make([]string, 600)
	for i := range dirs {
		dirs[i] = `C:\Users\User Name\` + strings.Repeat("😀", 40) + `\`
	}
	chunks := chunkSearchDirs(base, dirs)
	if len(chunks) < 2 {
		t.Fatalf("expected multiple chunks, got %d", len(chunks))
	}
	var flattened []string
	for i, chunk := range chunks {
		n := windowsArgsLen(append(append([]string{}, base...), chunk...))
		if n > rgCmdLineBudget || n >= windowsCmdLineLimit {
			t.Fatalf("chunk %d command line exceeds budget: %d", i, n)
		}
		flattened = append(flattened, chunk...)
	}
	if !reflect.DeepEqual(flattened, dirs) {
		t.Fatal("chunking lost or reordered directories")
	}
}

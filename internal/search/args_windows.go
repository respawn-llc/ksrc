package search

import (
	"syscall"
	"unicode/utf16"
)

const windowsCmdLineLimit = 32_767

// Leave room for the executable path and terminating NUL.
const rgCmdLineBudget = 28_000

func chunkSearchDirs(baseArgs []string, dirs []string) [][]string {
	if len(dirs) == 0 {
		return [][]string{nil}
	}
	return chunkByBudget(windowsArgsLen(baseArgs), dirs, rgCmdLineBudget)
}

func windowsArgLen(arg string) int {
	return len(utf16.Encode([]rune(syscall.EscapeArg(arg))))
}

func windowsArgsLen(args []string) int {
	n := 0
	for i, arg := range args {
		if i > 0 {
			n++
		}
		n += windowsArgLen(arg)
	}
	return n
}

func chunkByBudget(baseLen int, dirs []string, budget int) [][]string {
	var chunks [][]string
	start, size := 0, baseLen
	for i, dir := range dirs {
		cost := windowsArgLen(dir) + 1
		if size+cost > budget && i > start {
			chunks = append(chunks, dirs[start:i])
			start, size = i, baseLen
		}
		size += cost
	}
	if start < len(dirs) {
		chunks = append(chunks, dirs[start:])
	}
	return chunks
}

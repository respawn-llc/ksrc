//go:build !windows

package search

func chunkSearchDirs(_ []string, dirs []string) [][]string {
	return [][]string{dirs}
}

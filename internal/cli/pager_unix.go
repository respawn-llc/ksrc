//go:build !windows

package cli

import "os/exec"

func pagerCommand(pager string) *exec.Cmd {
	if pager == "" {
		pager = "less -R"
	}
	return exec.Command("sh", "-c", pager)
}

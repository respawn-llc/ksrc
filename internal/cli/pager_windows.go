package cli

import (
	"os/exec"
	"syscall"
)

func pagerCommand(pager string) *exec.Cmd {
	if pager == "" {
		pager = "more"
	}
	cmd := exec.Command("cmd.exe")
	// PAGER is an intentional shell command, not a native argv list.
	cmd.SysProcAttr = &syscall.SysProcAttr{CmdLine: `cmd.exe /d /s /c "` + pager + `"`}
	return cmd
}

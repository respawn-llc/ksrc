package executil

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

func command(ctx context.Context, name string, args ...string) (*exec.Cmd, error) {
	path, err := exec.LookPath(name)
	if err != nil {
		return nil, err
	}
	ext := filepath.Ext(path)
	if !strings.EqualFold(ext, ".bat") && !strings.EqualFold(ext, ".cmd") {
		return exec.CommandContext(ctx, path, args...), nil
	}

	cmd := exec.CommandContext(ctx, "cmd.exe")
	cmd.Env = os.Environ()
	// cmd.exe has different quoting rules from native executables. Expanding
	// quoted environment values avoids interpreting %, &, and other shell
	// syntax in paths and arguments. Delayed expansion must stay disabled.
	var line strings.Builder
	line.WriteString(`cmd.exe /d /v:off /s /c "`)
	for i, value := range append([]string{path}, args...) {
		if strings.ContainsAny(value, "\"\r\n\x00") {
			return nil, fmt.Errorf("cannot pass quotes, newlines, or NUL to a Windows batch command")
		}
		if i > 0 {
			line.WriteByte(' ')
		}
		key := "KSRC_BATCH_ARG_" + strconv.Itoa(i)
		cmd.Env = append(cmd.Env, key+"="+value)
		line.WriteString(`"%` + key + `%"`)
	}
	line.WriteByte('"')
	cmd.SysProcAttr = &syscall.SysProcAttr{CmdLine: line.String()}
	return cmd, nil
}

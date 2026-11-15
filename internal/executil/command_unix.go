//go:build !windows

package executil

import (
	"context"
	"os/exec"
)

func command(ctx context.Context, name string, args ...string) (*exec.Cmd, error) {
	return exec.CommandContext(ctx, name, args...), nil
}

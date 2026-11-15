package executil

import (
	"bytes"
	"context"
	"os/exec"
)

// Runner executes external commands.
type Runner interface {
	Run(ctx context.Context, dir string, name string, args ...string) (stdout string, stderr string, err error)
	LookPath(file string) (string, error)
}

// OSRunner uses os/exec.
type OSRunner struct{}

func (OSRunner) Run(ctx context.Context, dir string, name string, args ...string) (string, string, error) {
	cmd, err := command(ctx, name, args...)
	if err != nil {
		return "", "", err
	}
	cmd.Dir = dir
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	err = cmd.Run()
	return outBuf.String(), errBuf.String(), err
}

func (OSRunner) LookPath(file string) (string, error) {
	return exec.LookPath(file)
}

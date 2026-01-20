package gradle

import (
	"errors"
	"fmt"
)

type ExecError struct {
	Err         error
	Stdout      string
	Stderr      string
	Cmd         string
	Args        []string
	ProjectDir  string
	ProjectPath string
	Invocation  []string
}

func (e *ExecError) Error() string {
	if e == nil {
		return "gradle failed"
	}
	if e.Err != nil {
		return fmt.Sprintf("gradle failed: %v", e.Err)
	}
	return "gradle failed"
}

func (e *ExecError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func AsExecError(err error) (*ExecError, bool) {
	var execErr *ExecError
	if errors.As(err, &execErr) {
		return execErr, true
	}
	return nil, false
}

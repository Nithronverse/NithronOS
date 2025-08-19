package shell

import (
	"bytes"
	"context"
	"errors"
	"os/exec"
	"time"
)

type Result struct {
	Stdout []byte
	Stderr []byte
	Code   int
}

var ErrTimeout = errors.New("command timed out")

func Run(ctx context.Context, timeout time.Duration, name string, args ...string) (Result, error) {
	cctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(cctx, name, args...)
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	err := cmd.Run()
	res := Result{Stdout: outBuf.Bytes(), Stderr: errBuf.Bytes(), Code: exitCode(err)}
	if cctx.Err() == context.DeadlineExceeded {
		return res, ErrTimeout
	}
	return res, err
}

func exitCode(err error) int {
	if err == nil {
		return 0
	}
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		return ee.ExitCode()
	}
	return -1
}




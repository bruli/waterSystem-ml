package python

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

func runCommand(ctx context.Context, timeout time.Duration, name string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, name, args...)

	var (
		stdout bytes.Buffer
		stderr bytes.Buffer
	)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	outStr := strings.TrimSpace(stdout.String())

	combined := outStr
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return combined, fmt.Errorf("timeout excedit executant %s %v", name, args)
	}

	if err != nil {
		if exitErr, ok := errors.AsType[*exec.ExitError](err); ok {
			if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
				return combined, fmt.Errorf("exit code %d", status.ExitStatus())
			}
		}
		return combined, err
	}

	return combined, nil
}

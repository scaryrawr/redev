// Package devpod wraps calls to the devpod CLI.
package devpod

import (
	"context"
	"io"
	"os"
	"os/exec"
)

// Run executes devpod with stdout and stderr connected to the supplied writers.
func Run(ctx context.Context, stdout, stderr io.Writer, args ...string) error {
	cmd := command(ctx, args...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	return cmd.Run()
}

// RunInteractive executes devpod with stdin, stdout, and stderr connected to the supplied streams.
func RunInteractive(ctx context.Context, stdin io.Reader, stdout, stderr io.Writer, args ...string) error {
	return RunInteractiveWithEnv(ctx, stdin, stdout, stderr, nil, args...)
}

// RunInteractiveWithEnv executes devpod interactively with additional environment variables.
func RunInteractiveWithEnv(ctx context.Context, stdin io.Reader, stdout, stderr io.Writer, env []string, args ...string) error {
	cmd := command(ctx, args...)
	if len(env) > 0 {
		cmd.Env = append(os.Environ(), env...)
	}
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	return cmd.Run()
}

func command(ctx context.Context, args ...string) *exec.Cmd {
	return exec.CommandContext(ctx, "devpod", args...)
}

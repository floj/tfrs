//go:build windows

package main

import (
	"errors"
	"os"
	"os/exec"
)

// execProcess runs bin as a child process on Windows, which has no exec(3)
// equivalent. It wires up stdio, forwards the environment and exits with the
// child's exit code so callers observe the same behaviour as on Unix.
func execProcess(bin string, args, env []string) error {
	cmd := exec.Command(bin, args[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = env

	err := cmd.Run()
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		os.Exit(exitErr.ExitCode())
	}
	if err != nil {
		return err
	}
	os.Exit(0)
	return nil
}

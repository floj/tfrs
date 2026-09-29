//go:build !windows

package main

import "syscall"

// execProcess replaces the current process with bin, following the classic
// exec(3) semantics. args[0] is expected to be the program name.
func execProcess(bin string, args, env []string) error {
	return syscall.Exec(bin, args, env)
}

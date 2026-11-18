//go:build !windows

package main

import (
	"os"
	"syscall"
)

// terminateForRestart uses SIGTERM so the ordinary server signal path can
// drain active requests before the service manager starts a new process.
func terminateForRestart() error {
	return syscall.Kill(os.Getpid(), syscall.SIGTERM)
}

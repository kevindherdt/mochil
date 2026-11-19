//go:build !windows

package main

import (
	"os"
	"syscall"
)

// terminateForRestart replaces this process with a fresh copy of itself,
// using the same executable, arguments and environment. A self-exec does not
// depend on the service manager's restart policy: it works whether the
// container or unit is set to restart on failure, restart always, or not
// restart at all, because from the supervisor's point of view the process
// never exits. The previous approach (SIGTERM and wait for the supervisor to
// notice) went dark forever on a restart policy of "on-failure", because a
// clean SIGTERM exit is not a failure — systemd and podman both leave the
// unit stopped rather than restarting it.
//
// Fall back to the old signal-and-hope behaviour only if the self-exec setup
// itself fails (the executable cannot be located, or the kernel refuses the
// exec) — a deployment with a working restart policy still recovers then.
func terminateForRestart() error {
	exe, err := os.Executable()
	if err != nil {
		return syscall.Kill(os.Getpid(), syscall.SIGTERM)
	}
	err = syscall.Exec(exe, os.Args, os.Environ())
	// syscall.Exec only returns on failure; success replaces this process and
	// never reaches here.
	if err != nil {
		return syscall.Kill(os.Getpid(), syscall.SIGTERM)
	}
	return nil
}

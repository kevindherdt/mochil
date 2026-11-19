//go:build windows

package main

import "os"

// Windows has no SIGTERM equivalent. Kill the current process after the HTTP
// restart response has been delivered; the configured service manager owns
// starting its replacement.
func terminateForRestart() error {
	process, err := os.FindProcess(os.Getpid())
	if err != nil {
		return err
	}
	return process.Kill()
}

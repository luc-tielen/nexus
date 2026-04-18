package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

// acquireRunLock writes the current PID to dbDir/nexus.pid and returns a
// release function. If another nexus process is already running it prints a
// message and exits immediately — it never returns in that case.
func acquireRunLock(dbDir string) func() {
	path := filepath.Join(dbDir, "nexus.pid")

	if data, err := os.ReadFile(path); err == nil {
		if pid, err := strconv.Atoi(strings.TrimSpace(string(data))); err == nil && pid > 1 {
			if proc, err := os.FindProcess(pid); err == nil {
				if proc.Signal(syscall.Signal(0)) == nil {
					fmt.Fprintf(os.Stderr, "nexus: already running (pid %d) — stop that process first\n", pid)
					os.Exit(1)
				}
			}
		}
	}

	_ = os.WriteFile(path, []byte(strconv.Itoa(os.Getpid())), 0o600)
	return func() { _ = os.Remove(path) }
}

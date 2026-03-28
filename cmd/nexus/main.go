package main

import (
	"os"
	"os/exec"
	"syscall"
)

func main() {
	claude, err := exec.LookPath("claude")
	if err != nil {
		os.Stderr.WriteString("nexus: claude not found in PATH\n")
		os.Exit(1)
	}

	args := append([]string{"claude"}, os.Args[1:]...)
	if err := syscall.Exec(claude, args, os.Environ()); err != nil {
		os.Stderr.WriteString("nexus: " + err.Error() + "\n")
		os.Exit(1)
	}
}

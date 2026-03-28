package main

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

func main() {
	claude, err := exec.LookPath("claude")
	if err != nil {
		fmt.Fprintln(os.Stderr, "nexus: claude not found in PATH")
		os.Exit(1)
	}

	args := append([]string{"claude"}, os.Args[1:]...)
	if err := syscall.Exec(claude, args, os.Environ()); err != nil {
		fmt.Fprintln(os.Stderr, "nexus:", err)
		os.Exit(1)
	}
}

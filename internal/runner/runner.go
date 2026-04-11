package runner

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/luc/nexus/internal/pty"
)

// Config holds the fixed parameters for spawning claude subprocesses.
type Config struct {
	ClaudePath string
	ClaudeArgs []string
	ExcludeEnv []string // env keys stripped from every subprocess environment
}

// Job describes a single subprocess invocation.
type Job struct {
	Config   Config
	WorkDir  string   // working directory; empty inherits the caller's cwd
	ExtraEnv []string // additional KEY=VALUE pairs injected into the subprocess
	Prompt   string   // passed as --print <prompt>
}

// Run spawns claude --print with job.Prompt in job.WorkDir, injects
// job.ExtraEnv, strips job.Config.ExcludeEnv, and returns the combined
// stdout+stderr output. A non-zero exit code is noted in the output but
// does not cause Run itself to return an error.
func Run(ctx context.Context, job Job) (string, error) {
	args := make([]string, len(job.Config.ClaudeArgs), len(job.Config.ClaudeArgs)+2)
	copy(args, job.Config.ClaudeArgs)
	args = append(args, "--print", job.Prompt)

	cmd := exec.CommandContext(ctx, job.Config.ClaudePath, args...)
	cmd.Dir = job.WorkDir
	cmd.Env = append(pty.FilterEnv(os.Environ(), job.Config.ExcludeEnv), job.ExtraEnv...)

	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	if runErr := cmd.Run(); runErr != nil {
		fmt.Fprintf(&out, "\n[nexus: process exited: %v]", runErr)
	}

	return out.String(), nil
}

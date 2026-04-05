package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/luc/nexus/internal/pty"
)

const cronJobTimeout = 5 * time.Minute

// discordChunkSize is the maximum characters per Discord message.
const discordChunkSize = 1900

// runnerConfig holds the parameters needed to spawn cron job subprocesses.
type runnerConfig struct {
	claudePath string
	claudeArgs []string
	excludeEnv []string
	logSend    func(string) error
}

// makeRunner returns a function that, when called with a message, spawns
// claude --print <message> in a subprocess and forwards its output to cfg.logSend.
// If logSend is nil, output is written to stderr instead.
func makeRunner(ctx context.Context, cfg runnerConfig) func(string) {
	return func(msg string) {
		go func() {
			if err := runCronJob(ctx, cfg, msg); err != nil {
				fmt.Fprintf(os.Stderr, "nexus: cron job error: %v\n", err)
			}
		}()
	}
}

func runCronJob(ctx context.Context, cfg runnerConfig, msg string) error {
	jobCtx, cancel := context.WithTimeout(ctx, cronJobTimeout)
	defer cancel()

	args := make([]string, len(cfg.claudeArgs), len(cfg.claudeArgs)+2)
	copy(args, cfg.claudeArgs)
	args = append(args, "--print", msg)

	cmd := exec.CommandContext(jobCtx, cfg.claudePath, args...)
	cmd.Env = pty.FilterEnv(os.Environ(), cfg.excludeEnv)

	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	runErr := cmd.Run()
	if runErr != nil {
		fmt.Fprintf(&out, "\n[nexus: process exited: %v]", runErr)
	}

	output := out.String()
	if output == "" {
		return nil
	}

	if cfg.logSend == nil {
		fmt.Fprintf(os.Stderr, "nexus: cron output:\n%s\n", output)
		return nil
	}

	for len(output) > 0 {
		chunk := output
		if len(chunk) > discordChunkSize {
			chunk = output[:discordChunkSize]
		}
		output = output[len(chunk):]
		if err := cfg.logSend(chunk); err != nil {
			fmt.Fprintf(os.Stderr, "nexus: sending cron output to Discord: %v\n", err)
		}
	}
	return nil
}

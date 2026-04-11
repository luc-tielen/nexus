package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/luc/nexus/internal/runner"
)

const cronJobTimeout = 5 * time.Minute

// discordChunkSize is the maximum characters per Discord message.
const discordChunkSize = 1900

// cronConfig holds the parameters needed to spawn cron job subprocesses.
type cronConfig struct {
	runnerCfg runner.Config
	logSend   func(string) error
}

// makeCronRunner returns a function that, when called with a message, spawns
// claude --print <message> in a subprocess and forwards its output to cfg.logSend.
// If logSend is nil, output is written to stderr instead.
func makeCronRunner(ctx context.Context, cfg cronConfig) func(string) {
	return func(msg string) {
		go func() {
			if err := runCronJob(ctx, cfg, msg); err != nil {
				fmt.Fprintf(os.Stderr, "nexus: cron job error: %v\n", err)
			}
		}()
	}
}

func runCronJob(ctx context.Context, cfg cronConfig, msg string) error {
	jobCtx, cancel := context.WithTimeout(ctx, cronJobTimeout)
	defer cancel()

	output, err := runner.Run(jobCtx, runner.Job{
		Config: cfg.runnerCfg,
		Prompt: msg,
	})
	if err != nil {
		return err
	}
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

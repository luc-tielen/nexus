package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

// TestMain enables the helper-process pattern: when NEXUS_TEST_HELPER is set,
// the test binary acts as a fake claude subprocess instead of running tests.
func TestMain(m *testing.M) {
	if mode := os.Getenv("NEXUS_TEST_HELPER"); mode != "" {
		runTestHelper(mode)
		return // runTestHelper calls os.Exit; this is a safety net
	}
	os.Exit(m.Run())
}

func runTestHelper(mode string) {
	switch mode {
	case "print":
		fmt.Print(os.Getenv("NEXUS_TEST_OUTPUT"))
		os.Exit(0)
	case "fail":
		fmt.Print(os.Getenv("NEXUS_TEST_OUTPUT"))
		os.Exit(1)
	case "sleep":
		time.Sleep(10 * time.Minute)
		os.Exit(0)
	}
	os.Exit(0)
}

// collectLogSend returns a logSend function and a pointer to the slice it appends to.
func collectLogSend() (func(string) error, *[]string) {
	var chunks []string
	return func(s string) error {
		chunks = append(chunks, s)
		return nil
	}, &chunks
}

func testConfig(logSend func(string) error) runnerConfig {
	return runnerConfig{
		claudePath: os.Args[0],
		logSend:    logSend,
	}
}

func TestRunCronJob_SendsOutputToLogSend(t *testing.T) {
	t.Setenv("NEXUS_TEST_HELPER", "print")
	t.Setenv("NEXUS_TEST_OUTPUT", "hello from cron")

	logSend, chunks := collectLogSend()
	if err := runCronJob(context.Background(), testConfig(logSend), "msg"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := strings.Join(*chunks, ""); got != "hello from cron" {
		t.Errorf("got %q, want %q", got, "hello from cron")
	}
}

func TestRunCronJob_EmptyOutputSkipsLogSend(t *testing.T) {
	t.Setenv("NEXUS_TEST_HELPER", "print")
	t.Setenv("NEXUS_TEST_OUTPUT", "")

	logSend, chunks := collectLogSend()
	if err := runCronJob(context.Background(), testConfig(logSend), "msg"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(*chunks) != 0 {
		t.Errorf("expected no logSend calls for empty output, got %d", len(*chunks))
	}
}

func TestRunCronJob_LongOutputChunked(t *testing.T) {
	output := strings.Repeat("x", discordChunkSize*2+50)
	t.Setenv("NEXUS_TEST_HELPER", "print")
	t.Setenv("NEXUS_TEST_OUTPUT", output)

	logSend, chunks := collectLogSend()
	if err := runCronJob(context.Background(), testConfig(logSend), "msg"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := strings.Join(*chunks, ""); got != output {
		t.Errorf("reassembled output does not match (len %d vs %d)", len(got), len(output))
	}
	for i, c := range *chunks {
		if len(c) > discordChunkSize {
			t.Errorf("chunk %d exceeds discordChunkSize: len=%d", i, len(c))
		}
	}
	if len(*chunks) != 3 {
		t.Errorf("expected 3 chunks, got %d", len(*chunks))
	}
}

func TestRunCronJob_NonZeroExitAppendsError(t *testing.T) {
	t.Setenv("NEXUS_TEST_HELPER", "fail")
	t.Setenv("NEXUS_TEST_OUTPUT", "partial output")

	logSend, chunks := collectLogSend()
	// runCronJob itself returns nil even on subprocess failure; error is in the output.
	_ = runCronJob(context.Background(), testConfig(logSend), "msg")

	combined := strings.Join(*chunks, "")
	if !strings.Contains(combined, "partial output") {
		t.Errorf("expected stdout in output, got: %q", combined)
	}
	if !strings.Contains(combined, "nexus: process exited:") {
		t.Errorf("expected exit error in output, got: %q", combined)
	}
}

func TestRunCronJob_CancelledContextReportsError(t *testing.T) {
	t.Setenv("NEXUS_TEST_HELPER", "sleep")

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	logSend, chunks := collectLogSend()
	_ = runCronJob(ctx, testConfig(logSend), "msg")

	combined := strings.Join(*chunks, "")
	if !strings.Contains(combined, "nexus: process exited:") {
		t.Errorf("expected error in output, got: %q", combined)
	}
}

func TestMakeRunner_SendsOutputToLogSend(t *testing.T) {
	t.Setenv("NEXUS_TEST_HELPER", "print")
	t.Setenv("NEXUS_TEST_OUTPUT", "async result")

	done := make(chan string, 1)
	logSend := func(s string) error {
		done <- s
		return nil
	}

	runner := makeRunner(context.Background(), testConfig(logSend))
	runner("msg")

	select {
	case got := <-done:
		if got != "async result" {
			t.Errorf("got %q, want %q", got, "async result")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for cron output")
	}
}

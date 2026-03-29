package main

import (
	"context"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

const mcpPort = "7744"
const mcpAddr = ":" + mcpPort
const mcpBaseURL = "http://localhost:" + mcpPort

// newMCPServer creates the MCP server and registers all tools.
// The wrapper is passed so tool handlers can interact with the running Claude
// session (e.g. via w.Inject).
func newMCPServer(w *wrapper) *mcpserver.SSEServer {
	s := mcpserver.NewMCPServer("nexus", "0.1.0")

	s.AddTool(
		mcp.NewTool("get_time",
			mcp.WithDescription("Returns the current time on the host machine."),
		),
		func(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return mcp.NewToolResultText(time.Now().String()), nil
		},
	)

	return mcpserver.NewSSEServer(s, mcpserver.WithBaseURL(mcpBaseURL))
}

// startMCPServer launches the SSE server in the background and returns a
// shutdown function. It blocks briefly until the server is ready.
func startMCPServer(ctx context.Context, w *wrapper) (shutdown func(), err error) {
	srv := newMCPServer(w)

	errCh := make(chan error, 1)
	go func() {
		if err := srv.Start(mcpAddr); err != nil {
			errCh <- err
		}
	}()

	// Give the server a moment to start; surface any immediate bind errors.
	select {
	case err := <-errCh:
		return nil, err
	case <-time.After(50 * time.Millisecond):
	}

	shutdown = func() {
		shutCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutCtx)
	}
	return shutdown, nil
}

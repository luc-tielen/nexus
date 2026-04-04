package server

import (
	"context"
	"time"

	"github.com/luc/nexus/internal/discord"
	"github.com/luc/nexus/internal/pty"
	"github.com/luc/nexus/internal/scheduler"
	"github.com/luc/nexus/internal/todoist"
	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

const mcpPort = "7744"
const mcpAddr = ":" + mcpPort

// New creates the MCP server and registers all tools.
func New(w *pty.Wrapper, s *scheduler.Scheduler, dc *discord.Client, tc *todoist.Client) *mcpserver.StreamableHTTPServer {
	srv := mcpserver.NewMCPServer("nexus", "0.1.0")

	srv.AddTool(
		mcp.NewTool("get_time",
			mcp.WithDescription("Returns the current time on the host machine."),
		),
		func(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleGetTime(req)
		},
	)

	registerSchedulerTools(srv, s)
	registerDiscordTools(srv, dc)
	registerTodoistTools(srv, tc)

	_ = w // reserved for future tools that need the wrapper
	return mcpserver.NewStreamableHTTPServer(srv)
}

func handleGetTime(_ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return mcp.NewToolResultText(time.Now().String()), nil
}

// Start launches the MCP server in the background and returns a shutdown
// function. It blocks briefly until the server is ready.
func Start(ctx context.Context, w *pty.Wrapper, s *scheduler.Scheduler, dc *discord.Client, tc *todoist.Client) (shutdown func(), err error) {
	srv := New(w, s, dc, tc)

	errCh := make(chan error, 1)
	go func() {
		if err := srv.Start(mcpAddr); err != nil {
			errCh <- err
		}
	}()

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

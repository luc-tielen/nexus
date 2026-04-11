package server

import (
	"context"
	"time"

	"github.com/luc/nexus/internal/discord"
	"github.com/luc/nexus/internal/projects"
	"github.com/luc/nexus/internal/pty"
	"github.com/luc/nexus/internal/runner"
	"github.com/luc/nexus/internal/scheduler"
	"github.com/luc/nexus/internal/secrets"
	"github.com/luc/nexus/internal/telegram"
	"github.com/luc/nexus/internal/todoist"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

const mcpPort = "7744"
const mcpAddr = ":" + mcpPort

// New creates the MCP server and registers all tools.
func New(w *pty.Wrapper, s *scheduler.Scheduler, dc *discord.Client, tgc *telegram.Client, tc *todoist.Client, ps *projects.Store, ss *secrets.Store, cfg runner.Config) *mcpserver.StreamableHTTPServer {
	srv := mcpserver.NewMCPServer("nexus", "0.1.0")

	registerSchedulerTools(srv, s)
	registerDiscordTools(srv, dc)
	registerTelegramTools(srv, tgc)
	registerTodoTools(srv, tc)
	registerProjectTools(srv, ps, ss, cfg)

	_ = w // reserved for future tools that need the wrapper
	return mcpserver.NewStreamableHTTPServer(srv)
}

// Start launches the MCP server in the background and returns a shutdown
// function. It blocks briefly until the server is ready.
func Start(ctx context.Context, w *pty.Wrapper, s *scheduler.Scheduler, dc *discord.Client, tgc *telegram.Client, tc *todoist.Client, ps *projects.Store, ss *secrets.Store, cfg runner.Config) (shutdown func(), err error) {
	srv := New(w, s, dc, tgc, tc, ps, ss, cfg)

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

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

// Options holds all dependencies for the MCP server.
type Options struct {
	PTY       *pty.Wrapper
	Scheduler *scheduler.Scheduler
	Discord   *discord.Client
	Telegram  *telegram.Client
	Todoist   *todoist.Client
	Projects  *projects.Store
	Secrets   *secrets.Store
	Runner    runner.Config
}

// New creates the MCP server and registers all tools.
func New(opts Options) *mcpserver.StreamableHTTPServer {
	srv := mcpserver.NewMCPServer("nexus", "0.1.0")

	registerSchedulerTools(srv, opts.Scheduler)
	registerDiscordTools(srv, opts.Discord)
	registerTelegramTools(srv, opts.Telegram)
	registerTodoTools(srv, opts.Todoist)
	registerProjectTools(srv, opts.Projects, opts.Secrets, opts.Runner)
	registerPTYTools(srv, opts.PTY)

	return mcpserver.NewStreamableHTTPServer(srv)
}

// Start launches the MCP server in the background and returns a shutdown
// function. It blocks briefly until the server is ready.
func Start(ctx context.Context, opts Options) (shutdown func(), err error) {
	srv := New(opts)

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

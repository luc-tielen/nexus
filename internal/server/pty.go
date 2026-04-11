package server

import (
	"context"

	"github.com/luc/nexus/internal/pty"
	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

func registerPTYTools(srv *mcpserver.MCPServer, w *pty.Wrapper) {
	srv.AddTool(
		mcp.NewTool("clear_context",
			mcp.WithDescription("Clear the current conversation context, equivalent to typing /clear in the terminal. "+
				"Use this when a remote channel (e.g. Telegram, Discord) sends a /clear command."),
		),
		func(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleClearContext(w)
		},
	)
}

func handleClearContext(w *pty.Wrapper) (*mcp.CallToolResult, error) {
	// Injecting directly into the PTY is fine here: /clear only arrives
	// via a remote channel, never from the local terminal at this point.
	w.WriteInput([]byte("/clear\r"))
	return mcp.NewToolResultText("context cleared"), nil
}

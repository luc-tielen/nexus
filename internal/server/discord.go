package server

import (
	"context"
	"fmt"

	"github.com/luc/nexus/internal/discord"
	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

func registerDiscordTools(srv *mcpserver.MCPServer, dc *discord.Client) {
	srv.AddTool(
		mcp.NewTool("send_discord_message",
			mcp.WithDescription("Send a message to the configured Discord channel. "+
				"Requires DISCORD_BOT_TOKEN and DISCORD_CHANNEL_ID environment variables."),
			mcp.WithString("message",
				mcp.Required(),
				mcp.Description("The message content to send."),
			),
		),
		func(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleSendDiscordMessage(dc, req)
		},
	)
}

func handleSendDiscordMessage(dc *discord.Client, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if dc == nil {
		return nil, fmt.Errorf("discord is not configured: set DISCORD_BOT_TOKEN and DISCORD_CHANNEL_ID")
	}
	message := req.GetString("message", "")
	if message == "" {
		return nil, fmt.Errorf("message is required")
	}
	if err := dc.Send(message); err != nil {
		return nil, fmt.Errorf("sending Discord message: %w", err)
	}
	return mcp.NewToolResultText("message sent"), nil
}

package server

import (
	"context"
	"fmt"

	"github.com/luc/nexus/internal/telegram"
	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

func registerTelegramTools(srv *mcpserver.MCPServer, tc *telegram.Client) {
	srv.AddTool(
		mcp.NewTool("send_telegram_message",
			mcp.WithDescription("Send a message to the configured Telegram chat. "+
				"Requires TELEGRAM_BOT_TOKEN and TELEGRAM_CHAT_ID environment variables."),
			mcp.WithString("message",
				mcp.Required(),
				mcp.Description("The message content to send."),
			),
		),
		func(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleSendTelegramMessage(tc, req)
		},
	)
}

func handleSendTelegramMessage(tc *telegram.Client, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if tc == nil {
		return nil, fmt.Errorf("telegram is not configured: set TELEGRAM_BOT_TOKEN and TELEGRAM_CHAT_ID")
	}
	message := req.GetString("message", "")
	if message == "" {
		return nil, fmt.Errorf("message is required")
	}
	if err := tc.Send(message); err != nil {
		return nil, fmt.Errorf("sending Telegram message: %w", err)
	}
	return mcp.NewToolResultText("message sent"), nil
}

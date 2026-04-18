package server

import (
	"context"
	"fmt"
	"strconv"

	"github.com/luc/nexus/internal/telegram"
	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

func registerTelegramTools(srv *mcpserver.MCPServer, tc *telegram.Client) {
	srv.AddTool(
		mcp.NewTool("send_telegram_message",
			mcp.WithDescription("Send a message to a Telegram chat. "+
				"Use the chat_id from the incoming [Telegram from <chat_id>] prefix to reply to the correct chat. "+
				"Omit chat_id to send to the default configured chat."),
			mcp.WithString("message",
				mcp.Required(),
				mcp.Description("The message content to send."),
			),
			mcp.WithString("chat_id",
				mcp.Description("Target chat ID. Use the ID from the [Telegram from <chat_id>] prefix to reply to the correct chat."),
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
	targetChatID := tc.ChatID
	if chatIDStr := req.GetString("chat_id", ""); chatIDStr != "" {
		id, err := strconv.ParseInt(chatIDStr, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("chat_id must be a numeric ID: %w", err)
		}
		targetChatID = id
	}
	if err := tc.SendTo(targetChatID, message); err != nil {
		return nil, fmt.Errorf("sending Telegram message: %w", err)
	}
	return mcp.NewToolResultText("message sent"), nil
}

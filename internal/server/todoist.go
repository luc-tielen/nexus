package server

import (
	"context"
	"fmt"

	"github.com/luc/nexus/internal/todoist"
	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

func registerTodoistTools(srv *mcpserver.MCPServer, tc *todoist.Client) {
	srv.AddTool(
		mcp.NewTool("create_todoist_task",
			mcp.WithDescription("Create a new task in Todoist. "+
				"Requires TODOIST_API_KEY environment variable."),
			mcp.WithString("content",
				mcp.Required(),
				mcp.Description("The task content."),
			),
		),
		func(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleCreateTodoistTask(tc, req)
		},
	)
}

func handleCreateTodoistTask(tc *todoist.Client, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if tc == nil {
		return nil, fmt.Errorf("todoist is not configured: set TODOIST_API_KEY")
	}
	content := req.GetString("content", "")
	if content == "" {
		return nil, fmt.Errorf("content is required")
	}
	if err := tc.CreateTask(content); err != nil {
		return nil, fmt.Errorf("creating Todoist task: %w", err)
	}
	return mcp.NewToolResultText("task created"), nil
}

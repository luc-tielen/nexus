package server

import (
	"context"
	"fmt"
	"strings"

	"github.com/luc/nexus/internal/todoist"
	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

func registerTodoTools(srv *mcpserver.MCPServer, tc *todoist.Client) {
	srv.AddTool(
		mcp.NewTool("list_tasks",
			mcp.WithDescription("List all active to-do tasks."),
		),
		func(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleListTodoTasks(tc, req)
		},
	)
	srv.AddTool(
		mcp.NewTool("complete_task",
			mcp.WithDescription("Mark a task as completed."),
			mcp.WithString("task_id",
				mcp.Required(),
				mcp.Description("The ID of the task to complete."),
			),
		),
		func(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleCompleteTodoTask(tc, req)
		},
	)
	srv.AddTool(
		mcp.NewTool("create_task",
			mcp.WithDescription("Create a new to-do task."),
			mcp.WithString("content",
				mcp.Required(),
				mcp.Description("The task content."),
			),
		),
		func(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleCreateTodoTask(tc, req)
		},
	)
}

func handleListTodoTasks(tc *todoist.Client, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if tc == nil {
		return nil, fmt.Errorf("todo backend is not configured")
	}
	tasks, err := tc.ListTasks()
	if err != nil {
		return nil, fmt.Errorf("listing tasks: %w", err)
	}
	if len(tasks) == 0 {
		return mcp.NewToolResultText("no tasks"), nil
	}
	var lines []string
	for _, t := range tasks {
		lines = append(lines, fmt.Sprintf("[%s] %s", t.ID, t.Content))
	}
	return mcp.NewToolResultText(strings.Join(lines, "\n")), nil
}

func handleCompleteTodoTask(tc *todoist.Client, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if tc == nil {
		return nil, fmt.Errorf("todo backend is not configured")
	}
	taskID := req.GetString("task_id", "")
	if taskID == "" {
		return nil, fmt.Errorf("task_id is required")
	}
	if err := tc.CompleteTask(taskID); err != nil {
		return nil, fmt.Errorf("completing task: %w", err)
	}
	return mcp.NewToolResultText("task completed"), nil
}

func handleCreateTodoTask(tc *todoist.Client, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if tc == nil {
		return nil, fmt.Errorf("todo backend is not configured")
	}
	content := req.GetString("content", "")
	if content == "" {
		return nil, fmt.Errorf("content is required")
	}
	if err := tc.CreateTask(content); err != nil {
		return nil, fmt.Errorf("creating task: %w", err)
	}
	return mcp.NewToolResultText("task created"), nil
}

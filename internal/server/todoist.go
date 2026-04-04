package server

import (
	"context"
	"fmt"
	"strings"

	"github.com/luc/nexus/internal/todoist"
	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

func registerTodoistTools(srv *mcpserver.MCPServer, tc *todoist.Client) {
	srv.AddTool(
		mcp.NewTool("list_todoist_tasks",
			mcp.WithDescription("List all active tasks in Todoist. "+
				"Requires TODOIST_API_KEY environment variable."),
		),
		func(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleListTodoistTasks(tc, req)
		},
	)
	srv.AddTool(
		mcp.NewTool("complete_todoist_task",
			mcp.WithDescription("Mark a Todoist task as completed. "+
				"Requires TODOIST_API_KEY environment variable."),
			mcp.WithString("task_id",
				mcp.Required(),
				mcp.Description("The ID of the task to complete."),
			),
		),
		func(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleCompleteTodoistTask(tc, req)
		},
	)
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

func handleListTodoistTasks(tc *todoist.Client, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if tc == nil {
		return nil, fmt.Errorf("todoist is not configured: set TODOIST_API_KEY")
	}
	tasks, err := tc.ListTasks()
	if err != nil {
		return nil, fmt.Errorf("listing Todoist tasks: %w", err)
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

func handleCompleteTodoistTask(tc *todoist.Client, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if tc == nil {
		return nil, fmt.Errorf("todoist is not configured: set TODOIST_API_KEY")
	}
	taskID := req.GetString("task_id", "")
	if taskID == "" {
		return nil, fmt.Errorf("task_id is required")
	}
	if err := tc.CompleteTask(taskID); err != nil {
		return nil, fmt.Errorf("completing Todoist task: %w", err)
	}
	return mcp.NewToolResultText("task completed"), nil
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

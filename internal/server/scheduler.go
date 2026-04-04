package server

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/luc/nexus/internal/scheduler"
	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

func registerSchedulerTools(srv *mcpserver.MCPServer, s *scheduler.Scheduler) {
	srv.AddTool(
		mcp.NewTool("schedule_task",
			mcp.WithDescription("Schedule a recurring task using a cron expression. "+
				"Accepts standard 5-field expressions ('*/5 * * * *') or descriptors "+
				"like '@hourly' and '@every 30m'. Returns the task ID."),
			mcp.WithString("schedule",
				mcp.Required(),
				mcp.Description("Cron expression, e.g. '0 9 * * 1-5' for weekdays at 9am."),
			),
			mcp.WithString("message",
				mcp.Required(),
				mcp.Description("Message to inject into Claude when the schedule fires."),
			),
		),
		func(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleScheduleTask(s, req)
		},
	)

	srv.AddTool(
		mcp.NewTool("list_tasks",
			mcp.WithDescription("List all currently scheduled tasks."),
		),
		func(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleListTasks(s, req)
		},
	)

	srv.AddTool(
		mcp.NewTool("delete_task",
			mcp.WithDescription("Delete a scheduled task by ID."),
			mcp.WithString("id",
				mcp.Required(),
				mcp.Description("Task ID returned by schedule_task."),
			),
		),
		func(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleDeleteTask(s, req)
		},
	)
}

func handleScheduleTask(s *scheduler.Scheduler, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	schedule := req.GetString("schedule", "")
	message := req.GetString("message", "")
	if schedule == "" {
		return nil, fmt.Errorf("schedule is required")
	}
	if message == "" {
		return nil, fmt.Errorf("message is required")
	}
	id, err := s.Add(schedule, message)
	if err != nil {
		return nil, err
	}
	return mcp.NewToolResultText(fmt.Sprintf("scheduled task %s", id)), nil
}

func handleListTasks(s *scheduler.Scheduler, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	jobs := s.List()
	type row struct {
		ID       string `json:"id"`
		Schedule string `json:"schedule"`
		Message  string `json:"message"`
	}
	rows := make([]row, len(jobs))
	for i, j := range jobs {
		rows[i] = row{ID: j.ID, Schedule: j.Schedule, Message: j.Message}
	}
	out, _ := json.Marshal(rows)
	return mcp.NewToolResultText(string(out)), nil
}

func handleDeleteTask(s *scheduler.Scheduler, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id := req.GetString("id", "")
	if id == "" {
		return nil, fmt.Errorf("id is required")
	}
	if !s.Delete(id) {
		return nil, fmt.Errorf("task %s not found", id)
	}
	return mcp.NewToolResultText(fmt.Sprintf("deleted task %s", id)), nil
}

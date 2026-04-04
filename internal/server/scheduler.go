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
		mcp.NewTool("add_cron",
			mcp.WithDescription("Schedule a recurring task using a cron expression. "+
				"Accepts standard 5-field expressions ('*/5 * * * *') or descriptors "+
				"like '@hourly' and '@every 30m'. Returns the cron ID."),
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
			return handleAddCron(s, req)
		},
	)

	srv.AddTool(
		mcp.NewTool("list_crons",
			mcp.WithDescription("List all currently scheduled cron jobs."),
		),
		func(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleListCrons(s, req)
		},
	)

	srv.AddTool(
		mcp.NewTool("delete_cron",
			mcp.WithDescription("Delete a scheduled cron job by ID."),
			mcp.WithString("id",
				mcp.Required(),
				mcp.Description("Cron ID returned by add_cron."),
			),
		),
		func(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleDeleteCron(s, req)
		},
	)
}

func handleAddCron(s *scheduler.Scheduler, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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
	return mcp.NewToolResultText(fmt.Sprintf("added cron %s", id)), nil
}

func handleListCrons(s *scheduler.Scheduler, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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

func handleDeleteCron(s *scheduler.Scheduler, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id := req.GetString("id", "")
	if id == "" {
		return nil, fmt.Errorf("id is required")
	}
	if !s.Delete(id) {
		return nil, fmt.Errorf("task %s not found", id)
	}
	return mcp.NewToolResultText(fmt.Sprintf("deleted cron %s", id)), nil
}

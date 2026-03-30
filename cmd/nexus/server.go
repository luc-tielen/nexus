package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

const mcpPort = "7744"
const mcpAddr = ":" + mcpPort
const mcpBaseURL = "http://localhost:" + mcpPort

// newMCPServer creates the MCP server and registers all tools.
func newMCPServer(w *wrapper, s *scheduler, dc *discordClient) *mcpserver.SSEServer {
	srv := mcpserver.NewMCPServer("nexus", "0.1.0")

	srv.AddTool(
		mcp.NewTool("get_time",
			mcp.WithDescription("Returns the current time on the host machine."),
		),
		func(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return mcp.NewToolResultText(time.Now().String()), nil
		},
	)

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
		},
	)

	srv.AddTool(
		mcp.NewTool("list_tasks",
			mcp.WithDescription("List all currently scheduled tasks."),
		),
		func(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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
			id := req.GetString("id", "")
			if id == "" {
				return nil, fmt.Errorf("id is required")
			}
			if !s.Delete(id) {
				return nil, fmt.Errorf("task %s not found", id)
			}
			return mcp.NewToolResultText(fmt.Sprintf("deleted task %s", id)), nil
		},
	)

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
		},
	)

	return mcpserver.NewSSEServer(srv, mcpserver.WithBaseURL(mcpBaseURL))
}

// startMCPServer launches the SSE server in the background and returns a
// shutdown function. It blocks briefly until the server is ready.
func startMCPServer(ctx context.Context, w *wrapper, s *scheduler, dc *discordClient) (shutdown func(), err error) {
	srv := newMCPServer(w, s, dc)

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

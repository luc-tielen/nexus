package server

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/luc/nexus/internal/projects"
	"github.com/luc/nexus/internal/runner"
	"github.com/luc/nexus/internal/secrets"
	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

const projectJobTimeout = 5 * time.Minute

// projectState holds the name of the currently active project in memory.
type projectState struct {
	mu      sync.Mutex
	current string
}

func (ps *projectState) set(name string) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	ps.current = name
}

func (ps *projectState) get() string {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	return ps.current
}

// projectEnv builds the KEY=VALUE env slice for a project by injecting all
// secrets whose key starts with "projectName::", stripping the prefix to get
// the env var name.
func projectEnv(ss *secrets.Store, projectName string) []string {
	keys := ss.KeysForProject(projectName)
	prefix := projectName + "::"
	env := make([]string, 0, len(keys))
	for _, k := range keys {
		val, ok := ss.Get(k)
		if !ok {
			continue
		}
		envVar := k[len(prefix):]
		env = append(env, envVar+"="+val)
	}
	return env
}

func registerProjectTools(srv *mcpserver.MCPServer, store *projects.Store, ss *secrets.Store, cfg runner.Config) {
	state := &projectState{}

	srv.AddTool(
		mcp.NewTool("list_projects",
			mcp.WithDescription("List all configured projects with their paths and env injection mappings."),
		),
		func(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleListProjects(store, req)
		},
	)

	srv.AddTool(
		mcp.NewTool("switch_project",
			mcp.WithDescription("Switch the active project for this session. "+
				"Returns the project path and a list of env vars to export so that "+
				"subsequent work operates in the correct directory. "+
				"Pass an empty name to clear the active project."),
			mcp.WithString("name",
				mcp.Required(),
				mcp.Description("Project name to switch to, or empty string to clear."),
			),
		),
		func(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleSwitchProject(store, ss, state, req)
		},
	)

	srv.AddTool(
		mcp.NewTool("get_current_project",
			mcp.WithDescription("Return the name and path of the currently active project, "+
				"or an empty result if no project is active."),
		),
		func(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleGetCurrentProject(store, state, req)
		},
	)

	srv.AddTool(
		mcp.NewTool("run_in_project",
			mcp.WithDescription("Run a prompt via 'claude --print' inside a project's directory "+
				"with the project's secrets injected as environment variables. "+
				"Blocks until the subprocess finishes and returns the full output."),
			mcp.WithString("project",
				mcp.Required(),
				mcp.Description("Project name."),
			),
			mcp.WithString("prompt",
				mcp.Required(),
				mcp.Description("Prompt to pass to claude --print."),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleRunInProject(ctx, store, ss, cfg, req)
		},
	)
}

func handleListProjects(store *projects.Store, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	list, err := store.List()
	if err != nil {
		return nil, err
	}
	type row struct {
		Name string `json:"name"`
		Path string `json:"path"`
	}
	rows := make([]row, len(list))
	for i, p := range list {
		rows[i] = row{Name: p.Name, Path: p.Path}
	}
	out, _ := json.Marshal(rows)
	return mcp.NewToolResultText(string(out)), nil
}

func handleSwitchProject(store *projects.Store, ss *secrets.Store, state *projectState, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name := req.GetString("name", "")
	if name == "" {
		state.set("")
		return mcp.NewToolResultText("active project cleared"), nil
	}

	p, err := store.Get(name)
	if err != nil {
		return nil, err
	}

	type exportLine struct {
		EnvVar string `json:"env_var"`
		Value  string `json:"value"`
	}
	prefix := name + "::"
	keys := ss.KeysForProject(name)
	exports := make([]exportLine, 0, len(keys))
	for _, k := range keys {
		val, ok := ss.Get(k)
		if !ok {
			continue
		}
		exports = append(exports, exportLine{EnvVar: k[len(prefix):], Value: val})
	}

	state.set(name)

	type result struct {
		Project string       `json:"project"`
		Path    string       `json:"path"`
		Exports []exportLine `json:"exports"`
	}
	out, _ := json.Marshal(result{Project: name, Path: p.Path, Exports: exports})
	return mcp.NewToolResultText(string(out)), nil
}

func handleGetCurrentProject(store *projects.Store, state *projectState, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name := state.get()
	if name == "" {
		return mcp.NewToolResultText(`{"project":"","path":""}`), nil
	}
	p, err := store.Get(name)
	if err != nil {
		return nil, err
	}
	type result struct {
		Project string `json:"project"`
		Path    string `json:"path"`
	}
	out, _ := json.Marshal(result{Project: p.Name, Path: p.Path})
	return mcp.NewToolResultText(string(out)), nil
}

func handleRunInProject(ctx context.Context, store *projects.Store, ss *secrets.Store, cfg runner.Config, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectName := req.GetString("project", "")
	prompt := req.GetString("prompt", "")
	if projectName == "" {
		return nil, fmt.Errorf("project is required")
	}
	if prompt == "" {
		return nil, fmt.Errorf("prompt is required")
	}

	p, err := store.Get(projectName)
	if err != nil {
		return nil, err
	}

	extraEnv := projectEnv(ss, projectName)

	jobCtx, cancel := context.WithTimeout(ctx, projectJobTimeout)
	defer cancel()

	output, err := runner.Run(jobCtx, runner.Job{
		Config:   cfg,
		WorkDir:  p.Path,
		ExtraEnv: extraEnv,
		Prompt:   prompt,
	})
	if err != nil {
		return nil, err
	}
	return mcp.NewToolResultText(output), nil
}

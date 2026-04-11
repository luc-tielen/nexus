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
		mcp.NewTool("add_project",
			mcp.WithDescription("Add or update a project mapping (name → directory path). "+
				"The path should be an absolute path to the project's root directory."),
			mcp.WithString("name",
				mcp.Required(),
				mcp.Description("Short identifier for the project, e.g. 'myapp'."),
			),
			mcp.WithString("path",
				mcp.Required(),
				mcp.Description("Absolute path to the project directory."),
			),
		),
		func(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleAddProject(store, req)
		},
	)

	srv.AddTool(
		mcp.NewTool("delete_project",
			mcp.WithDescription("Delete a project, its env injection mappings, and all its secrets. "+
				"Call without 'confirm' first to see a summary of what will be deleted. "+
				"This action is irreversible."),
			mcp.WithString("name",
				mcp.Required(),
				mcp.Description("Project name to delete."),
			),
			mcp.WithBoolean("confirm",
				mcp.Description("Set to true to confirm deletion. Without this the tool performs a dry run and shows what would be deleted."),
			),
		),
		func(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleDeleteProject(store, ss, state, req)
		},
	)

	srv.AddTool(
		mcp.NewTool("list_projects",
			mcp.WithDescription("List all configured projects with their paths and env injection mappings."),
		),
		func(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleListProjects(store, req)
		},
	)

	srv.AddTool(
		mcp.NewTool("add_project_env",
			mcp.WithDescription("Configure a secret to be injected as an environment variable when "+
				"running subprocesses in a project context. "+
				"The secret must exist in the secrets store under the key 'projectname::SECRET_KEY'."),
			mcp.WithString("project",
				mcp.Required(),
				mcp.Description("Project name."),
			),
			mcp.WithString("secret_key",
				mcp.Required(),
				mcp.Description("Full secret key as stored (e.g. 'myapp::DB_URL')."),
			),
			mcp.WithString("env_var",
				mcp.Required(),
				mcp.Description("Environment variable name to inject the secret as (e.g. 'DATABASE_URL')."),
			),
		),
		func(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleAddProjectEnv(store, req)
		},
	)

	srv.AddTool(
		mcp.NewTool("delete_project_env",
			mcp.WithDescription("Remove an env injection mapping from a project."),
			mcp.WithString("project",
				mcp.Required(),
				mcp.Description("Project name."),
			),
			mcp.WithString("secret_key",
				mcp.Required(),
				mcp.Description("Secret key to remove (e.g. 'myapp::DB_URL')."),
			),
		),
		func(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleDeleteProjectEnv(store, req)
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

func handleAddProject(store *projects.Store, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name := req.GetString("name", "")
	path := req.GetString("path", "")
	if name == "" {
		return nil, fmt.Errorf("name is required")
	}
	if path == "" {
		return nil, fmt.Errorf("path is required")
	}
	if err := store.Add(name, path); err != nil {
		return nil, err
	}
	return mcp.NewToolResultText(fmt.Sprintf("project %q saved (path: %s)", name, path)), nil
}

func handleDeleteProject(store *projects.Store, ss *secrets.Store, state *projectState, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name := req.GetString("name", "")
	if name == "" {
		return nil, fmt.Errorf("name is required")
	}

	p, err := store.Get(name)
	if err != nil {
		return nil, err
	}

	secretKeys := ss.KeysForProject(name)

	if !req.GetBool("confirm", false) {
		type dryRun struct {
			Warning string   `json:"warning"`
			Project string   `json:"project"`
			Path    string   `json:"path"`
			Secrets []string `json:"secrets"`
			EnvMaps int      `json:"env_mappings"`
		}
		envs, _ := store.ListEnv(name)
		out, _ := json.Marshal(dryRun{
			Warning: "This action is irreversible. Call again with confirm:true to proceed.",
			Project: p.Name,
			Path:    p.Path,
			Secrets: secretKeys,
			EnvMaps: len(envs),
		})
		return mcp.NewToolResultText(string(out)), nil
	}

	// Delete all project-scoped secrets first.
	for _, k := range secretKeys {
		if err := ss.Delete(k); err != nil {
			return nil, fmt.Errorf("deleting secret %q: %w", k, err)
		}
	}

	if err := store.Delete(name); err != nil {
		return nil, err
	}
	if state.get() == name {
		state.set("")
	}

	type result struct {
		Deleted        string   `json:"deleted"`
		SecretsDeleted []string `json:"secrets_deleted"`
	}
	out, _ := json.Marshal(result{Deleted: name, SecretsDeleted: secretKeys})
	return mcp.NewToolResultText(string(out)), nil
}

func handleListProjects(store *projects.Store, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	list, err := store.List()
	if err != nil {
		return nil, err
	}
	type envRow struct {
		SecretKey string `json:"secret_key"`
		EnvVar    string `json:"env_var"`
	}
	type row struct {
		Name string   `json:"name"`
		Path string   `json:"path"`
		Env  []envRow `json:"env,omitempty"`
	}
	rows := make([]row, len(list))
	for i, p := range list {
		envs, err := store.ListEnv(p.Name)
		if err != nil {
			return nil, err
		}
		envRows := make([]envRow, len(envs))
		for j, e := range envs {
			envRows[j] = envRow{SecretKey: e.SecretKey, EnvVar: e.EnvVar}
		}
		rows[i] = row{Name: p.Name, Path: p.Path, Env: envRows}
	}
	out, _ := json.Marshal(rows)
	return mcp.NewToolResultText(string(out)), nil
}

func handleAddProjectEnv(store *projects.Store, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	project := req.GetString("project", "")
	secretKey := req.GetString("secret_key", "")
	envVar := req.GetString("env_var", "")
	if project == "" || secretKey == "" || envVar == "" {
		return nil, fmt.Errorf("project, secret_key, and env_var are all required")
	}
	if err := store.AddEnv(project, secretKey, envVar); err != nil {
		return nil, err
	}
	return mcp.NewToolResultText(fmt.Sprintf("env mapping added: %s → %s", secretKey, envVar)), nil
}

func handleDeleteProjectEnv(store *projects.Store, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	project := req.GetString("project", "")
	secretKey := req.GetString("secret_key", "")
	if project == "" || secretKey == "" {
		return nil, fmt.Errorf("project and secret_key are required")
	}
	if err := store.DeleteEnv(project, secretKey); err != nil {
		return nil, err
	}
	return mcp.NewToolResultText(fmt.Sprintf("env mapping for %q removed from project %q", secretKey, project)), nil
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

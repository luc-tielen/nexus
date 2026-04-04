package install

import (
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"path/filepath"
)

// SettingsPath is the Claude Code user config file, resolved relative to the
// user's home directory at runtime.
const SettingsPath = ".claude.json"

// MCPBaseURL is the base URL the nexus MCP server listens on.
const MCPBaseURL = "http://localhost:7744"

// MCPEndpoint is the MCP endpoint Claude Code connects to.
const MCPEndpoint = MCPBaseURL + "/mcp"

type mcpServerConfig struct {
	Type string `json:"type"`
	URL  string `json:"url"`
}

type settings struct {
	MCPServers map[string]mcpServerConfig `json:"mcpServers,omitempty"`

	// Extra captures all other fields so they are preserved on round-trip.
	Extra map[string]json.RawMessage `json:"-"`
}

func (s *settings) UnmarshalJSON(b []byte) error {
	// Unmarshal known fields.
	type plain settings
	if err := json.Unmarshal(b, (*plain)(s)); err != nil {
		return err
	}

	// Unmarshal everything into Extra, then delete known keys.
	if err := json.Unmarshal(b, &s.Extra); err != nil {
		return err
	}
	delete(s.Extra, "mcpServers")
	return nil
}

func (s settings) MarshalJSON() ([]byte, error) {
	// Start from Extra so unknown fields are preserved.
	out := make(map[string]json.RawMessage, len(s.Extra)+1)
	maps.Copy(out, s.Extra)

	if len(s.MCPServers) > 0 {
		b, err := json.Marshal(s.MCPServers)
		if err != nil {
			return nil, err
		}
		out["mcpServers"] = b
	}

	return json.Marshal(out)
}

// defaultConfigYAML is written to config.yaml on first install.
const defaultConfigYAML = `# Optional Claude CLI arguments prepended to every nexus invocation.
# claude_args:
#   - --model
#   - claude-opus-4-6
`

// Install writes the nexus MCP server entry into ~/.claude.json and creates a
// default config.yaml in dataDir, both only if they do not already exist.
func Install(dataDir string) error {
	configPath := filepath.Join(dataDir, "config.yaml")
	created, err := initConfigFile(dataDir)
	if err != nil {
		return err
	}
	if created {
		fmt.Fprintf(os.Stderr, "nexus: wrote default config to %s\n", configPath)
	} else {
		fmt.Fprintf(os.Stderr, "nexus: config already exists at %s\n", configPath)
	}

	data, err := readSettings()
	if err != nil {
		return err
	}

	if data.MCPServers == nil {
		data.MCPServers = map[string]mcpServerConfig{}
	}
	data.MCPServers["nexus"] = mcpServerConfig{
		Type: "http",
		URL:  MCPEndpoint,
	}

	if err := writeSettings(data); err != nil {
		return err
	}

	settingsPath, _ := settingsPath()
	fmt.Fprintf(os.Stderr, "nexus: wrote MCP server config to %s\n", settingsPath)
	return nil
}

// initConfigFile creates dataDir and writes a default config.yaml into it if
// the file does not already exist. It returns true if the file was created.
func initConfigFile(dataDir string) (created bool, err error) {
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		return false, fmt.Errorf("creating config directory: %w", err)
	}

	path := filepath.Join(dataDir, "config.yaml")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if os.IsExist(err) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("creating %s: %w", path, err)
	}
	defer func() {
		if cerr := f.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("closing %s: %w", path, cerr)
		}
	}()

	if _, err = f.WriteString(defaultConfigYAML); err != nil {
		return false, fmt.Errorf("writing %s: %w", path, err)
	}

	return true, nil
}

func settingsPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}
	return filepath.Join(home, SettingsPath), nil
}

func readSettings() (settings, error) {
	path, err := settingsPath()
	if err != nil {
		return settings{}, err
	}
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return settings{}, nil
	}
	if err != nil {
		return settings{}, fmt.Errorf("reading %s: %w", path, err)
	}

	var data settings
	if err := json.Unmarshal(raw, &data); err != nil {
		return settings{}, fmt.Errorf("parsing %s: %w", path, err)
	}
	return data, nil
}

func writeSettings(data settings) error {
	path, err := settingsPath()
	if err != nil {
		return err
	}

	out, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	out = append(out, '\n')

	return os.WriteFile(path, out, 0o644)
}

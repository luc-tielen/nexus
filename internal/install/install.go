package install

import (
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"path/filepath"
)

const SettingsPath = ".claude/settings.json"

// MCPBaseURL is the base URL the nexus MCP server listens on.
const MCPBaseURL = "http://localhost:7744"

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

// Install writes the nexus MCP server entry into .claude/settings.json in the
// current directory, creating the file if it does not exist.
func Install() error {
	data, err := readSettings()
	if err != nil {
		return err
	}

	if data.MCPServers == nil {
		data.MCPServers = map[string]mcpServerConfig{}
	}
	data.MCPServers["nexus"] = mcpServerConfig{
		Type: "http",
		URL:  MCPBaseURL + "/mcp",
	}

	if err := writeSettings(data); err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "nexus: wrote MCP server config to %s\n", SettingsPath)
	return nil
}

func readSettings() (settings, error) {
	raw, err := os.ReadFile(SettingsPath)
	if os.IsNotExist(err) {
		return settings{}, nil
	}
	if err != nil {
		return settings{}, fmt.Errorf("reading %s: %w", SettingsPath, err)
	}

	var data settings
	if err := json.Unmarshal(raw, &data); err != nil {
		return settings{}, fmt.Errorf("parsing %s: %w", SettingsPath, err)
	}
	return data, nil
}

func writeSettings(data settings) error {
	if err := os.MkdirAll(filepath.Dir(SettingsPath), 0o755); err != nil {
		return fmt.Errorf("creating %s: %w", filepath.Dir(SettingsPath), err)
	}

	out, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	out = append(out, '\n')

	return os.WriteFile(SettingsPath, out, 0o644)
}

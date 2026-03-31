package install

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const SettingsPath = ".claude/settings.json"

// MCPBaseURL is the base URL the nexus MCP server listens on.
const MCPBaseURL = "http://localhost:7744"

// Install writes the nexus MCP server entry into .claude/settings.json in the
// current directory, creating the file if it does not exist.
func Install() error {
	data, err := readSettings()
	if err != nil {
		return err
	}

	// Ensure mcpServers map exists.
	mcpServers, _ := data["mcpServers"].(map[string]any)
	if mcpServers == nil {
		mcpServers = map[string]any{}
	}

	mcpServers["nexus"] = map[string]any{
		"type": "sse",
		"url":  MCPBaseURL + "/sse",
	}
	data["mcpServers"] = mcpServers

	if err := writeSettings(data); err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "nexus: wrote MCP server config to %s\n", SettingsPath)
	return nil
}

func readSettings() (map[string]any, error) {
	raw, err := os.ReadFile(SettingsPath)
	if os.IsNotExist(err) {
		return map[string]any{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", SettingsPath, err)
	}

	var data map[string]any
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", SettingsPath, err)
	}
	return data, nil
}

func writeSettings(data map[string]any) error {
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

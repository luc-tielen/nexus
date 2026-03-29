package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const settingsPath = ".claude/settings.json"

// install writes the nexus MCP server entry into .claude/settings.json in the
// current directory, creating the file if it does not exist.
func install() error {
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
		"url":  mcpBaseURL + "/sse",
	}
	data["mcpServers"] = mcpServers

	if err := writeSettings(data); err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "nexus: wrote MCP server config to %s\n", settingsPath)
	return nil
}

func readSettings() (map[string]any, error) {
	raw, err := os.ReadFile(settingsPath)
	if os.IsNotExist(err) {
		return map[string]any{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", settingsPath, err)
	}

	var data map[string]any
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", settingsPath, err)
	}
	return data, nil
}

func writeSettings(data map[string]any) error {
	if err := os.MkdirAll(filepath.Dir(settingsPath), 0o755); err != nil {
		return fmt.Errorf("creating %s: %w", filepath.Dir(settingsPath), err)
	}

	out, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	out = append(out, '\n')

	return os.WriteFile(settingsPath, out, 0o644)
}

package install

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestInstall_CreatesFileFromScratch(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)

	if err := Install(); err != nil {
		t.Fatalf("install: %v", err)
	}

	data := readSettingsFile(t, dir)
	assertMCPEntry(t, data)
}

func TestInstall_Idempotent(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)

	if err := Install(); err != nil {
		t.Fatalf("first install: %v", err)
	}
	if err := Install(); err != nil {
		t.Fatalf("second install: %v", err)
	}

	data := readSettingsFile(t, dir)
	assertMCPEntry(t, data)

	// mcpServers must have exactly one entry.
	mcpServers, ok := data["mcpServers"].(map[string]any)
	if !ok {
		t.Fatal("mcpServers is not an object")
	}
	if len(mcpServers) != 1 {
		t.Errorf("expected 1 mcpServer entry, got %d", len(mcpServers))
	}
}

func TestInstall_PreservesExistingContent(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)

	// Write a settings.json that already has hooks config.
	existing := map[string]any{
		"hooks": map[string]any{
			"Stop": []any{"some-hook"},
		},
	}
	writeSettingsFile(t, dir, existing)

	if err := Install(); err != nil {
		t.Fatalf("install: %v", err)
	}

	data := readSettingsFile(t, dir)
	assertMCPEntry(t, data)

	// Existing hooks must be preserved.
	if _, ok := data["hooks"]; !ok {
		t.Error("install removed existing hooks key")
	}
}

func TestInstall_PreservesExistingMCPServers(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)

	existing := map[string]any{
		"mcpServers": map[string]any{
			"other": map[string]any{"type": "sse", "url": "http://example.com/sse"},
		},
	}
	writeSettingsFile(t, dir, existing)

	if err := Install(); err != nil {
		t.Fatalf("install: %v", err)
	}

	data := readSettingsFile(t, dir)
	assertMCPEntry(t, data)

	mcpServers := data["mcpServers"].(map[string]any)
	if _, ok := mcpServers["other"]; !ok {
		t.Error("install removed existing mcpServer entry")
	}
}

// helpers

func chdir(t *testing.T, dir string) {
	t.Helper()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(orig) })
}

func readSettingsFile(t *testing.T, dir string) map[string]any {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(dir, SettingsPath))
	if err != nil {
		t.Fatalf("reading settings: %v", err)
	}
	var data map[string]any
	if err := json.Unmarshal(raw, &data); err != nil {
		t.Fatalf("parsing settings: %v", err)
	}
	return data
}

func writeSettingsFile(t *testing.T, dir string, data map[string]any) {
	t.Helper()
	path := filepath.Join(dir, SettingsPath)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	raw, _ := json.MarshalIndent(data, "", "  ")
	if err := os.WriteFile(path, append(raw, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
}

func assertMCPEntry(t *testing.T, data map[string]any) {
	t.Helper()
	mcpServers, ok := data["mcpServers"].(map[string]any)
	if !ok {
		t.Fatal("mcpServers key missing or not an object")
	}
	nexus, ok := mcpServers["nexus"].(map[string]any)
	if !ok {
		t.Fatal("mcpServers.nexus missing or not an object")
	}
	if nexus["type"] != "sse" {
		t.Errorf("type = %q, want %q", nexus["type"], "sse")
	}
	wantURL := MCPBaseURL + "/sse"
	if nexus["url"] != wantURL {
		t.Errorf("url = %q, want %q", nexus["url"], wantURL)
	}
}

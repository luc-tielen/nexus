package install

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func dataDir(t *testing.T, home string) string {
	t.Helper()
	dir := filepath.Join(home, ".config", "nexus-ai")
	return dir
}

func TestInstall_CreatesFileFromScratch(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	if err := Install(dataDir(t, dir)); err != nil {
		t.Fatalf("install: %v", err)
	}

	data := readSettingsFile(t, dir)
	assertMCPEntry(t, data)
}

func TestInstall_CreatesDefaultConfig(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	if err := Install(dataDir(t, dir)); err != nil {
		t.Fatalf("install: %v", err)
	}

	path := filepath.Join(dataDir(t, dir), "config.yaml")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("config.yaml not created: %v", err)
	}
}

func TestInstall_DoesNotOverwriteExistingConfig(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	cfgDir := dataDir(t, dir)
	if err := os.MkdirAll(cfgDir, 0o700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(cfgDir, "config.yaml")
	existing := []byte("# existing\n")
	if err := os.WriteFile(path, existing, 0o644); err != nil {
		t.Fatal(err)
	}

	if err := Install(cfgDir); err != nil {
		t.Fatalf("install: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(existing) {
		t.Errorf("config.yaml was overwritten: got %q", got)
	}
}

func TestInstall_Idempotent(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	if err := Install(dataDir(t, dir)); err != nil {
		t.Fatalf("first install: %v", err)
	}
	if err := Install(dataDir(t, dir)); err != nil {
		t.Fatalf("second install: %v", err)
	}

	data := readSettingsFile(t, dir)
	assertMCPEntry(t, data)

	// mcpServers must have exactly one entry.
	if len(data.MCPServers) != 1 {
		t.Errorf("expected 1 mcpServer entry, got %d", len(data.MCPServers))
	}
}

func TestInstall_PreservesExistingContent(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	// Write a settings.json that already has hooks config.
	hooksJSON, _ := json.Marshal(map[string]any{"Stop": []any{"some-hook"}})
	existing := settings{
		Extra: map[string]json.RawMessage{
			"hooks": hooksJSON,
		},
	}
	writeSettingsFile(t, dir, existing)

	if err := Install(dataDir(t, dir)); err != nil {
		t.Fatalf("install: %v", err)
	}

	data := readSettingsFile(t, dir)
	assertMCPEntry(t, data)

	// Existing hooks must be preserved.
	if _, ok := data.Extra["hooks"]; !ok {
		t.Error("install removed existing hooks key")
	}
}

func TestInstall_PreservesExistingMCPServers(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	existing := settings{
		MCPServers: map[string]mcpServerConfig{
			"other": {Type: "sse", URL: "http://example.com/sse"},
		},
	}
	writeSettingsFile(t, dir, existing)

	if err := Install(dataDir(t, dir)); err != nil {
		t.Fatalf("install: %v", err)
	}

	data := readSettingsFile(t, dir)
	assertMCPEntry(t, data)

	if _, ok := data.MCPServers["other"]; !ok {
		t.Error("install removed existing mcpServer entry")
	}
}

// helpers

func readSettingsFile(t *testing.T, dir string) settings {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(dir, SettingsPath))
	if err != nil {
		t.Fatalf("reading settings: %v", err)
	}
	var data settings
	if err := json.Unmarshal(raw, &data); err != nil {
		t.Fatalf("parsing settings: %v", err)
	}
	return data
}

func writeSettingsFile(t *testing.T, dir string, data settings) {
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

func assertMCPEntry(t *testing.T, data settings) {
	t.Helper()
	nexus, ok := data.MCPServers["nexus"]
	if !ok {
		t.Fatal("mcpServers.nexus missing")
	}
	if nexus.Type != "http" {
		t.Errorf("type = %q, want %q", nexus.Type, "http")
	}
	if nexus.URL != MCPEndpoint {
		t.Errorf("url = %q, want %q", nexus.URL, MCPEndpoint)
	}
}

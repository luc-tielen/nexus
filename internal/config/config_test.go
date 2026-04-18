package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoad_missing(t *testing.T) {
	cfg, err := Load(t.TempDir())
	if err != nil {
		t.Fatalf("expected no error for missing file, got: %v", err)
	}
	if len(cfg.ClaudeArgs) != 0 {
		t.Fatalf("expected empty ClaudeArgs, got %v", cfg.ClaudeArgs)
	}
}

func TestParse(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    Config
		wantErr string
	}{
		{
			name:  "empty file",
			input: "",
			want:  Config{},
		},
		{
			name:  "claude_args",
			input: "claude_args:\n  - --model\n  - claude-opus-4-6\n",
			want:  Config{ClaudeArgs: []string{"--model", "claude-opus-4-6"}},
		},
		{
			name:  "channel telegram",
			input: "channel: telegram\n",
			want:  Config{Channel: "telegram"},
		},
		{
			name:  "channel telegram_nexus",
			input: "channel: telegram_nexus\n",
			want:  Config{Channel: "telegram_nexus"},
		},
		{
			name:    "channel unsupported",
			input:   "channel: discord\n",
			wantErr: `channel: unsupported value "discord"`,
		},
		{
			name:    "unknown key",
			input:   "unknown_key: value\n",
			wantErr: "field unknown_key not found",
		},
		{
			name:    "empty arg entry",
			input:   "claude_args:\n  - --model\n  - \"\"\n",
			wantErr: "claude_args[1]: empty string is not allowed",
		},
		{
			name:    "invalid yaml",
			input:   "claude_args: [\n",
			wantErr: "did not find expected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := parse([]byte(tt.input))
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("expected error containing %q, got %q", tt.wantErr, err.Error())
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(cfg.ClaudeArgs) != len(tt.want.ClaudeArgs) {
				t.Fatalf("ClaudeArgs = %v, want %v", cfg.ClaudeArgs, tt.want.ClaudeArgs)
			}
			for i := range cfg.ClaudeArgs {
				if cfg.ClaudeArgs[i] != tt.want.ClaudeArgs[i] {
					t.Errorf("ClaudeArgs[%d] = %q, want %q", i, cfg.ClaudeArgs[i], tt.want.ClaudeArgs[i])
				}
			}
			if cfg.Channel != tt.want.Channel {
				t.Errorf("Channel = %q, want %q", cfg.Channel, tt.want.Channel)
			}
		})
	}
}

func TestLoad_errorWrapsPath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte("bad: key\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := Load(dir)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), path) {
		t.Errorf("expected error to contain path %q, got %q", path, err.Error())
	}
}

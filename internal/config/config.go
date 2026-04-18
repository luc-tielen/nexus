package config

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config holds user-level nexus configuration loaded from config.yaml.
type Config struct {
	// ClaudeArgs are extra arguments prepended to every `nexus` invocation.
	ClaudeArgs []string `yaml:"claude_args"`

	// Channel is an optional communication channel to enable.
	// When set, the required Claude plugin args are added to the main
	// interactive Claude process only — never to background subprocesses.
	// Supported values: "telegram" (uses Claude plugin), "telegram_nexus" (built-in listener).
	Channel string `yaml:"channel"`
}

// Load reads ~/.config/nexus-ai/config.yaml. If the file does not exist an empty
// Config is returned with no error.
func Load(dataDir string) (Config, error) {
	path := filepath.Join(dataDir, "config.yaml")
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return Config{}, nil
	}
	if err != nil {
		return Config{}, fmt.Errorf("reading %s: %w", path, err)
	}

	cfg, err := parse(raw)
	if err != nil {
		return Config{}, fmt.Errorf("%s: %w", path, err)
	}
	return cfg, nil
}

func parse(raw []byte) (Config, error) {
	dec := yaml.NewDecoder(bytes.NewReader(raw))
	dec.KnownFields(true)

	var cfg Config
	if err := dec.Decode(&cfg); errors.Is(err, io.EOF) {
		return Config{}, nil
	} else if err != nil {
		return Config{}, err
	}

	for i, arg := range cfg.ClaudeArgs {
		if arg == "" {
			return Config{}, fmt.Errorf("claude_args[%d]: empty string is not allowed", i)
		}
	}

	switch cfg.Channel {
	case "", "telegram", "telegram_nexus":
		// valid
	default:
		return Config{}, fmt.Errorf("channel: unsupported value %q (supported: telegram, telegram_nexus)", cfg.Channel)
	}

	return cfg, nil
}

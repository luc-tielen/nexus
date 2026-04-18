package telegram

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func transcribeWithWhisper(ctx context.Context, audioPath string) (string, error) {
	tmpDir, err := os.MkdirTemp("", "nexus_whisper_*")
	if err != nil {
		return "", fmt.Errorf("creating temp dir: %w", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	cmd := exec.CommandContext(ctx, "whisper", audioPath,
		"--model", "tiny",
		"--output-format", "txt",
		"--output_dir", tmpDir,
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("whisper: %w\n%s", err, out)
	}

	base := filepath.Base(audioPath)
	txtFile := filepath.Join(tmpDir, strings.TrimSuffix(base, filepath.Ext(base))+".txt")

	data, err := os.ReadFile(txtFile)
	if err != nil {
		return "", fmt.Errorf("reading whisper output: %w", err)
	}

	return strings.TrimSpace(string(data)), nil
}

func httpDownloadFile(ctx context.Context, url string) (string, func(), error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", nil, fmt.Errorf("creating request: %w", err)
	}

	f, err := os.CreateTemp("", "nexus_voice_*.ogg")
	if err != nil {
		return "", nil, fmt.Errorf("creating temp file: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		_ = f.Close()
		_ = os.Remove(f.Name())
		return "", nil, fmt.Errorf("downloading: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if _, err := io.Copy(f, resp.Body); err != nil {
		_ = f.Close()
		_ = os.Remove(f.Name())
		return "", nil, fmt.Errorf("saving: %w", err)
	}
	_ = f.Close()

	name := f.Name()
	return name, func() { _ = os.Remove(name) }, nil
}

package telegram

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Updater is the subset of the Telegram bot API used for receiving updates.
type Updater interface {
	GetUpdates(config tgbotapi.UpdateConfig) ([]tgbotapi.Update, error)
	GetFileDirectURL(fileID string) (string, error)
}

// Listener listens for incoming Telegram messages and injects them into the PTY.
type Listener struct {
	Bot          Updater
	Transcribe   func(ctx context.Context, audioPath string) (string, error)
	downloadFile func(ctx context.Context, url string) (path string, cleanup func(), err error)
}

// killStalePluginPoller sends SIGTERM to the process recorded in the Claude
// Telegram plugin's PID file. An orphaned plugin server holds the getUpdates
// slot indefinitely, causing 409 Conflicts for any new listener.
func killStalePluginPoller() {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	data, err := os.ReadFile(filepath.Join(home, ".claude", "channels", "telegram", "bot.pid"))
	if err != nil {
		return
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil || pid <= 1 {
		return
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return
	}
	if proc.Signal(syscall.Signal(0)) != nil {
		return // process is already gone
	}
	_ = proc.Signal(syscall.SIGTERM)
	time.Sleep(time.Second)
}

// NewListener creates a Listener for the given bot token.
// It accepts messages from all chats the bot is a member of.
// It makes a network request to validate the token.
func NewListener(token string) (*Listener, error) {
	if token == "" {
		return nil, fmt.Errorf("TELEGRAM_BOT_TOKEN is not set")
	}
	killStalePluginPoller()
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, fmt.Errorf("creating Telegram listener bot: %w", err)
	}
	return &Listener{
		Bot:          bot,
		Transcribe:   transcribeWithWhisper,
		downloadFile: httpDownloadFile,
	}, nil
}

// Listen blocks until ctx is cancelled, calling inject for each incoming message.
// Voice messages are transcribed via Whisper before injection.
// Errors are silently retried — never written to stdout/stderr to avoid
// corrupting the PTY display.
func (l *Listener) Listen(ctx context.Context, inject func(string)) error {
	cfg := tgbotapi.NewUpdate(0)
	cfg.Timeout = 1 // Short timeout so stale connections from a previous run expire in ~1s on restart

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Run GetUpdates in a goroutine so context cancellation is not blocked
		// by a long-polling HTTP request.
		type pollResult struct {
			updates []tgbotapi.Update
			err     error
		}
		ch := make(chan pollResult, 1)
		go func() {
			updates, err := l.Bot.GetUpdates(cfg)
			ch <- pollResult{updates, err}
		}()

		var res pollResult
		select {
		case <-ctx.Done():
			return ctx.Err()
		case res = <-ch:
		}

		if res.err != nil {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(200 * time.Millisecond):
			}
			continue
		}

		for _, update := range res.updates {
			cfg.Offset = update.UpdateID + 1
			if update.Message == nil {
				continue
			}
			_ = l.handleUpdate(ctx, update.Message, inject)
		}
	}
}

func (l *Listener) handleUpdate(ctx context.Context, msg *tgbotapi.Message, inject func(string)) error {
	prefix := fmt.Sprintf("[Telegram(chat_id=%d)]: ", msg.Chat.ID)
	if msg.Voice != nil {
		return l.handleVoice(ctx, msg.Voice.FileID, prefix, inject)
	}
	if msg.Text != "" {
		inject(prefix + msg.Text + "\r")
	}
	return nil
}

func (l *Listener) handleVoice(ctx context.Context, fileID, prefix string, inject func(string)) error {
	url, err := l.Bot.GetFileDirectURL(fileID)
	if err != nil {
		return fmt.Errorf("getting voice file URL: %w", err)
	}

	path, cleanup, err := l.downloadFile(ctx, url)
	if err != nil {
		return fmt.Errorf("downloading voice: %w", err)
	}
	defer cleanup()

	text, err := l.Transcribe(ctx, path)
	if err != nil {
		return fmt.Errorf("transcribing voice: %w", err)
	}

	if text != "" {
		inject(prefix + text + "\r")
	}
	return nil
}

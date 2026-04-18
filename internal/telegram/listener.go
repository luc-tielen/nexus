package telegram

import (
	"context"
	"fmt"
	"strconv"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Updater is the subset of the Telegram bot API used for receiving updates.
type Updater interface {
	GetUpdatesChan(config tgbotapi.UpdateConfig) tgbotapi.UpdatesChannel
	GetFileDirectURL(fileID string) (string, error)
	StopReceivingUpdates()
}

// Listener listens for incoming Telegram messages and injects them into the PTY.
type Listener struct {
	Bot          Updater
	ChatID       int64
	Transcribe   func(ctx context.Context, audioPath string) (string, error)
	downloadFile func(ctx context.Context, url string) (path string, cleanup func(), err error)
}

// NewListener creates a Listener for the given bot token and chat ID.
// It makes a network request to validate the token.
func NewListener(token, chatID string) (*Listener, error) {
	if token == "" {
		return nil, fmt.Errorf("TELEGRAM_BOT_TOKEN is not set")
	}
	if chatID == "" {
		return nil, fmt.Errorf("TELEGRAM_CHAT_ID is not set")
	}
	id, err := strconv.ParseInt(chatID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("TELEGRAM_CHAT_ID must be a numeric chat ID: %w", err)
	}
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, fmt.Errorf("creating Telegram listener bot: %w", err)
	}
	return &Listener{
		Bot:          bot,
		ChatID:       id,
		Transcribe:   transcribeWithWhisper,
		downloadFile: httpDownloadFile,
	}, nil
}

// Listen blocks until ctx is cancelled, calling inject for each incoming message.
// Voice messages are transcribed via Whisper before injection.
func (l *Listener) Listen(ctx context.Context, inject func(string)) error {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 30
	updates := l.Bot.GetUpdatesChan(u)
	defer l.Bot.StopReceivingUpdates()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case update, ok := <-updates:
			if !ok {
				return nil
			}
			if update.Message == nil || update.Message.Chat.ID != l.ChatID {
				continue
			}
			// Errors are silently dropped: the terminal is the foreground
			// process and writing to stderr/stdout would corrupt the display.
			_ = l.handleUpdate(ctx, update.Message, inject)
		}
	}
}

func (l *Listener) handleUpdate(ctx context.Context, msg *tgbotapi.Message, inject func(string)) error {
	if msg.Voice != nil {
		return l.handleVoice(ctx, msg.Voice.FileID, inject)
	}
	if msg.Text != "" {
		inject(msg.Text + "\r")
	}
	return nil
}

func (l *Listener) handleVoice(ctx context.Context, fileID string, inject func(string)) error {
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
		inject(text + "\r")
	}
	return nil
}

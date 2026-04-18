package telegram

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// fakeUpdater is a test double for Updater.
type fakeUpdater struct {
	updates chan tgbotapi.Update
	fileURL string
	fileErr error
}

func (f *fakeUpdater) GetUpdatesChan(_ tgbotapi.UpdateConfig) tgbotapi.UpdatesChannel {
	return f.updates
}

func (f *fakeUpdater) GetFileDirectURL(_ string) (string, error) {
	return f.fileURL, f.fileErr
}

func (f *fakeUpdater) StopReceivingUpdates() {}

func newFakeUpdater() *fakeUpdater {
	return &fakeUpdater{updates: make(chan tgbotapi.Update, 8)}
}

// fakeDownload writes a temp file and returns its path without making network calls.
func fakeDownload(_ context.Context, _ string) (string, func(), error) {
	f, err := os.CreateTemp("", "nexus_test_voice_*.ogg")
	if err != nil {
		return "", nil, err
	}
	_ = f.Close()
	return f.Name(), func() { _ = os.Remove(f.Name()) }, nil
}

const testChatID = int64(42)

func newTestListener(bot *fakeUpdater, transcribe func(context.Context, string) (string, error)) *Listener {
	if transcribe == nil {
		transcribe = func(_ context.Context, _ string) (string, error) { return "", nil }
	}
	return &Listener{
		Bot:          bot,
		ChatID:       testChatID,
		Transcribe:   transcribe,
		downloadFile: fakeDownload,
	}
}

func awaitInject(t *testing.T, ch <-chan string, want string) {
	t.Helper()
	select {
	case got := <-ch:
		if got != want {
			t.Errorf("inject got %q, want %q", got, want)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for injected message")
	}
}

func TestListen_TextMessage(t *testing.T) {
	bot := newFakeUpdater()
	injected := make(chan string, 1)

	l := newTestListener(bot, nil)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() { _ = l.Listen(ctx, func(s string) { injected <- s }) }()

	bot.updates <- tgbotapi.Update{
		Message: &tgbotapi.Message{
			Chat: &tgbotapi.Chat{ID: testChatID},
			Text: "hello world",
		},
	}

	awaitInject(t, injected, "hello world\r")
}

func TestListen_FiltersByChat(t *testing.T) {
	bot := newFakeUpdater()
	injected := make(chan string, 1)

	l := newTestListener(bot, nil)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() { _ = l.Listen(ctx, func(s string) { injected <- s }) }()

	bot.updates <- tgbotapi.Update{
		Message: &tgbotapi.Message{
			Chat: &tgbotapi.Chat{ID: 99},
			Text: "wrong chat",
		},
	}
	bot.updates <- tgbotapi.Update{
		Message: &tgbotapi.Message{
			Chat: &tgbotapi.Chat{ID: testChatID},
			Text: "right chat",
		},
	}

	awaitInject(t, injected, "right chat\r")

	select {
	case extra := <-injected:
		t.Errorf("unexpected extra inject: %q", extra)
	default:
	}
}

func TestListen_EmptyText(t *testing.T) {
	bot := newFakeUpdater()
	injected := make(chan string, 1)

	l := newTestListener(bot, nil)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() { _ = l.Listen(ctx, func(s string) { injected <- s }) }()

	bot.updates <- tgbotapi.Update{
		Message: &tgbotapi.Message{
			Chat: &tgbotapi.Chat{ID: testChatID},
			Text: "",
		},
	}

	select {
	case s := <-injected:
		t.Errorf("expected no inject for empty text, got %q", s)
	case <-time.After(100 * time.Millisecond):
		// expected: empty text is ignored
	}
}

func TestListen_ContextCancellation(t *testing.T) {
	bot := newFakeUpdater()

	l := newTestListener(bot, nil)
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() { done <- l.Listen(ctx, func(_ string) {}) }()

	cancel()

	select {
	case err := <-done:
		if err != context.Canceled {
			t.Errorf("expected context.Canceled, got %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for listener to stop")
	}
}

func TestListen_UpdatesChannelClosed(t *testing.T) {
	bot := newFakeUpdater()

	l := newTestListener(bot, nil)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan error, 1)
	go func() { done <- l.Listen(ctx, func(_ string) {}) }()

	close(bot.updates)

	select {
	case err := <-done:
		if err != nil {
			t.Errorf("expected nil on closed channel, got %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for listener to stop")
	}
}

func TestListen_VoiceMessage(t *testing.T) {
	bot := newFakeUpdater()
	bot.fileURL = "http://fake/voice.ogg"
	injected := make(chan string, 1)

	transcribeCalled := false
	l := newTestListener(bot, func(_ context.Context, path string) (string, error) {
		transcribeCalled = true
		if path == "" {
			t.Error("expected non-empty audio path")
		}
		return "transcribed text", nil
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() { _ = l.Listen(ctx, func(s string) { injected <- s }) }()

	bot.updates <- tgbotapi.Update{
		Message: &tgbotapi.Message{
			Chat:  &tgbotapi.Chat{ID: testChatID},
			Voice: &tgbotapi.Voice{FileID: "file123"},
		},
	}

	awaitInject(t, injected, "transcribed text\r")

	if !transcribeCalled {
		t.Error("expected Transcribe to be called")
	}
}

func TestListen_VoiceEmptyTranscription(t *testing.T) {
	bot := newFakeUpdater()
	bot.fileURL = "http://fake/voice.ogg"
	injected := make(chan string, 1)

	l := newTestListener(bot, func(_ context.Context, _ string) (string, error) {
		return "", nil
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() { _ = l.Listen(ctx, func(s string) { injected <- s }) }()

	bot.updates <- tgbotapi.Update{
		Message: &tgbotapi.Message{
			Chat:  &tgbotapi.Chat{ID: testChatID},
			Voice: &tgbotapi.Voice{FileID: "file123"},
		},
	}

	select {
	case s := <-injected:
		t.Errorf("expected no inject for empty transcription, got %q", s)
	case <-time.After(100 * time.Millisecond):
		// expected: empty transcription is ignored
	}
}

func TestNewListener_Validation(t *testing.T) {
	tests := []struct {
		name    string
		token   string
		chatID  string
		wantErr string
	}{
		{"missing token", "", "123", "TELEGRAM_BOT_TOKEN is not set"},
		{"missing chat ID", "tok", "", "TELEGRAM_CHAT_ID is not set"},
		{"non-numeric chat ID", "tok", "abc", "must be a numeric chat ID"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewListener(tt.token, tt.chatID)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error %q does not contain %q", err.Error(), tt.wantErr)
			}
		})
	}
}

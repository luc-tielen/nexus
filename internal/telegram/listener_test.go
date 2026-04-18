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
// GetUpdates blocks until a batch is pushed via send() or the channel is closed.
type fakeUpdater struct {
	batches chan []tgbotapi.Update
	fileURL string
	fileErr error
}

func newFakeUpdater() *fakeUpdater {
	return &fakeUpdater{batches: make(chan []tgbotapi.Update, 8)}
}

func (f *fakeUpdater) GetUpdates(_ tgbotapi.UpdateConfig) ([]tgbotapi.Update, error) {
	batch, ok := <-f.batches
	if !ok {
		return nil, nil
	}
	return batch, nil
}

func (f *fakeUpdater) GetFileDirectURL(_ string) (string, error) {
	return f.fileURL, f.fileErr
}

func (f *fakeUpdater) send(updates ...tgbotapi.Update) {
	f.batches <- updates
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
	t.Cleanup(func() { close(bot.batches) })
	injected := make(chan string, 1)

	l := newTestListener(bot, nil)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() { _ = l.Listen(ctx, func(s string) { injected <- s }) }()

	bot.send(tgbotapi.Update{
		Message: &tgbotapi.Message{
			Chat: &tgbotapi.Chat{ID: testChatID},
			Text: "hello world",
		},
	})

	awaitInject(t, injected, "[Telegram(chat_id=42)]: hello world\r")
}

func TestListen_MultipleChats(t *testing.T) {
	bot := newFakeUpdater()
	t.Cleanup(func() { close(bot.batches) })
	injected := make(chan string, 2)

	l := newTestListener(bot, nil)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() { _ = l.Listen(ctx, func(s string) { injected <- s }) }()

	bot.send(
		tgbotapi.Update{Message: &tgbotapi.Message{Chat: &tgbotapi.Chat{ID: 99}, Text: "from chat 99"}},
		tgbotapi.Update{Message: &tgbotapi.Message{Chat: &tgbotapi.Chat{ID: testChatID}, Text: "from chat 42"}},
	)

	awaitInject(t, injected, "[Telegram(chat_id=99)]: from chat 99\r")
	awaitInject(t, injected, "[Telegram(chat_id=42)]: from chat 42\r")
}

func TestListen_TextMessageTrailingNewline(t *testing.T) {
	bot := newFakeUpdater()
	t.Cleanup(func() { close(bot.batches) })
	injected := make(chan string, 1)

	l := newTestListener(bot, nil)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() { _ = l.Listen(ctx, func(s string) { injected <- s }) }()

	bot.send(tgbotapi.Update{
		Message: &tgbotapi.Message{
			Chat: &tgbotapi.Chat{ID: testChatID},
			Text: "hello world\n",
		},
	})

	awaitInject(t, injected, "[Telegram(chat_id=42)]: hello world\r")
}

func TestListen_MultilineMessage(t *testing.T) {
	bot := newFakeUpdater()
	t.Cleanup(func() { close(bot.batches) })
	injected := make(chan string, 1)

	l := newTestListener(bot, nil)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() { _ = l.Listen(ctx, func(s string) { injected <- s }) }()

	bot.send(tgbotapi.Update{
		Message: &tgbotapi.Message{
			Chat: &tgbotapi.Chat{ID: testChatID},
			Text: "line one\nline two\nline three",
		},
	})

	// Multi-line input needs two Enter presses: the first \r lands on the last
	// content line (adding a trailing newline), the second \r lands on the
	// resulting empty line and submits.
	awaitInject(t, injected, "[Telegram(chat_id=42)]: line one\nline two\nline three\r\r")
}

func TestListen_EmptyText(t *testing.T) {
	bot := newFakeUpdater()
	t.Cleanup(func() { close(bot.batches) })
	injected := make(chan string, 1)

	l := newTestListener(bot, nil)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() { _ = l.Listen(ctx, func(s string) { injected <- s }) }()

	bot.send(tgbotapi.Update{
		Message: &tgbotapi.Message{Chat: &tgbotapi.Chat{ID: testChatID}, Text: ""},
	})

	select {
	case s := <-injected:
		t.Errorf("expected no inject for empty text, got %q", s)
	case <-time.After(100 * time.Millisecond):
		// expected: empty text is ignored
	}
}

func TestListen_ContextCancellation(t *testing.T) {
	bot := newFakeUpdater()
	t.Cleanup(func() { close(bot.batches) })

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

func TestListen_VoiceMessage(t *testing.T) {
	bot := newFakeUpdater()
	t.Cleanup(func() { close(bot.batches) })
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

	bot.send(tgbotapi.Update{
		Message: &tgbotapi.Message{
			Chat:  &tgbotapi.Chat{ID: testChatID},
			Voice: &tgbotapi.Voice{FileID: "file123"},
		},
	})

	awaitInject(t, injected, "[Telegram(chat_id=42)]: transcribed text\r")

	if !transcribeCalled {
		t.Error("expected Transcribe to be called")
	}
}

func TestListen_VoiceEmptyTranscription(t *testing.T) {
	bot := newFakeUpdater()
	t.Cleanup(func() { close(bot.batches) })
	bot.fileURL = "http://fake/voice.ogg"
	injected := make(chan string, 1)

	l := newTestListener(bot, func(_ context.Context, _ string) (string, error) {
		return "", nil
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() { _ = l.Listen(ctx, func(s string) { injected <- s }) }()

	bot.send(tgbotapi.Update{
		Message: &tgbotapi.Message{
			Chat:  &tgbotapi.Chat{ID: testChatID},
			Voice: &tgbotapi.Voice{FileID: "file123"},
		},
	})

	select {
	case s := <-injected:
		t.Errorf("expected no inject for empty transcription, got %q", s)
	case <-time.After(100 * time.Millisecond):
		// expected: empty transcription is ignored
	}
}

func TestNewListener_MissingToken(t *testing.T) {
	_, err := NewListener("")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "TELEGRAM_BOT_TOKEN is not set") {
		t.Errorf("unexpected error: %v", err)
	}
}

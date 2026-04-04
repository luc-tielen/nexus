package telegram

import (
	"errors"
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// fakeTelegramSender records calls to Send.
type fakeTelegramSender struct {
	sentChatID int64
	sentText   string
	err        error
}

func (f *fakeTelegramSender) Send(c tgbotapi.Chattable) (tgbotapi.Message, error) {
	if msg, ok := c.(tgbotapi.MessageConfig); ok {
		f.sentChatID = msg.ChatID
		f.sentText = msg.Text
	}
	return tgbotapi.Message{}, f.err
}

func TestClient_Send(t *testing.T) {
	fake := &fakeTelegramSender{}
	c := &Client{Bot: fake, ChatID: 12345}

	if err := c.Send("hello telegram"); err != nil {
		t.Fatalf("Send: %v", err)
	}
	if fake.sentChatID != 12345 {
		t.Errorf("chatID = %d, want %d", fake.sentChatID, 12345)
	}
	if fake.sentText != "hello telegram" {
		t.Errorf("text = %q, want %q", fake.sentText, "hello telegram")
	}
}

func TestClient_SendPropagatesError(t *testing.T) {
	fake := &fakeTelegramSender{err: errors.New("forbidden")}
	c := &Client{Bot: fake, ChatID: 12345}

	if err := c.Send("hello"); err == nil {
		t.Error("expected error, got nil")
	}
}

func TestNewClient_MissingToken(t *testing.T) {
	_, err := NewClient("", "12345")
	if err == nil {
		t.Error("expected error when token missing")
	}
}

func TestNewClient_MissingChatID(t *testing.T) {
	_, err := NewClient("tok", "")
	if err == nil {
		t.Error("expected error when chat ID missing")
	}
}

func TestNewClient_InvalidChatID(t *testing.T) {
	_, err := NewClient("tok", "not-a-number")
	if err == nil {
		t.Error("expected error for non-numeric chat ID")
	}
}

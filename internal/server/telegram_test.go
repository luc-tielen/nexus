package server

import (
	"errors"
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/luc/nexus/internal/telegram"
)

// fakeTelegramSender is a test double for telegram.Sender.
type fakeTelegramSender struct {
	err error
}

func (f *fakeTelegramSender) Send(_ tgbotapi.Chattable) (tgbotapi.Message, error) {
	return tgbotapi.Message{}, f.err
}

func TestHandleSendTelegramMessage_NilClient(t *testing.T) {
	_, err := handleSendTelegramMessage(nil, toolReq(map[string]any{"message": "hi"}))
	if err == nil {
		t.Error("expected error when client is nil")
	}
}

func TestHandleSendTelegramMessage_MissingMessage(t *testing.T) {
	tc := &telegram.Client{Bot: &fakeTelegramSender{}, ChatID: 1}

	_, err := handleSendTelegramMessage(tc, toolReq(map[string]any{}))
	if err == nil {
		t.Error("expected error for missing message")
	}
}

func TestHandleSendTelegramMessage_Success(t *testing.T) {
	tc := &telegram.Client{Bot: &fakeTelegramSender{}, ChatID: 1}

	res, err := handleSendTelegramMessage(tc, toolReq(map[string]any{"message": "hi"}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if textContent(res) != "message sent" {
		t.Errorf("got %q, want %q", textContent(res), "message sent")
	}
}

func TestHandleSendTelegramMessage_SendError(t *testing.T) {
	tc := &telegram.Client{Bot: &fakeTelegramSender{err: errors.New("forbidden")}, ChatID: 1}

	_, err := handleSendTelegramMessage(tc, toolReq(map[string]any{"message": "hi"}))
	if err == nil {
		t.Error("expected error when send fails")
	}
}

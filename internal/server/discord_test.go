package server

import (
	"errors"
	"testing"

	"github.com/bwmarrin/discordgo"
	"github.com/luc/nexus/internal/discord"
)

// fakeDiscordSender is a test double for discord.Sender.
type fakeDiscordSender struct {
	err error
}

func (f *fakeDiscordSender) ChannelMessageSend(_, _ string, _ ...discordgo.RequestOption) (*discordgo.Message, error) {
	return nil, f.err
}

func TestHandleSendDiscordMessage_NilClient(t *testing.T) {
	_, err := handleSendDiscordMessage(nil, toolReq(map[string]any{"message": "hi"}))
	if err == nil {
		t.Error("expected error when client is nil")
	}
}

func TestHandleSendDiscordMessage_MissingMessage(t *testing.T) {
	dc := &discord.Client{Session: &fakeDiscordSender{}, ChannelID: "c"}

	_, err := handleSendDiscordMessage(dc, toolReq(map[string]any{}))
	if err == nil {
		t.Error("expected error for missing message")
	}
}

func TestHandleSendDiscordMessage_Success(t *testing.T) {
	dc := &discord.Client{Session: &fakeDiscordSender{}, ChannelID: "c"}

	res, err := handleSendDiscordMessage(dc, toolReq(map[string]any{"message": "hi"}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if textContent(res) != "message sent" {
		t.Errorf("got %q, want %q", textContent(res), "message sent")
	}
}

func TestHandleSendDiscordMessage_SendError(t *testing.T) {
	dc := &discord.Client{Session: &fakeDiscordSender{err: errors.New("forbidden")}, ChannelID: "c"}

	_, err := handleSendDiscordMessage(dc, toolReq(map[string]any{"message": "hi"}))
	if err == nil {
		t.Error("expected error when send fails")
	}
}

package discord

import (
	"errors"
	"testing"

	"github.com/bwmarrin/discordgo"
)

// fakeDiscordSender records calls to ChannelMessageSend.
type fakeDiscordSender struct {
	sentChannelID string
	sentContent   string
	err           error
}

func (f *fakeDiscordSender) ChannelMessageSend(channelID, content string, _ ...discordgo.RequestOption) (*discordgo.Message, error) {
	f.sentChannelID = channelID
	f.sentContent = content
	return nil, f.err
}

func TestClient_Send(t *testing.T) {
	fake := &fakeDiscordSender{}
	c := &Client{Session: fake, ChannelID: "chan123"}

	if err := c.Send("hello discord"); err != nil {
		t.Fatalf("Send: %v", err)
	}
	if fake.sentChannelID != "chan123" {
		t.Errorf("channelID = %q, want %q", fake.sentChannelID, "chan123")
	}
	if fake.sentContent != "hello discord" {
		t.Errorf("content = %q, want %q", fake.sentContent, "hello discord")
	}
}

func TestClient_SendPropagatesError(t *testing.T) {
	fake := &fakeDiscordSender{err: errors.New("forbidden")}
	c := &Client{Session: fake, ChannelID: "chan123"}

	if err := c.Send("hello"); err == nil {
		t.Error("expected error, got nil")
	}
}

func TestNewClient_MissingToken(t *testing.T) {
	_, err := NewClient("", "chan123")
	if err == nil {
		t.Error("expected error when token missing")
	}
}

func TestNewClient_MissingChannelID(t *testing.T) {
	_, err := NewClient("tok", "")
	if err == nil {
		t.Error("expected error when channel ID missing")
	}
}

func TestNewClient_BothSet(t *testing.T) {
	c, err := NewClient("mytoken", "mychannel")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.ChannelID != "mychannel" {
		t.Errorf("ChannelID = %q, want %q", c.ChannelID, "mychannel")
	}
}

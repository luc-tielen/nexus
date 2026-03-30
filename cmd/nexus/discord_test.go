package main

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

func TestDiscordClient_Send(t *testing.T) {
	fake := &fakeDiscordSender{}
	c := &discordClient{session: fake, channelID: "chan123"}

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

func TestDiscordClient_SendPropagatesError(t *testing.T) {
	fake := &fakeDiscordSender{err: errors.New("forbidden")}
	c := &discordClient{session: fake, channelID: "chan123"}

	if err := c.Send("hello"); err == nil {
		t.Error("expected error, got nil")
	}
}

func TestNewDiscordClient_MissingToken(t *testing.T) {
	t.Setenv("DISCORD_BOT_TOKEN", "")
	t.Setenv("DISCORD_CHANNEL_ID", "chan123")

	_, err := newDiscordClient()
	if err == nil {
		t.Error("expected error when token missing")
	}
}

func TestNewDiscordClient_MissingChannelID(t *testing.T) {
	t.Setenv("DISCORD_BOT_TOKEN", "tok")
	t.Setenv("DISCORD_CHANNEL_ID", "")

	_, err := newDiscordClient()
	if err == nil {
		t.Error("expected error when channel ID missing")
	}
}

func TestNewDiscordClient_BothSet(t *testing.T) {
	t.Setenv("DISCORD_BOT_TOKEN", "mytoken")
	t.Setenv("DISCORD_CHANNEL_ID", "mychannel")

	c, err := newDiscordClient()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.channelID != "mychannel" {
		t.Errorf("channelID = %q, want %q", c.channelID, "mychannel")
	}
}

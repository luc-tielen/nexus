package main

import (
	"fmt"
	"os"

	"github.com/bwmarrin/discordgo"
)

// discordSender is the subset of discordgo.Session used for sending messages.
// Extracted as an interface to allow test doubles.
type discordSender interface {
	ChannelMessageSend(channelID, content string, options ...discordgo.RequestOption) (*discordgo.Message, error)
}

// discordClient holds a Discord session and the target channel.
type discordClient struct {
	session   discordSender
	channelID string
}

// newDiscordClient reads DISCORD_BOT_TOKEN and DISCORD_CHANNEL_ID from the
// environment and returns a ready client. Returns an error if either variable
// is unset or if the session cannot be created.
func newDiscordClient() (*discordClient, error) {
	token := os.Getenv("DISCORD_BOT_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("DISCORD_BOT_TOKEN is not set")
	}
	channelID := os.Getenv("DISCORD_CHANNEL_ID")
	if channelID == "" {
		return nil, fmt.Errorf("DISCORD_CHANNEL_ID is not set")
	}

	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		return nil, fmt.Errorf("creating Discord session: %w", err)
	}

	return &discordClient{session: dg, channelID: channelID}, nil
}

// Send posts content to the configured Discord channel.
func (c *discordClient) Send(content string) error {
	_, err := c.session.ChannelMessageSend(c.channelID, content)
	return err
}

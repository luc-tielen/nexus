package discord

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
)

// Sender is the subset of discordgo.Session used for sending messages.
// Extracted as an interface to allow test doubles.
type Sender interface {
	ChannelMessageSend(channelID, content string, options ...discordgo.RequestOption) (*discordgo.Message, error)
}

// Client holds a Discord session and the target channel.
type Client struct {
	Session   Sender
	ChannelID string
}

// NewClient returns a ready client for the given token and channelID.
// Returns an error if either value is empty or if the session cannot be created.
func NewClient(token, channelID string) (*Client, error) {
	if token == "" {
		return nil, fmt.Errorf("DISCORD_BOT_TOKEN is not set")
	}
	if channelID == "" {
		return nil, fmt.Errorf("DISCORD_CHANNEL_ID is not set")
	}

	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		return nil, fmt.Errorf("creating Discord session: %w", err)
	}

	return &Client{Session: dg, ChannelID: channelID}, nil
}

// Send posts content to the configured Discord channel.
func (c *Client) Send(content string) error {
	_, err := c.Session.ChannelMessageSend(c.ChannelID, content)
	return err
}
